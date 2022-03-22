package scan

import (
	"context"
	"os"
	"path/filepath"

	"backup/consts"
	"backup/pkg/group"
	"backup/pkg/logger"
	"backup/pkg/pcs_client"
	"backup/pkg/util"
)

type Scanner struct {
	ctx           context.Context // 上下文
	root          string          // 扫描的根路径，可能是目录，也可能是文件
	excludePrefix string          // 上传文件时需要排除的
	isDir         bool            // root是否是目录
}

func NewScanner(ctx context.Context, root string) (*Scanner, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("get file absolute path fail")
		return nil, err
	}

	stat, err := os.Stat(root)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("get file stat fail")
		return nil, err
	}

	// 处理根目录，得到server前缀
	// 如果root是文件夹，则prefix表示文件在上传时候的统一名称
	return &Scanner{
		root:          root,
		ctx:           ctx,
		excludePrefix: filepath.Dir(filepath.Dir(root + "/")),
		isDir:         stat.IsDir(),
	}, nil
}

func (s *Scanner) WithExcludePrefix(excludePrefix string) *Scanner {
	s.excludePrefix = excludePrefix
	return s
}

func (s *Scanner) Scan() {
	if !s.isDir {
		err := pcs_client.UploadFileWithRetry(s.ctx, s.root, util.GenerateServerFile(s.root, s.excludePrefix), consts.MaxRetryCount)
		if err != nil {
			logger.Logger.WithContext(s.ctx).WithError(err).Error("send fail")
		}
		return
	}

	sendGroup := group.NewMyGroup(s.ctx, 5, 5).WithErrorStrategy(group.AbortStrategy{})
	sendGroup.Start()
	pcs_client.UploadDirectory(s.ctx, s.root, s.excludePrefix, sendGroup)

	if err := sendGroup.Wait(); err != nil {
		logger.Logger.WithContext(s.ctx).WithError(err).Error("send fail")
	}
}
