package watch

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"

	"backup/consts"
	"backup/internal/scan"
	"backup/pkg/logger"
	"backup/pkg/pcs_client"
	"backup/pkg/util"
)

type Watcher struct {
	root          string             // 被监听文件的根路径，可能是文件夹，也可能是文件
	watcher       *fsnotify.Watcher  // 监听器
	ctx           context.Context    // 上下文，默认会加一个cancel类型的上下文
	cancelFunc    context.CancelFunc // 取消上下文函数
	excludePrefix string             // 去除掉的前缀，如果root是文件夹类型时有效
	isDir         bool               // 监听的是否为文件夹
}

// NewWatcher 初始化监听
func NewWatcher(ctx context.Context, root string) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).WithField("dirname", root).Error("new watcher fail")
		log.Fatal(err)
	}

	newCtx, cancel := context.WithCancel(ctx)
	stat, err := os.Stat(root)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("get file stat fail")
		cancel()
		return nil, err
	}

	root, err = filepath.Abs(root)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("get file absolute path fail")
		cancel()
		return nil, err
	}
	// 处理根目录，得到server前缀
	// 如果root是文件夹，则prefix表示文件在上传时候的统一名称
	return &Watcher{
		watcher:       watcher,
		root:          root,
		ctx:           newCtx,
		cancelFunc:    cancel,
		excludePrefix: filepath.Dir(filepath.Dir(root + "/")),
		isDir:         stat.IsDir(),
	}, nil
}

func (w *Watcher) Start() error {
	err := Watch(w.ctx, w.watcher, w.root)
	if err != nil {
		logger.Logger.WithContext(w.ctx).WithError(err).WithField("root", w.root).Error("add watch fail")
		return err
	}
	go func() {
		defer w.watcher.Close()
		for {
			select {
			case event, ok := <-w.watcher.Events:
				logger.Logger.Infof("receive event: %+v", event)
				if !ok {
					continue
				}
				w.eventDeal(event)
			case err, ok := <-w.watcher.Errors: // 事件监听出错
				if !ok {
					continue
				}
				logger.Logger.WithContext(w.ctx).WithError(err).Error("watch encounter error")
			case <-w.ctx.Done(): // 调用了Cancel函数
				return
			}
		}
	}()

	logger.Logger.WithContext(w.ctx).WithField("root", w.root).Info("watch started")
	return nil
}

// Cancel 停止监听
func (w *Watcher) Cancel() {
	w.cancelFunc()
}

// eventDeal 处理事件，分为两类事件
// 1. 文件夹
//   * Create：创建文件夹事件(如果是修改文件名，也会有一个新文件名的Create事件)，需要添加监听事件
//   * Delete：删除文件夹，不做操作
// 2. 文件
//   * Create：新文件，触发上传
//   * Write：文件被修改，触发上传
//   * Delete：删除文件，不做操作
// 注意，这里的文件名称都是绝对路径
func (w *Watcher) eventDeal(event fsnotify.Event) error {
	logger.Logger.WithContext(w.ctx).WithField("event", event).Info("begin deal event")
	stat, err := os.Stat(event.Name)
	if err != nil {
		logger.Logger.WithContext(w.ctx).WithError(err).WithField("filename", event.Name).Error("open file fail")
		return err
	}
	switch {
	case event.Op&fsnotify.Create > 0:
		if stat.IsDir() {
			err = w.eventDirCreate(event.Name)
			if err != nil {
				logger.Logger.WithContext(w.ctx).WithError(err).WithField("directory", event.Name).Error("deal directory create event fail")
				return err
			}
		} else {
			err = w.eventFileCreate(event.Name)
			if err != nil {
				logger.Logger.WithContext(w.ctx).WithError(err).WithField("directory", event.Name).Error("deal file create event fail")
				return err
			}
		}
	case event.Op&fsnotify.Write > 0:
		if !stat.IsDir() {
			err = w.eventFileWrite(event.Name)
			if err != nil {
				logger.Logger.WithContext(w.ctx).WithError(err).WithField("directory", event.Name).Error("deal file create event fail")
				return err
			}
		}
	}
	logger.Logger.WithContext(w.ctx).WithField("event", event).Info("end deal event")
	return nil
}

// eventDirCreate 创建文件夹事件
func (w *Watcher) eventDirCreate(dirname string) error {
	logger.Logger.WithContext(w.ctx).WithField("dirname", dirname).Info("begin deal create directory event")
	err := Watch(w.ctx, w.watcher, dirname)
	if err != nil {
		logger.Logger.WithContext(w.ctx).WithError(err).WithField("dirname", dirname).Error("add watch fail")
		return err
	}
	// 扫描文件夹，上传文件
	scanner, err := scan.NewScanner(w.ctx, dirname)
	if err != nil {
		logger.Logger.WithContext(w.ctx).WithError(err).WithField("dirname", dirname).Error("create scanner fail")
	}
	scanner.WithExcludePrefix(w.excludePrefix)
	go scanner.Scan()

	logger.Logger.WithContext(w.ctx).WithField("dirname", dirname).Info("end deal create directory event")
	return nil
}

// eventFileCreate 创建文件事件
func (w *Watcher) eventFileCreate(filename string) error {
	logger.Logger.WithContext(w.ctx).WithField("filename", filename).Info("begin deal create file event")
	serverFilename := filepath.Base(filename)
	// 如果是文件夹，则需要加上前缀
	if w.isDir {
		serverFilename = filename[len(w.excludePrefix):]
	}
	err := pcs_client.UploadFileWithRetry(w.ctx, filename, serverFilename, consts.MaxRetryCount)
	if err != nil {
		logger.Logger.WithContext(w.ctx).WithError(err).WithField("filename", filename).Error("fail to deal create file event, upload fail")
		return err
	}
	logger.Logger.WithContext(w.ctx).WithField("filename", filename).WithField("serverFilename", serverFilename).Info("end deal create file event")
	return nil
}

// eventFileWrite 修改文件事件
func (w *Watcher) eventFileWrite(filename string) error {
	logger.Logger.WithContext(w.ctx).WithField("filename", filename).Info("begin deal write file event")
	serverFilename := filepath.Base(filename)
	// 如果是文件夹，则需要加上前缀
	if w.isDir {
		serverFilename = filename[len(w.excludePrefix):]
	}
	err := pcs_client.UploadFileWithRetry(w.ctx, filename, serverFilename, consts.MaxRetryCount)
	if err != nil {
		logger.Logger.WithContext(w.ctx).WithError(err).WithField("filename", filename).Error("fail to deal create file event, upload fail")
		return err
	}
	logger.Logger.WithContext(w.ctx).WithField("filename", filename).WithField("serverFilename", serverFilename).Info("end deal write file event")
	return nil
}

// Watch 对dirname目录及其所有子目录进行监控，
func Watch(ctx context.Context, watcher *fsnotify.Watcher, dirname string) (err error) {
	defer func() {
		if err != nil {
			err := watcher.Close()
			if err != nil {
				logger.Logger.WithContext(ctx).WithError(err).WithField("dirname", dirname).Error("close watch fail")
			}
		}
	}()
	dirs, err := util.GetSubDir(ctx, dirname)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).WithField("dirname", dirname).Error("get subdir fail")
		return err
	}

	for _, dir := range dirs {
		err = watcher.Add(dir)
		if err != nil {
			logger.Logger.WithContext(ctx).WithError(err).WithField("dirname", dirname).Error("watch fail")
			return err
		}
	}

	return nil
}
