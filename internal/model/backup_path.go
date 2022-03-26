package model

import "time"

const BackupPathTableName = "backup_path"

type BackupPath struct {
	ID         uint64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"` // 自增ID
	AbsPath    string     `json:"abs_path" gorm:"column:abs_path;unique"`       // 文件绝对路径
	IsDir      bool       `json:"is_dir" gorm:"column:is_dir"`                  // 是否是文件夹
	CreateTime *time.Time `json:"create_time" gorm:"column:create_time"`        // 创建时间
	UpdateTime *time.Time `json:"update_time" gorm:"column:update_time"`        // 更新时间
}

func (b *BackupPath) TableName() string {
	return BackupPathTableName
}
