package util

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"strings"

	"backup/consts"
	"backup/pkg/byte_pool"
	"backup/pkg/logger"
)

func GetBlockList(ctx context.Context, filename string) ([]string, error) {
	baseLogger := logger.Logger.WithContext(ctx)
	baseLogger.WithField("filename", filename).Info("generate block list start")

	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		baseLogger.WithError(err).Errorf("open file fail")
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		baseLogger.WithError(err).Errorf("get file stat fail")
		return nil, err
	}

	number := (stat.Size() + consts.Size4MB - 1) / consts.Size4MB
	if number == 0 {
		block, err := Md5(ctx, nil)
		if err != nil {
			baseLogger.WithError(err).Errorf("md5 encode fail")
			return nil, err
		}
		return []string{block}, nil
	}

	var result = make([]string, 0, number)

	chunk := byte_pool.DefaultBytePool.Get()
	for {
		n, err := file.Read(chunk)
		if err != nil && err != io.EOF {
			baseLogger.WithError(err).WithField("filename", filename).Errorf("read file fail fail")
			return nil, err
		}
		if n == 0 || err == io.EOF {
			break
		}

		block, err := Md5(ctx, chunk[:n])
		if err != nil {
			baseLogger.WithError(err).Errorf("md5 encode fail")
			return nil, err
		}

		result = append(result, block)
	}

	byte_pool.DefaultBytePool.Put(chunk)

	baseLogger.WithField("filename", filename).WithField("result", result).Info("block list result")
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
	defer file.Close()

	hash := md5.New()

	chunk := byte_pool.DefaultBytePool.Get()
	for {
		n, err := file.Read(chunk)
		if err != nil && err != io.EOF {
			logger.Logger.WithContext(ctx).WithError(err).Error("open file fail")
			return "", err
		}

		if err == io.EOF {
			break
		}

		hash.Write(chunk[:n])
	}
	byte_pool.DefaultBytePool.Put(chunk)

	return strings.ToLower(hex.EncodeToString(hash.Sum(nil))), nil
}

func GetFileSize(ctx context.Context, filename string) int64 {
	file, err := os.Stat(filename)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("stat file fail")
		return 0
	}

	return file.Size()
}
