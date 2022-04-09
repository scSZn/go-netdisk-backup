package config_ui

import (
	"fmt"
	"os"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"gopkg.in/yaml.v3"

	"backup/consts"
	"backup/internal/config"
	"backup/pkg/logger"
	"backup/ui/upload_ui"
	ui_util "backup/ui/util"
)

type UploadConfigCard struct {
	slider      *widget.Slider
	sliderLabel *widget.Label

	saveBtn *widget.Button

	window fyne.Window
}

func NewUploadConfigCard(window fyne.Window) *UploadConfigCard {
	return &UploadConfigCard{
		window: window,
	}
}

func (c *UploadConfigCard) buildCard() *widget.Card {
	uploadCount := config.GetUploadCount()
	c.slider = &widget.Slider{
		Min:   1,
		Max:   10,
		Value: float64(uploadCount),
		Step:  1,
		OnChanged: func(f float64) {
			value := fmt.Sprintf("%.f", f)
			c.sliderLabel.SetText(value)
		},
	}

	c.sliderLabel = widget.NewLabel(strconv.Itoa(uploadCount))
	c.saveBtn = &widget.Button{
		Text:       "保存",
		Importance: widget.HighImportance,
		OnTapped:   c.SaveConfig,
	}

	return &widget.Card{
		Title: "上传配置",
		Content: container.NewVBox(container.NewGridWithColumns(2,
			widget.NewLabel("同时上传文件数"),
			container.NewBorder(nil, nil, nil, c.sliderLabel, c.slider),
		), container.NewHBox(layout.NewSpacer(), c.saveBtn)),
	}
}

func (c *UploadConfigCard) SaveConfig() {
	value := int(c.slider.Value)
	file, err := os.OpenFile(config.UploadConfigPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logger.Logger.WithField("path", config.UploadConfigPath).WithError(err).Error("open file fail")
		ui_util.ShowErrorDialog("保存配置失败", c.window)
		return
	}
	defer file.Close()

	err = file.Truncate(0)
	if err != nil {
		logger.Logger.WithField("path", config.UploadConfigPath).WithError(err).Error("truncate file fail")
		ui_util.ShowErrorDialog("保存配置失败", c.window)
		return
	}

	data, err := yaml.Marshal(map[string]int{
		consts.UploadCountKey: value,
	})
	if err != nil {
		logger.Logger.WithField("path", config.UploadConfigPath).WithField("config", map[string]int{
			consts.UploadCountKey: value,
		}).WithError(err).Error("marshal data fail")
		ui_util.ShowErrorDialog("保存配置失败", c.window)
		return
	}

	_, err = file.Write(data)
	if err != nil {
		logger.Logger.WithField("path", config.UploadConfigPath).WithField("config", map[string]int{
			consts.UploadCountKey: value,
		}).WithError(err).Error("write file fail")
		ui_util.ShowErrorDialog("保存配置失败", c.window)
		return
	}
	ui_util.ShowInfoDialog("保存配置成功", c.window)
	upload_ui.ExportUploadList.AddSignal()

	// 有可能文件不存在，这里补充配置
	config.UploadConfigViper.SetConfigFile(config.UploadConfigPath)
}
