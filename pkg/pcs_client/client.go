package pcs_client

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	"backup/consts"
	"backup/internal/config"
	"backup/internal/dao"
	"backup/internal/model"
	"backup/internal/token"
	"backup/pkg/database"
	"backup/pkg/group"
	"backup/pkg/logger"
	"backup/pkg/util"
)

func UploadFile(ctx context.Context, filename string, serverFilename string) error {
	if serverFilename == "" {
		logger.Logger.WithContext(ctx).WithField("filename", filename).Warn("serverFilename is empty")
		return errors.Errorf("serverFilename is empty, filename is %s", filename)
	}
	list, err := util.GetBlockList(ctx, filename)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("get block list fail")
		return err
	}

	serverPath := path.Join(config.Config.PcsConfig.PathPrefix, serverFilename)

	stat, err := os.Stat(filename)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("get file stat fail")
		return err
	}

	var isDir uint8 = 0
	if stat.IsDir() {
		isDir = 1
	}

	preCreateParams := &PreCreateParams{
		Path:      serverPath,
		Size:      stat.Size(),
		IsDir:     isDir,
		BlockList: list,
		RType:     consts.RTypeOverride,
	}

	preCreateResponse, err := pcsPreCreate(ctx, preCreateParams)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("precreate fail")
		return err
	}

	if preCreateResponse.Errno == consts.ErrnoAccessTokenInvalid {
		logger.Logger.WithContext(ctx).Error("access_token is expired")
		err = token.RefreshTokenFromServerByRefreshCode()
		if err != nil {
			logger.Logger.WithContext(ctx).WithError(err).Error("refresh access_token fail")
			return err
		}
		return errors.New("access_token is expired")
	}

	if preCreateResponse.Errno != consts.ErrnoSuccess {
		logger.Logger.WithContext(ctx).Errorf("precreate response errno isn't 0, but get %+v", preCreateResponse.Errno)
		return err
	}

	err = Upload(ctx, preCreateResponse.UploadId, serverPath, preCreateResponse.BlockList, filename)

	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("upload fail")
		return err
	}

	createParams := &CreateParams{
		Path:       serverPath,
		Size:       stat.Size(),
		IsDir:      isDir,
		BlockList:  list,
		UploadId:   preCreateResponse.UploadId,
		Mode:       consts.ModeManual,
		IsRevision: consts.EnableMultiVersion,
	}
	createResponse, err := pcsCreate(ctx, createParams)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("create fail")
		return err
	}

	if createResponse.Errno != consts.ErrnoSuccess {
		logger.Logger.WithContext(ctx).WithField("createResponse", createResponse).Error("create fail")
		return fmt.Errorf("create fail")
	}

	return nil
}

func UploadFileWithSignal(ctx context.Context, filename string, serverPath string, signal chan struct{}) error {
	baseLogger := logger.Logger.WithContext(ctx)
	serverPath = path.Join(config.Config.PcsConfig.PathPrefix, serverPath)
	serverPath = filepath.Clean(serverPath)

	baseLogger.WithFields(map[string]interface{}{
		"filename":        filename,
		"server_filename": serverPath,
	}).Info("upload start")

	if serverPath == "" {
		baseLogger.WithField("filename", filename).Warn("serverPath is empty")
		return errors.Errorf("serverFilename is empty, filename is %s", filename)
	}

	list, err := util.GetBlockList(ctx, filename)
	if err != nil {
		baseLogger.WithError(err).Error("get block list fail")
		return err
	}

	stat, err := os.Stat(filename)
	if err != nil {
		baseLogger.WithError(err).Error("get file stat fail")
		return err
	}

	//if stat.Size() == 0 {
	//	if signal != nil {
	//		signal <- struct{}{}
	//	}
	//	return nil
	//}

	var isDir uint8 = 0
	if stat.IsDir() {
		isDir = 1
	}

	preCreateParams := &PreCreateParams{
		Path:      serverPath,
		Size:      stat.Size(),
		IsDir:     isDir,
		BlockList: list,
		RType:     consts.RTypeOverride,
	}

	preCreateResponse, err := pcsPreCreate(ctx, preCreateParams)
	if err != nil {
		baseLogger.WithError(err).Error("precreate fail")
		if preCreateResponse != nil && preCreateResponse.Errno == consts.ErrnoAccessTokenInvalid { // 如果是token失效
			baseLogger.Error("access_token is expired")
			err = token.RefreshTokenFromServerByRefreshCode()
			if err != nil {
				baseLogger.WithError(err).Error("refresh access_token fail")
				return err
			}
			return errors.New("access_token is expired")
		}
		return err
	}

	err = pcsUploadWithSignal(ctx, preCreateResponse.UploadId, serverPath, preCreateResponse.BlockList, filename, signal)

	if err != nil {
		baseLogger.WithError(err).Error("pcs upload fail")
		return err
	}

	createParams := &CreateParams{
		Path:       serverPath,
		Size:       stat.Size(),
		IsDir:      isDir,
		BlockList:  list,
		UploadId:   preCreateResponse.UploadId,
		Mode:       consts.ModeManual,
		IsRevision: consts.EnableMultiVersion,
	}

	_, err = pcsCreate(ctx, createParams)
	if err != nil {
		baseLogger.WithError(err).Errorf("create fail")
		return err
	}

	if signal != nil {
		signal <- struct{}{}
	}
	baseLogger.Info("upload success")
	return nil
}

