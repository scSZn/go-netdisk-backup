package dao

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"backup/internal/model"
	"backup/pkg/logger"
)

type FileInfoDao struct {
	ctx context.Context
	DB  *gorm.DB
}

func NewFileInfoDao(ctx context.Context, db *gorm.DB) *FileInfoDao {
	return &FileInfoDao{
		ctx: ctx,
		DB:  db,
	}
}

func (d *FileInfoDao) Add(info *model.FileInfo) error {
	err := d.DB.Table(model.FileInfoTableName).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "abs_path"}}, UpdateAll: true}).Create(info).Error
	if err != nil {
		logger.Logger.WithContext(d.ctx).WithError(err).WithField("info", info).Error("create file info fail")
		return err
	}
	return nil
}

func (d *FileInfoDao) Update(updates map[string]interface{}, absPath string) error {
	err := d.DB.Table(model.FileInfoTableName).Where("abs_path = ?", absPath).Updates(updates).Error
	if err != nil {
		logger.Logger.WithContext(d.ctx).WithError(err).WithField("updates", updates).Error("update file info fail")
		return err
	}
	return nil
}

func (d *FileInfoDao) QueryByAbsPath(absPath string) (*model.FileInfo, error) {
	var res = &model.FileInfo{}
	if err := d.DB.Table(model.FileInfoTableName).Where("abs_path = ?", absPath).First(&res).Error; err != nil && err != gorm.ErrRecordNotFound {
		logger.Logger.WithContext(d.ctx).WithError(err).WithField("filename", absPath).Error("query file info fail")
		return nil, err
	}
	return res, nil
}
