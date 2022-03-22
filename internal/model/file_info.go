package model

import "time"

const FileInfoTableName = "file_info"

type FileInfo struct {
	ID           uint64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"` // 自增ID
	AbsPath      string     `json:"abs_path" gorm:"column:abs_path;unique"`       // 文件绝对路径
	Size         int64      `json:"size" gorm:"column:size"`                      // 文件大小
	Md5          string     `json:"md5" gorm:"column:md5"`                        // 文件md5值
	UploadStatus uint8      `json:"upload_status" gorm:"upload_status"`           // 文件上传状态
	CreateTime   *time.Time `json:"create_time" gorm:"column:create_time"`        // 创建时间
	UpdateTime   *time.Time `json:"update_time" gorm:"column:update_time"`        // 更新时间
}

func (f *FileInfo) TableName() string {
	return FileInfoTableName
}