func UploadDirectory(ctx context.Context, root string, excludeDirectory string, sendGroup *group.MyGroup) {
	infoDao := dao.NewFileInfoDao(ctx, database.DB)
	// 批量上传文件
	filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			logger.Logger.WithContext(ctx).WithError(err).WithField("path", path).Error("walk path fail")
			return nil
		}
		// 如果path等于根目录，直接返回
		if path == root {
			return nil
		}
		// 如果是目录，递归上传目录下的文件
		if info.IsDir() {
			UploadDirectory(ctx, path, excludeDirectory, sendGroup)
			return nil
		}
		fileInfo, err := infoDao.QueryByAbsPath(path)
		if err != nil {
			logger.Logger.WithContext(ctx).WithError(err).WithField("path", path).Error("query file info fail")
			return nil
		}
		// 如果文件正在上传中，记录日志并返回
		if fileInfo.UploadStatus == consts.UploadStatusUploading {
			logger.Logger.WithContext(ctx).WithField("path", path).Info("file is uploading")
			return nil
		}

		contentMd5, err := util.GetFileMd5(ctx, path)
		if err != nil {
			logger.Logger.WithContext(ctx).WithError(err).WithField("path", path).Error("generate file md5 fail")
			return nil
		}

		// 如果已经上传完成了，校验MD5值，如果相同，不上传
		if fileInfo.UploadStatus == consts.UploadStatusUploaded && fileInfo.Md5 == contentMd5 {
			logger.Logger.WithContext(ctx).WithField("path", path).Info("file is already uploaded")
			return nil
		}

		// 如果没有上传，则添加记录
		err = infoDao.Add(&model.FileInfo{
			AbsPath: path,
			Size:    info.Size(),
			Md5:     contentMd5,
		})
		if err != nil {
			logger.Logger.WithContext(ctx).WithError(err).WithField("path", path).Error("add file info fail")
			return err
		}
		// 尝试上传文件，最大重试3次
		var submitError error
		var count = 0
		for count < 3 {
			submitError = sendGroup.Submit(&group.Task{
				Name: fmt.Sprintf("scan_upload-%s", path),
				RunField: func(ctx context.Context) error {
					// 将状态修改为正在上传
					err := infoDao.Update(map[string]interface{}{
						"upload_status": consts.UploadStatusUploading,
					}, path)
					err = UploadFile(ctx, path, util.GenerateServerFile(path, excludeDirectory))
					// 上传失败，修改状态
					if err != nil {
						logger.Logger.WithContext(ctx).WithError(err).WithField("path", path).Error("upload fail")
						err := infoDao.Update(map[string]interface{}{
							"upload_status": consts.UploadStatusFail,
						}, path)
						if err != nil {
							logger.Logger.WithContext(ctx).WithError(err).WithField("path", path).Error("update status fail")
							return nil
						}
					}
					// 上传成功，修改状态
					err = infoDao.Update(map[string]interface{}{
						"upload_status": consts.UploadStatusUploaded,
					}, path)
					if err != nil {
						logger.Logger.WithContext(ctx).WithError(err).WithField("path", path).Error("update status fail")
						return nil
					}
					return nil
				},
			})
			if submitError == nil {
				break
			}
			count++
		}
		if submitError != nil {
			logger.Logger.WithContext(ctx).WithError(err).WithField("path", path).Error("submit upload task fail")
		}
		return nil
	})
}

//func UploadFileWithRetry(ctx context.Context, filename string, serverFilename string, maxRetry int) error {
//	var count int = 0
//	var err error
//
//	for count < maxRetry {
//		count++
//		err = UploadFile(ctx, filename, serverFilename)
//		if err != nil {
//			logger.Logger.WithContext(ctx).WithError(err).WithField("filename", filename).Error("upload %s fail, retry %d time", filename, count)
//		}
//	}
//
//	return err
//}
