package dao

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"backup/internal/model"
	"backup/pkg/logger"
)

type BackupPathDao struct {
	ctx context.Context
	DB  *gorm.DB
}

func NewBackupPathDao(ctx context.Context, db *gorm.DB) *BackupPathDao {
	return &BackupPathDao{
		ctx: ctx,
		DB:  db,
	}
}

func (d *BackupPathDao) Add(info *model.BackupPath) (int64, error) {
	db := d.DB.Table(model.BackupPathTableName).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "abs_path"}}, DoNothing: true}).Create(info)
	err := db.Error
	if err != nil {
		logger.Logger.WithContext(d.ctx).WithError(err).WithField("info", info).Error("create backup path fail")
		return 0, err
	}
	return db.RowsAffected, nil
}

func (d *BackupPathDao) QueryByAbsPath(absPath string) (*model.BackupPath, error) {
	var result = model.BackupPath{}
	db := d.DB.Table(model.BackupPathTableName).Where("abs_path = ?", absPath).First(&result)
	err := db.Error
	if err != nil {
		logger.Logger.WithContext(d.ctx).WithError(err).WithField("abs_path", absPath).Error("query backup path fail")
		return nil, err
	}
	return &result, nil
}

func (d *BackupPathDao) Total() int64 {
	var count int64
	err := d.DB.Table(model.BackupPathTableName).Count(&count).Error
	if err != nil {
		logger.Logger.WithContext(d.ctx).WithError(err).Error("count backup path fail")
		return 0
	}
	return 0
}

func (d *BackupPathDao) GetAll() []*model.BackupPath {
	var result []*model.BackupPath
	err := d.DB.Table(model.BackupPathTableName).Find(&result).Error
	if err != nil {
		logger.Logger.WithContext(d.ctx).WithError(err).Error("get all backup path fail")
		return nil
	}
	return result
}

func (d *BackupPathDao) Delete(absPath string) error {
	err := d.DB.Table(model.BackupPathTableName).Where("abs_path = ?", absPath).Delete(&model.BackupPath{}).Error
	if err != nil {
		logger.Logger.WithContext(d.ctx).WithError(err).WithField("abs_path", absPath).Error("delete backup path fail")
		return err
	}
	return nil
}
