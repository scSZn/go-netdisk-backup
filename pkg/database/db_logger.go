package database

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type DBLogger struct {
	LogLevel      logger.LogLevel
	logger        *logrus.Logger
	SlowThreshold time.Duration
}

func New(logger *logrus.Logger, logLevel logger.LogLevel, slowThreshold time.Duration) *DBLogger {
	return &DBLogger{
		LogLevel:      logLevel,
		logger:        logger,
		SlowThreshold: slowThreshold,
	}
}

func (l *DBLogger) LogMode(level logger.LogLevel) logger.Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

func (l *DBLogger) Info(ctx context.Context, format string, args ...interface{}) {
	if l.LogLevel >= logger.Info {
		l.logger.WithContext(ctx).Infof(format, args)
	}
}

func (l *DBLogger) Warn(ctx context.Context, format string, args ...interface{}) {
	if l.LogLevel >= logger.Warn {
		l.logger.WithContext(ctx).Warnf(format, args)
	}
}

func (l *DBLogger) Error(ctx context.Context, format string, args ...interface{}) {
	if l.LogLevel >= logger.Error {
		l.logger.WithContext(ctx).Errorf(format, args)
	}
}

func (l *DBLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	fields := map[string]interface{}{}
	fields["caller"] = utils.FileWithLineNum()
	fields["proc"] = fmt.Sprintf("%fms", float64(elapsed.Nanoseconds())/1e6)

	switch {
	case err != nil && l.LogLevel >= logger.Error:
		sql, rows := fc()
		fields["sql"] = sql
		fields["rows"] = rows
		l.logger.Error(ctx, fields, err)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel > logger.Warn:
		sql, rows := fc()
		fields["slowlog"] = fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		fields["sql"] = sql
		fields["rows"] = rows
		l.logger.Warn(ctx, fields, nil)
	case l.LogLevel == logger.Info:
		sql, rows := fc()
		fields["sql"] = sql
		fields["rows"] = rows
		l.logger.Info(ctx, fields)
	}
}
