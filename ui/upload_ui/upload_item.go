package upload_ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"backup/consts"
	"backup/internal/dao"
	"backup/pkg/database"
	"backup/pkg/logger"
	"backup/pkg/pcs_client"
	"backup/pkg/util"
)

type UploadItem struct {
	path       string
	ctx        context.Context
	cancelFunc context.CancelFunc
	state      int
	progress   string
	serverPath string

	list *UploadList
}

func NewUploadItem(path, serverPath string, list *UploadList) *UploadItem {
	baseCtx := util.NewContext() // 基础ctx，用于生成trace_id
	ctx, cancelFunc := context.WithCancel(baseCtx)
	return &UploadItem{
		path:       filepath.Clean(path),
		ctx:        ctx,
		cancelFunc: cancelFunc,
		state:      consts.UploadStatusWaitUploaded,
		progress:   "等待上传",
		serverPath: serverPath,
		list:       list,
	}
}

func (i *UploadItem) Cancel() {
	i.UploadStatus(consts.UploadStatusCancel)
	i.cancelFunc()
}

func (i *UploadItem) UploadStatus(status int) {
	i.state = status
	i.progress = consts.UploadTextMap[status]
}

func (i *UploadItem) Upload() {
	baseLogger := logger.Logger.WithContext(i.ctx)

	stat, err := os.Stat(i.path)
	if err != nil {
		baseLogger.WithField("path", i.path).WithError(err).Error("get file stat fail")
		i.progress = consts.UploadFailText
		i.list.Refresh()
		return
	}
	fileInfoDao := dao.NewFileInfoDao(i.ctx, database.DB)
	// 这里是兼容空文件的情况，如果是一个空文件，至少会上传一个空的分块，最后create也会有一个signal，总共两个signal
	total := (stat.Size()+consts.Size4MB-1)/consts.Size4MB + 1
	if total < 2 {
		total = 2
	}
	var current int64 = 0
	signal := make(chan struct{})
	go func() {
		defer close(signal)
		err := pcs_client.UploadFileWithSignal(i.ctx, i.path, i.serverPath, signal)
		if err != nil {
			baseLogger.WithField("upload_item", i).WithError(err).Error("upload file fail")
			err = fileInfoDao.Update(map[string]interface{}{
				"upload_status": consts.UploadStatusFail,
			}, i.path)
			if err != nil {
				baseLogger.WithField("status", consts.UploadStatusFail).Warn("upload file info status fail")
			}
			i.UploadStatus(consts.UploadStatusFail)
			i.list.release(i) // list从上传列表中移除item
			return
		}
		// 上传完成，更新上传状态
		baseLogger.WithField("item", i).Info("upload item upload success")
		err = fileInfoDao.Update(map[string]interface{}{
			"upload_status": consts.UploadStatusUploaded,
		}, i.path)
		if err != nil {
			baseLogger.WithField("status", consts.UploadStatusUploaded).Warn("upload file info status fail")
		}
		i.UploadStatus(consts.UploadStatusUploaded)
		i.list.release(i) // list从上传列表中移除item
	}()

	i.UploadStatus(consts.UploadStatusUploading)
	err = fileInfoDao.Update(map[string]interface{}{
		"upload_status": consts.UploadStatusUploading,
	}, i.path)
	if err != nil {
		baseLogger.WithField("status", consts.UploadStatusUploading).Warn("upload file info status fail")
	}
	for {
		select {
		case _, ok := <-signal:
			if !ok {
				return
			}
			current++
			if current > total {
				i.cancelFunc()
				baseLogger.WithField("upload_item", i).Error("upload fail, too many signal")
				// 更改数据库状态为上传失败
				err = fileInfoDao.Update(map[string]interface{}{
					"upload_status": consts.UploadStatusFail,
				}, i.path)
				if err != nil {
					baseLogger.WithField("status", consts.UploadStatusFail).Warn("upload file info status fail")
				}
				i.UploadStatus(consts.UploadStatusFail) // 更新状态为上传失败
				i.list.release(i)                       // list从上传列表中移除item
				return
			}
			i.progress = fmt.Sprintf("%.2f%%", float64(current*100)/float64(total))
			if current == total { // 补充更新，否则可能出现覆盖
				i.UploadStatus(consts.UploadStatusUploaded)
			}
		}
	}
}

func (i *UploadItem) WithContext(ctx context.Context) {
	i.ctx, i.cancelFunc = context.WithCancel(ctx)
}
