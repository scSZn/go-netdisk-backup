package util

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"strings"

	"backup/consts"
	"backup/pkg/logger"
)

func GetBlockList(ctx context.Context, filename string) ([]string, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		logger.Logger.WithContext(ctx).WithContext(ctx).Errorf("open file fail")
		return nil, err
	}

	var result []string
	chunk := make([]byte, consts.Size4MB) // 4MB的缓存空间
	for {
		n, err := file.Read(chunk)
		if err == io.EOF || n == 0 {
			break
		}

		block, err := Md5(ctx, chunk[:n])
		if err != nil {
			logger.Logger.WithContext(ctx).WithContext(ctx).Errorf("md5 encode fail")
			return nil, err
		}

		result = append(result, block)
	}

	return result, nil
}

func Md5(ctx context.Context, data []byte) (string, error) {
	hash := md5.New()
	_, err := hash.Write(data)
	if err != nil {
		logger.Logger.WithError(err).WithContext(ctx).Errorf("md5 write fail")
		return "", err
	}

	return strings.ToLower(hex.EncodeToString(hash.Sum(nil))), nil
}

func GetFileMd5(ctx context.Context, filename string) (string, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("open file fail")
		return "", err
	}

	hash := md5.New()

	chunk := make([]byte, 1024*1024)
	for {
		n, err := file.Read(chunk)
		if err != nil && err != io.EOF {
			logger.Logger.WithContext(ctx).WithError(err).Error("open file fail")
			return "", err
		}

		if err == io.EOF || n < 1024*1024 {
			break
		}

		hash.Write(chunk[:n])
	}

	return strings.ToLower(hex.EncodeToString(hash.Sum(nil))), nil
}
