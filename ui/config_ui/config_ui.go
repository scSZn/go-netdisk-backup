package config_ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

func NewConfigTabItem(window fyne.Window) *container.TabItem {
	return container.NewTabItemWithIcon("配置", theme.SettingsIcon(),
		container.NewVScroll(container.NewVBox(
			NewPcsConfigCard(window).buildCard(),
			NewUploadConfigCard(window).buildCard(),
		)))
}
