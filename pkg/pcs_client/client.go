package pcs_client

import (
	"context"
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	"backup/consts"
	"backup/internal/config"
	"backup/internal/token"
	"backup/pkg/logger"
	"backup/pkg/work_pool"
)

var p = work_pool.NewWorkPool(context.Background(), 20, 10, work_pool.WorkModeSlowStart)

func init() {
	p.Start()
}

type UploadParams struct {
	filename     string // 需要上传的本地文件
	serverPath   string // 上传后在百度网盘的路径
	refreshFunc  func() // 上传完一个分片后的刷新函数
	completeFunc func() // 上传完成后的回调函数
}

func NewUploadParams(filename string, serverPath string, refreshFunc func(), completeFunc func()) *UploadParams {
	return &UploadParams{filename: filename, serverPath: serverPath, refreshFunc: refreshFunc, completeFunc: completeFunc}
}

func Upload(ctx context.Context, params *UploadParams) error {
	baseLogger := logger.Logger.WithContext(ctx)
	serverPath := path.Join(config.Config.PcsConfig.PathPrefix, params.serverPath)
	serverPath = filepath.Clean(serverPath)

	baseLogger.WithFields(map[string]interface{}{
		"filename":        params.filename,
		"server_filename": serverPath,
	}).Info("upload start")

	preCreateReq, err := NewPreCreateRequest(ctx, params.filename, serverPath)
	if err != nil {
		return errors.Wrap(err, "construct precreateRequest fail")
	}

retry:
	preCreateResp, err := pcsPreCreate(ctx, preCreateReq)
	if err != nil {
		baseLogger.WithError(err).Error("precreate fail")
		if preCreateResp != nil && preCreateResp.Errno == consts.ErrnoAccessTokenInvalid { // 如果是token失效
			baseLogger.Error("access_token is expired")
			err = token.RefreshTokenFromServerByRefreshCode()
			if err != nil {
				baseLogger.WithError(err).Error("refresh access_token fail")
				return err
			}
			goto retry
		} else {
			return err
		}
	}

	uploadReq := NewUploadRequest(preCreateResp.UploadId, serverPath, preCreateResp.BlockList, params.filename, params.refreshFunc)
	err = pcsUpload(ctx, uploadReq)

	if err != nil {
		baseLogger.WithError(err).Error("pcs upload fail")
		return err
	}

	createParams := &createRequest{
		Path:       serverPath,
		Size:       preCreateReq.Size,
		IsDir:      preCreateReq.IsDir,
		BlockList:  preCreateReq.BlockList,
		UploadId:   preCreateResp.UploadId,
		Mode:       consts.ModeManual,
		IsRevision: consts.EnableMultiVersion,
	}

	_, err = pcsCreate(ctx, createParams)
	if err != nil {
		baseLogger.WithError(err).Errorf("create fail")
		return err
	}

	if params.completeFunc != nil {
		params.completeFunc()
	}
	baseLogger.Info("upload success")
	return nil
}
