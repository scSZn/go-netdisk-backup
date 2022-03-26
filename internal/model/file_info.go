package model

import (
	"context"
	"os"
	"time"

	"backup/consts"
	"backup/pkg/logger"
	"backup/pkg/util"
)

const FileInfoTableName = "file_info"

type FileInfo struct {
	ID           uint64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"` // 自增ID
	AbsPath      string     `json:"abs_path" gorm:"column:abs_path;unique"`       // 文件绝对路径
	ServerPath   string     `json:"server_path" gorm:"column:server_path"`        // 上传到服务端的地址
	Size         int64      `json:"size" gorm:"column:size"`                      // 文件大小
	Md5          string     `json:"md5" gorm:"column:md5"`                        // 文件md5值
	UploadStatus uint8      `json:"upload_status" gorm:"column:upload_status"`    // 文件上传状态
	CreateTime   *time.Time `json:"create_time" gorm:"column:create_time"`        // 创建时间
	UpdateTime   *time.Time `json:"update_time" gorm:"column:update_time"`        // 更新时间
}

func (f *FileInfo) TableName() string {
	return FileInfoTableName
}

func NewFileInfo(path, excludePrefix string) *FileInfo {
	stat, err := os.Stat(path)
	if err != nil {
		logger.Logger.WithField("path", path).WithError(err).Error("get file stat fail")
		return nil
	}

	md5, err := util.GetFileMd5(context.Background(), path)
	if err != nil {
		logger.Logger.WithField("path", path).WithError(err).Error("generate file md5 fail")
		return nil
	}

	return &FileInfo{
		AbsPath:      path,                                         // 文件绝对路径
		ServerPath:   util.GenerateServerFile(path, excludePrefix), // 服务器上存储的文件名
		Size:         stat.Size(),                                  // 文件大小
		UploadStatus: consts.UploadStatusNoUploaded,                // 未上传状态
		Md5:          md5,                                          // 文件内容MD5值
	}
}
