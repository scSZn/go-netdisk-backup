package token

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	jsoniter "github.com/json-iterator/go"

	"backup/consts"
	"backup/internal/config"
	"backup/pkg/logger"
)

var AccessToken string
var RefreshToken string

type Config struct {
	AccessToken  Token `json:"access_token"`
	RefreshToken Token `json:"refresh_token"`
}

type Token struct {
	Value     string `json:"value"`
	StartTime string `json:"start_time" time_format:"2006-01-02 15:04:05"`
}

type TokenResponse struct {
	ExpiresIn     int    `json:"expires_in"`
	RefreshToken  string `json:"refresh_token"`
	AccessToken   string `json:"access_token"`
	SessionSecret string `json:"session_secret"`
	SessionKey    string `json:"session_key"`
	Scope         string `json:"scope"`
}

var watched = false

func init() {
	//AccessToken = "121.27abc2481b81f4c2a748f553362974a8.YljI3ndWLKW3GD1cBCdYltnC1vX6-pyHtfyL0-T.5GCmjA"
	err := RefreshTokenFromFile()
	if err != nil {
		logger.Logger.WithError(err).Error("refresh token from file fail")
	}
	watchTokenFile()
}

func StoreToken(accessToken, refreshToken string) error {
	now := time.Now().Format(consts.TimeFormatSecond)
	tokenConfig := &Config{
		AccessToken: Token{
			Value:     accessToken,
			StartTime: now,
		},
		RefreshToken: Token{
			Value:     refreshToken,
			StartTime: now,
		},
	}

	data, err := jsoniter.Marshal(tokenConfig)
	if err != nil {
		logger.Logger.WithField("accessToken", accessToken).WithField("refreshToken", refreshToken).WithError(err).Error("marshal token fail")
		return err
	}

	file, err := os.OpenFile(config.Config.PcsConfig.TokenPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		logger.Logger.WithField("accessToken", accessToken).WithField("refreshToken", refreshToken).WithError(err).Errorf("open file [%s] fail", config.Config.PcsConfig.TokenPath)
		return err
	}

	err = file.Truncate(0)
	if err != nil {
		logger.Logger.WithField("accessToken", accessToken).WithField("refreshToken", refreshToken).WithError(err).Errorf("truncate file [%s] fail", config.Config.PcsConfig.TokenPath)
		return err
	}

	_, err = file.Write(data)
	if err != nil {
		logger.Logger.WithField("accessToken", accessToken).WithField("refreshToken", refreshToken).WithError(err).Errorf("write file [%s] fail", config.Config.PcsConfig.TokenPath)
		return err
	}

	return nil
}

func RefreshTokenFromFile() error {
	file, err := os.Open(config.Config.PcsConfig.TokenPath)
	if err != nil {
		logger.Logger.WithError(err).Errorf("open file [%s] fail", config.Config.PcsConfig.TokenPath)
		return err
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		logger.Logger.WithError(err).Error("read token fail")
		return err
	}

	AccessToken = jsoniter.Get(data, "access_token", "value").ToString()
	RefreshToken = jsoniter.Get(data, "refresh_token", "value").ToString()
	logger.Logger.WithField("access_token", AccessToken).Info("access_token refreshed from file")
	return nil
}

func RefreshTokenFromServerByCode(code string) error {
	url := fmt.Sprintf(consts.AccessTokenCodeUrl, code, config.Config.PcsConfig.AppKey, config.Config.PcsConfig.AppSecret)

	logger.Logger.WithField("code", code).WithField("url", url).Info("start request token from server")
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		logger.Logger.WithField("url", url).WithError(err).Error("request fail")
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Logger.WithField("url", url).WithError(err).Error("read data fail")
		return err
	}
	logger.Logger.WithField("data", string(data)).Info("get token")

	var tokenResp TokenResponse
	err = jsoniter.Unmarshal(data, &tokenResp)
	if err != nil {
		logger.Logger.WithField("url", url).WithField("data", string(data)).WithError(err).Error("unmarshal data fail")
		return err
	}

	err = StoreToken(tokenResp.AccessToken, tokenResp.RefreshToken)
	if err != nil {
		logger.Logger.WithField("url", url).WithField("token", tokenResp).WithError(err).Error("store token fail")
		return err
	}

	RefreshTokenFromFile()

	logger.Logger.WithField("access_token", AccessToken).Info("access_token refreshed from server")
	return nil
}

func RefreshTokenFromServerByRefreshCode() error {
	url := fmt.Sprintf(consts.AccessTokenRefreshUrl, RefreshToken, config.Config.PcsConfig.AppKey, config.Config.PcsConfig.AppSecret)

	logger.Logger.WithField("refreshToken", RefreshToken).WithField("url", url).Info("start request token from server")
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		logger.Logger.WithField("url", url).WithError(err).Error("request fail")
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Logger.WithField("url", url).WithError(err).Error("read data fail")
		return err
	}
	logger.Logger.WithField("data", string(data)).Info("get token")

	var tokenResp TokenResponse
	err = jsoniter.Unmarshal(data, &tokenResp)
	if err != nil {
		logger.Logger.WithField("url", url).WithField("data", string(data)).WithError(err).Error("unmarshal data fail")
		return err
	}

	err = StoreToken(tokenResp.AccessToken, tokenResp.RefreshToken)
	if err != nil {
		logger.Logger.WithField("url", url).WithField("token", tokenResp).WithError(err).Error("store token fail")
		return err
	}

	RefreshTokenFromFile()

	logger.Logger.WithField("access_token", AccessToken).Info("access_token refreshed from server")
	return nil
}

func watchTokenFile() {
	filename := config.Config.PcsConfig.TokenPath
	if filename == "" {
		return
	}
	if watched {
		return
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Logger.WithField("filename", filename).WithError(err).Error("new watcher fail")
		return
	}

	err = watcher.Add(filename)
	if err != nil {
		logger.Logger.WithField("filename", filename).WithError(err).Error("add watch filename fail")
		return
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				logger.Logger.WithField("event", event).Info("token file changed, start refresh token")
				err := RefreshTokenFromFile()
				if err != nil {
					logger.Logger.WithField("filename", filename).WithError(err).Error("refresh token from file fail")
					return
				}
				logger.Logger.WithField("event", event).Info("refresh token success")
			}
		}
	}()

	watched = true
}
