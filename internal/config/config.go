package config

import (
	"log"
	"sync"

	"github.com/spf13/viper"
)

var once sync.Once

var pcsConfigPathKey = "pcs_config"

var Config config

type config struct {
	LogConfig    LogConfig    `json:"log"`
	PcsConfig    pcsConfig    `json:"pcs_config"`
	ServerConfig serverConfig `json:"server_config"`
}

type pcsConfig struct {
	AppKey     string `json:"app_key" mapstructure:"app_key"`
	AppSecret  string `json:"secret_key" mapstructure:"secret_key"`
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
		v := viper.New()
		v.SetConfigName("conf")
		v.SetConfigType("yaml")
		v.AddConfigPath("./")
		v.AddConfigPath("./conf")

		err := v.ReadInConfig()
		if err != nil {
			log.Fatalf("read config fail, err: %+v", err)
		}

		pcsConfigPath := v.GetString(pcsConfigPathKey)
		v.SetConfigFile(pcsConfigPath)
		err = v.MergeInConfig()
		if err != nil {
			log.Fatalf("merge config fail, err: %+v", err)
		}

		err = v.UnmarshalKey("log", &Config.LogConfig)
		if err != nil {
			log.Fatalf("unmarshal key `log` fail, err: %+v", err)
		}

		err = v.UnmarshalKey("pcs", &Config.PcsConfig)
		if err != nil {
			log.Fatalf("unmarshal key `pcs` fail, err: %+v", err)
		}

		err = v.UnmarshalKey("server", &Config.ServerConfig)
	})
}
