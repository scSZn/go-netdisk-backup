package upload_ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"backup/consts"
	"backup/pkg/logger"
	"backup/pkg/util"
	ui_util "backup/ui/util"
)

type UploadList struct {
	widget.List

	lock  sync.RWMutex
	items []*UploadItem

	window         fyne.Window
	uploadingItems []*UploadItem
	uploadingQueue chan struct{}
	waitQueue      chan *UploadItem

	signal chan *UploadItem
}

func NewUploadList(window fyne.Window) *UploadList {
	list := &UploadList{
		items:          []*UploadItem{},
		window:         window,
		waitQueue:      make(chan *UploadItem, 100), // 等待队列
		uploadingItems: make([]*UploadItem, 0, 5),   // 最大同时上传5个文件
		uploadingQueue: make(chan struct{}, 5),      // 最大同时上传5个文件
	}

	list.List.CreateItem = list.CreateItem
	list.List.Length = list.Length
	list.List.UpdateItem = list.UpdateItem

	list.ExtendBaseWidget(list)
	go list.upload()  // 启动协程开始上传任务
	go list.refresh() // 启动协程定时刷新
	return list
}

func (l *UploadList) Length() int {
	return len(l.items)
}

func (l *UploadList) CreateItem() fyne.CanvasObject {
	cancelBtn := &widget.Button{
		Icon: theme.CancelIcon(),
	}
	retryBtn := &widget.Button{
		Icon: theme.ViewRefreshIcon(),
	}
	progress := &widget.Label{
		Text: "",
	}
	return container.NewHBox(widget.NewLabel(""), layout.NewSpacer(), progress, retryBtn, cancelBtn)
}

func (l *UploadList) UpdateItem(id widget.ListItemID, canvas fyne.CanvasObject) {
	c := canvas.(*fyne.Container)
	item := l.items[id]
	c.Objects[0].(*widget.Label).SetText(item.path)

	c.Objects[2].(*widget.Label).Bind(binding.BindString(&item.progress))
	retryBtn := c.Objects[3].(*widget.Button)
	retryBtn.Hide()
	if item.state == consts.UploadStatusFail {
		retryBtn.Show()
		retryBtn.OnTapped = func() { // 点击重试按钮
			ctx := util.NewContext()
			item.WithContext(ctx)                              // 更新上下文
			item.UploadStatus(consts.UploadStatusWaitUploaded) //更新进度和状态
			select {
			case l.waitQueue <- item:
			case <-time.NewTimer(5 * time.Second).C:
				ui_util.ShowErrorDialog("重试失败", l.window)
			}
		}
	}
	c.Objects[4].(*widget.Button).OnTapped = func() {
		item.Cancel()
		l.removeItem(item)
	}
}

func (l *UploadList) removeItem(item *UploadItem) {
	// 加锁保证线程安全
	l.lock.Lock()
	defer l.lock.Unlock()
	// 先找到在总的列表中的index
	var index = -1
	for i, v := range l.items {
		if v == item {
			index = i
			break
		}
	}
	// 移除指定的item
	if index > 0 {
		l.items = append(l.items[:index], l.items[index+1:]...)
	}
}

// 清理指定状态的item
func (l *UploadList) CleanItem(state int) {
	l.lock.Lock()
	defer l.lock.Unlock()

	newItems := make([]*UploadItem, 0, len(l.items))
	for _, item := range l.items {
		if item.state == state {
			continue
		}
		newItems = append(newItems, item)
	}

	l.items = newItems
}

// 清理指定状态的item
func (l *UploadList) ClearItemPrefix(prefix string) {
	l.lock.Lock()
	defer l.lock.Unlock()

	newItems := make([]*UploadItem, 0, len(l.items))
	for _, item := range l.items {
		// 找到所有具有这个前缀的item，取消其上下文，同时从列表项中删除
		if strings.HasPrefix(filepath.Clean(item.path), filepath.Clean(prefix)) {
			item.Cancel()
			continue
		}
		newItems = append(newItems, item)
	}

	l.items = newItems
}

func (l *UploadList) RetryAll() *context.CancelFunc {
	ctx, cancelFunc := context.WithCancel(util.NewContext())
	go func() {
		defer cancelFunc()
		for _, item := range l.items {
			if item.state != consts.UploadStatusFail {
				continue
			}
			item.WithContext(util.NewContext())                // 更新上下文
			item.UploadStatus(consts.UploadStatusWaitUploaded) //更新进度和状态
			select {
			case l.waitQueue <- item:
			case <-ctx.Done():
				return
			}
		}
	}()
	return &cancelFunc
}

// 释放item
// 1. 从上传列表中移除
// 2. 信号量+1
func (l *UploadList) release(item *UploadItem) {
	<-l.uploadingQueue // 空出队列
	l.lock.Lock()
	defer l.lock.Unlock()

	// 将item从uploadingItems中移除
	var removeIndex = -1
	for i, queueItem := range l.uploadingItems {
		if item.path == queueItem.path {
			removeIndex = i
			break
		}
	}
	if removeIndex >= 0 {
		l.uploadingItems = append(l.uploadingItems[:removeIndex], l.uploadingItems[removeIndex+1:]...)
		return
	}
}

func (l *UploadList) AddItem(ctx context.Context, item *UploadItem) {
	l.lock.Lock()
	// 去重
	var exists = false
	for _, i := range l.items {
		if i.path == item.path {
			if i.state == consts.UploadStatusWaitUploaded || i.state == consts.UploadStatusUploading { // 判断状态是否为等待上传
				exists = true
			}
			break
		}
	}
	l.lock.Unlock() // 不使用defer，尽可能减少锁住的时间

	if !exists {
		select {
		case l.waitQueue <- item: // 添加item到等待队列中
			l.lock.Lock()
			l.items = append(l.items, item) // 添加item
			l.lock.Unlock()
		case <-ctx.Done(): // 取消上传
			logger.Logger.WithContext(ctx).WithField("item", item).Info("cancel add item")
			return
		}
	}
}

func (l *UploadList) upload() {
	// 从waitQueue中获取item，添加到uploadingQueue队列中
	for {
		select {
		// 从等待队列中获取上传队列的任务
		case item := <-l.waitQueue:
			// 如果item的状态已经不是待上传了，掠过
			if item.state == consts.UploadStatusWaitUploaded {
				l.uploadingQueue <- struct{}{}                    // 控制上传个数
				l.uploadingItems = append(l.uploadingItems, item) // 存储上传的item
				fmt.Println(item)
				go item.Upload() // 开始上传
			}
		}
	}
}

func (l *UploadList) refresh() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		// 定时刷新
		case <-ticker.C:
			l.Refresh()
		}
	}
}
