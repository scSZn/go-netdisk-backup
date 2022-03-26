package config

import (
	"log"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var once sync.Once

const configString = `pcs_config: pcs_config.yaml
upload_config: upload_config.yaml
log:
  path: log
  level: INFO
  max_size: 10
  backup: 10
server:
  host: 127.0.0.1
  port: 8080`

var (
	pcsConfigPathKey    = "pcs_config"
	uploadConfigPathKey = "upload_config"

	UploadCountKey = "upload_count"

	PcsConfigPath    string
	UploadConfigPath string

	ConfigViper       = viper.New()
	PcsConfigViper    = viper.New()
	UploadConfigViper = viper.New()
)

var Config config

type config struct {
	LogConfig    LogConfig    `json:"log"`
	PcsConfig    PcsConfig    `json:"pcs_config"`
	ServerConfig serverConfig `json:"server_config"`
}

type PcsConfig struct {
	AppKey     string `json:"app_key" mapstructure:"app_key"`
	AppSecret  string `json:"app_secret" mapstructure:"app_secret"`
	TokenPath  string `json:"token_path" mapstructure:"token_path"`
	PathPrefix string `json:"path_prefix" mapstructure:"path_prefix"`
}

type LogConfig struct {
	Path    string `json:"path" mapstructure:"path"`
	Level   string `json:"level" mapstructure:"level"`
	MaxSize int    `json:"max_size" mapstructure:"max_size"`
	Backup  int    `json:"backup" mapstructure:"backup"`
}

type serverConfig struct {
	Host string `json:"host" mapstructure:"host"`
	Port int    `json:"port" mapstructure:"port"`
}

func init() {
	once.Do(func() {
		ConfigViper.SetConfigType("yaml")
		ConfigViper.ReadConfig(strings.NewReader(configString))

		PcsConfigPath = ConfigViper.GetString(pcsConfigPathKey)
		UploadConfigPath = ConfigViper.GetString(uploadConfigPathKey)

		// 获取日志配置
		err := ConfigViper.UnmarshalKey("log", &Config.LogConfig)
		if err != nil {
			log.Fatalf("unmarshal key `log` fail, err: %+v", err)
		}

		// 获取PCS配置
		PcsConfigViper.SetConfigFile(PcsConfigPath)
		PcsConfigViper.ReadInConfig()
		err = PcsConfigViper.UnmarshalKey("pcs", &Config.PcsConfig)
		if err != nil {
			log.Fatalf("unmarshal key `pcs` fail, err: %+v", err)
		}

		UploadConfigViper.SetConfigFile(UploadConfigPath)
		UploadConfigViper.ReadInConfig()
		UploadConfigViper.WatchConfig()
	})
}

func (p *PcsConfig) IsValid() bool {
	return !(p.AppKey == "" || p.AppSecret == "")
}
