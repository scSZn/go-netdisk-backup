package logger

import (
	"log"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"backup/consts"
	"backup/internal/config"
)

var Logger *logrus.Logger

// 默认配置
var defaultConfig = &config.LogConfig{
	Path:    "./log",
	Level:   "INFO",
	MaxSize: 20,
	Backup:  10,
}

func init() {
	Init(defaultConfig)
}

func Init(config *config.LogConfig) {
	filename := config.Path + "/" + time.Now().Format(consts.TimeFormatLog) + ".log"
	output := &lumberjack.Logger{
		LocalTime: true,
		Filename:  filename,
	}
	if config.MaxSize != 0 {
		output.MaxSize = config.MaxSize
	}
	if config.Backup != 0 {
		output.MaxBackups = config.Backup
	}
	Logger = logrus.New()
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		log.Fatalf("log level is illegal: [%+v]", config.Level)
	}
	Logger.SetLevel(level)
	Logger.SetOutput(output)
	Logger.SetReportCaller(true)
	Logger.SetFormatter(&LogFormatter{})
}
