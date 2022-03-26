package scanner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"gorm.io/gorm"

	"backup/consts"
	"backup/internal/dao"
	"backup/internal/model"
	"backup/pkg/database"
	"backup/pkg/logger"
	"backup/pkg/util"
	"backup/ui/upload_ui"
)

var semaphore = make(chan struct{}, 200)

type Scanner struct {
	ctx           context.Context    // 上下文
	root          string             // 扫描的根路径，可能是目录，也可能是文件
	excludePrefix string             // 上传文件时需要排除的
	isDir         bool               // root是否是目录
	cancelFunc    context.CancelFunc // 取消上下文的函数
}

func NewScanner(ctx context.Context, root string) (*Scanner, error) {
	baseLogger := logger.Logger.WithContext(ctx)

	root, err := filepath.Abs(root)
	if err != nil {
		baseLogger.WithError(err).Error("get file absolute path fail")
		return nil, err
	}

	stat, err := os.Stat(root)
	if err != nil {
		baseLogger.WithError(err).Error("get file stat fail")
		return nil, err
	}

	newCtx, cancelFunc := context.WithCancel(ctx)
	// 处理根目录，得到server前缀
	// 如果root是文件夹，则prefix表示文件在上传时候的统一名称
	return &Scanner{
		root:          root,
		ctx:           newCtx,
		excludePrefix: filepath.Dir(filepath.Dir(root + "/")),
		isDir:         stat.IsDir(),
		cancelFunc:    cancelFunc,
	}, nil
}

func (s *Scanner) WithExcludePrefix(excludePrefix string) *Scanner {
	s.excludePrefix = excludePrefix
	return s
}

func (s *Scanner) Cancel() {
	s.cancelFunc()
}

// ScanAndUpload 扫描并上传
func (s *Scanner) ScanAndUpload() {
	scanAndUpload(s.ctx, s.root, s.excludePrefix, upload_ui.ExportUploadList) // 扫描并上传
}

// 扫描入库
func (s *Scanner) Scan() {
	fileInfoDao := dao.NewFileInfoDao(s.ctx, database.DB)
	if !s.isDir {
		fileInfo := model.NewFileInfo(s.root, s.excludePrefix)
		if fileInfo == nil {
			logger.Logger.WithField("path", s.root).Error("generate fileInfo fail")
			return
		}
		fileInfoDao.Add(fileInfo)
		return
	}

	s.scan(s.root, fileInfoDao)
}

func (s *Scanner) scan(dirname string, fileInfoDao *dao.FileInfoDao) {
	logger.Logger.WithField("dirname", dirname).Info("begin get subdir")

	err := filepath.WalkDir(dirname, func(path string, d fs.DirEntry, err error) error {
		if dirname == path {
			return nil
		}
		if !d.IsDir() {
			info := model.NewFileInfo(path, s.excludePrefix)
			if info != nil {
				fileInfoDao.Add(info)
			}
			return nil
		}

		s.scan(path, fileInfoDao)
		return nil
	})

	if err != nil {
		logger.Logger.WithField("path", dirname).WithError(err).Error("walkdir fail")
		return
	}

	logger.Logger.WithField("path", dirname).Info("end get subdir")
}

func scanAndUpload(ctx context.Context, root, excludePrefix string, list *upload_ui.UploadList) {
	baseLogger := logger.Logger.WithContext(ctx)

	fileInfoDao := dao.NewFileInfoDao(ctx, database.DB)
	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		select {
		case <-ctx.Done(): // 监听取消
			baseLogger.WithField("root", root).WithField("path", path).Info("scan cancel")
			return nil
		case semaphore <- struct{}{}: // 控制住并发数量
		}

		if err != nil {
			<-semaphore
			baseLogger.WithField("root", root).WithField("path", path).WithError(err).Errorf("walk fail")
			return nil
		}

		if path == root && info.IsDir() {
			<-semaphore
			return nil
		}

		// 如果是文件夹，递归扫描上传
		if info.IsDir() {
			<-semaphore // 防止嵌套太深的情况下出现死锁
			scanAndUpload(ctx, path, excludePrefix, upload_ui.ExportUploadList)
			return nil
		}

		defer func() {
			<-semaphore
		}()
		path = filepath.Clean(path) // 路径规范
		// 计算MD5值
		md5, err := util.GetFileMd5(ctx, path)
		if err != nil {
			baseLogger.WithField("path", path).WithError(err).Error("generate file md5 fail")
		}

		fileInfo, err := fileInfoDao.QueryByAbsPath(path)
		if err != nil && err != gorm.ErrRecordNotFound { // 查找出错了，当错没有查到
			baseLogger.WithField("path", path).WithError(err).Error("query file info fail")
			fileInfo = model.NewFileInfo(path, excludePrefix)
			err := fileInfoDao.Add(fileInfo)
			if err != nil {
				baseLogger.WithField("path", path).WithError(err).Error("add file info fail")
				return nil
			}
		} else if err == gorm.ErrRecordNotFound { // 如果没有查到
			fileInfo = model.NewFileInfo(path, excludePrefix)
			err := fileInfoDao.Add(fileInfo)
			if err != nil {
				baseLogger.WithField("path", path).WithError(err).Error("add file info fail")
				return nil
			}
		}

		if md5 == "" || err == gorm.ErrRecordNotFound {
			item := upload_ui.NewUploadItem(path, util.GenerateServerFile(path, excludePrefix), list)
			list.AddItem(ctx, item)
		} else if md5 != fileInfo.Md5 || (fileInfo.UploadStatus != consts.UploadStatusUploaded && fileInfo.UploadStatus != consts.UploadStatusUploading && fileInfo.UploadStatus != consts.UploadStatusWaitUploaded) { // 如果不相等，或者状态为未上传
			item := upload_ui.NewUploadItem(path, util.GenerateServerFile(path, excludePrefix), list)
			list.AddItem(ctx, item)
			err := fileInfoDao.Update(map[string]interface{}{
				"md5":  md5,
				"size": util.GetFileSize(ctx, path),
			}, fileInfo.AbsPath)

			if err != nil {
				baseLogger.WithField("path", path).WithError(err).Error("upload item md5 fail")
			}
		}

		return nil
	})

	if err != nil {
		baseLogger.WithField("root", root).WithError(err).Errorf("walk fail")
	}
}
