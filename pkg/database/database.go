package database

import (
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"backup/consts"
	"backup/internal/config"
	"backup/internal/model"
	"backup/pkg/logger"
)

var DB *gorm.DB

func init() {
	var err error
	DB, err = gorm.Open(sqlite.Open("files.db"), &gorm.Config{
		Logger: New(logger.Logger, TransferLevel(config.Config.LogConfig.Level), 5*time.Second),
	})
	if err != nil {
		log.Fatalf("init db fail, error: %+v", err)
	}

	DB.Callback().Create().Before("gorm:create").Register("gorm:create_time", CreateTimeCallback("create_time"))
	DB.Callback().Create().Before("gorm:create").Register("gorm:update_time", UpdateTimeCallback("update_time"))
	DB.Callback().Create().Before("gorm:update").Register("gorm:update_time", UpdateTimeCallback("update_time"))
	DB.Callback().Create().Before("gorm:delete").Register("gorm:update_time", UpdateTimeCallback("update_time"))
	DB.AutoMigrate(&model.FileInfo{})
}

func TransferLevel(level string) gormLogger.LogLevel {
	switch level {
	case consts.LevelWarn:
		return gormLogger.Warn
	case consts.LevelError:
		return gormLogger.Error
	case consts.LevelInfo:
		return gormLogger.Info
	}

	return gormLogger.Silent
}
