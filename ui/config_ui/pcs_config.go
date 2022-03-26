package config_ui

import (
	"fmt"
	"image/color"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"gopkg.in/yaml.v3"

	"backup/consts"
	"backup/internal/config"
	"backup/internal/token"
	"backup/pkg/logger"
	"backup/pkg/util"
	ui_util "backup/ui/util"
)

type PcsConfigCard struct {
	appKeyEntry    *widget.Entry
	appSecretEntry *widget.Entry
	//tokenPathEntry  *widget.Entry
	prefixPathEntry *widget.Entry

	getAccessTokenBtn *widget.Button
	saveBtn           *widget.Button
	resetBtn          *widget.Button

	window fyne.Window
}

func NewPcsConfigCard(window fyne.Window) *PcsConfigCard {
	pcsConfigUI := &PcsConfigCard{
		window: window,
	}
	return pcsConfigUI
}

func (p *PcsConfigCard) buildCard() *widget.Card {
	p.appKeyEntry = &widget.Entry{PlaceHolder: "百度网盘开放平台的AppKey", Text: config.Config.PcsConfig.AppKey}
	p.appSecretEntry = &widget.Entry{PlaceHolder: "百度网盘开放平台的AppSecret", Text: config.Config.PcsConfig.AppSecret}
	//p.tokenPathEntry = &widget.Entry{PlaceHolder: "百度网盘存储的token存储路径"}
	p.prefixPathEntry = &widget.Entry{PlaceHolder: "备份文件在百度网盘存储的路径", Text: config.Config.PcsConfig.PathPrefix}
	p.getAccessTokenBtn = widget.NewButton("获取access_token", func() {
		if !config.Config.PcsConfig.IsValid() {
			ui_util.ShowInfoDialog("请先配置AppKey和AppSecret", p.window)
			return
		}
		codeToken := &widget.Entry{PlaceHolder: "请输入授权码                                          "}

		err := util.OpenBrowser()
		if err != nil {
			ui_util.ShowErrorDialog(fmt.Sprintf("打开浏览器失败，请手动打开浏览器输入\n%s", fmt.Sprintf(consts.AuthorizationCodeUrl, config.Config.PcsConfig.AppKey)), p.window)
			return
		}
		dialog.NewCustomConfirm("授权码", "确认", "取消", codeToken, func(b bool) {
			if !b {
				return
			}
			err := token.RefreshTokenFromServerByCode(codeToken.Text)
			if err != nil {
				logger.Logger.WithField("code", codeToken.Text).WithError(err).Error("get token fail")
				ui_util.ShowErrorDialog("获取token失败，请检查AppKey和AppSecret是否正确", p.window)
				return
			}
			ui_util.ShowInfoDialog("获取token成功", p.window)
		}, p.window).Show()
	})
	p.saveBtn = &widget.Button{
		Text:       "保存",
		Importance: widget.HighImportance,
		OnTapped:   p.SaveConfig,
	}

	p.resetBtn = &widget.Button{
		Text:       "重置",
		Importance: widget.MediumImportance,
		OnTapped: func() {
			p.appKeyEntry.Text = ""
			p.appSecretEntry.Text = ""
			p.prefixPathEntry.Text = ""

			p.appKeyEntry.Refresh()
			p.appSecretEntry.Refresh()
			p.prefixPathEntry.Refresh()
		},
	}

	tipText1 := canvas.NewText("tips: AppKey和AppSecret需要去百度网盘开放平台申请", color.Black)
	tipText2 := canvas.NewText("平台网址：https://pan.baidu.com/union/home", color.Black)
	tipText1.TextSize = fyne.CurrentApp().Settings().Theme().Size("text") * 0.8
	tipText2.TextSize = fyne.CurrentApp().Settings().Theme().Size("text") * 0.8

	pcsContainer := container.NewVBox(container.New(layout.NewFormLayout(),
		newBoldLabel("AppKey"), p.appKeyEntry,
		newBoldLabel("AppSecret"), p.appSecretEntry,
		//newBoldLabel("token存储"), p.tokenPathEntry,
		newBoldLabel("存储路径"), p.prefixPathEntry,
	), container.NewHBox(layout.NewSpacer(), p.resetBtn, p.saveBtn), p.getAccessTokenBtn, tipText1, tipText2)

	return &widget.Card{Title: "网盘相关配置", Content: pcsContainer}
}

// 保存配置
func (p *PcsConfigCard) SaveConfig() {
	pcsConfig := map[string]interface{}{
		"pcs": map[string]string{
			"app_key":     p.appKeyEntry.Text,
			"app_secret":  p.appSecretEntry.Text,
			"token_path":  "token.json",
			"path_prefix": p.prefixPathEntry.Text,
		},
	}
	data, err := yaml.Marshal(pcsConfig)
	if err != nil {
		logger.Logger.WithField("config", pcsConfig).WithError(err).Error("yaml marshal fail")
		ui_util.ShowErrorDialog("保存配置失败", p.window)
		return
	}

	file, err := os.OpenFile(config.PcsConfigPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logger.Logger.WithField("config", pcsConfig).WithField("filename", config.PcsConfigPath).WithError(err).Error("open pcs config file fail")
		ui_util.ShowErrorDialog("保存配置失败", p.window)
		return
	}
	defer file.Close()

	err = file.Truncate(0)
	if err != nil {
		logger.Logger.WithField("config", pcsConfig).WithField("filename", config.PcsConfigPath).WithError(err).Error("truncate pcs config file fail")
		ui_util.ShowErrorDialog("保存配置失败", p.window)
		return
	}

	_, err = file.Write(data)
	if err != nil {
		logger.Logger.WithField("config", pcsConfig).WithField("filename", config.PcsConfigPath).WithError(err).Error("write pcs config fail")
		ui_util.ShowErrorDialog("保存配置失败", p.window)
		return
	}

	ui_util.ShowInfoDialog("保存配置成功", p.window)

	RefreshPcsConfig()
}

func RefreshPcsConfig() {
	config.PcsConfigViper.SetConfigFile(config.PcsConfigPath)
	err := config.PcsConfigViper.ReadInConfig()
	if err != nil {
		logger.Logger.WithField("pcs_config", config.PcsConfigPath).WithError(err).Error("read pcs_config fail")
		return
	}
	err = config.PcsConfigViper.UnmarshalKey("pcs", &config.Config.PcsConfig)
	if err != nil {
		logger.Logger.WithField("pcs_config", config.PcsConfigPath).WithError(err).Error("refresh pcs_config fail")
	}
}

func newBoldLabel(text string) *widget.Label {
	return &widget.Label{
		Text: text,
		TextStyle: fyne.TextStyle{
			Bold: true,
		},
	}
}
