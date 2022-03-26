package util

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"backup/pkg/logger"
)

// GetSubDir 递归获取目录下所有子目录
func GetSubDir(ctx context.Context, dirname string) ([]string, error) {
	var result = []string{dirname}
	logger.Logger.WithContext(ctx).WithField("dirname", dirname).Info("begin get subdir")
	dir, err := os.Open(dirname)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).WithField("path", dirname).Error("read dir fail")
		return nil, err
	}
	stat, err := dir.Stat()
	if !stat.IsDir() {
		logger.Logger.WithContext(ctx).WithField("dirname", dirname).Info("dirname is not directory")
		return result, err
	}

	subDirs, err := dir.ReadDir(0)
	dir.Close()
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).WithField("path", dirname).Error("read dirname fail")
		return nil, err
	}

	for _, subDir := range subDirs {
		if !subDir.IsDir() {
			continue
		}
		subSubDirs, err := GetSubDir(ctx, filepath.Join(dirname, subDir.Name()))
		if err != nil {
			logger.Logger.WithContext(ctx).WithError(err).WithField("path", subDir).Error("get subdir fail")
			return result, err
		}
		result = append(result, subSubDirs...)
	}

	logger.Logger.WithContext(ctx).WithField("path", dirname).WithField("subdir", result).Info("end get subdir")
	return result, nil
}

// GetSubDirV2 递归获取目录下所有子目录
func GetSubDirV2(ctx context.Context, dirname string) ([]string, error) {
	var result = []string{dirname}
	logger.Logger.WithContext(ctx).WithField("dirname", dirname).Info("begin get subdir")

	err := filepath.WalkDir(dirname, func(path string, d fs.DirEntry, err error) error {
		if StringInSlice(path, result) {
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		subDirs, err := GetSubDirV2(ctx, path)
		result = append(result, subDirs...)
		return nil
	})

	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("walkdir fail")
		return nil, err
	}

	logger.Logger.WithContext(ctx).WithField("path", dirname).WithField("subdir", result).Info("end get subdir")
	return result, nil
}
