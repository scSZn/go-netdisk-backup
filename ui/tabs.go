package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"backup/ui/backup_ui"
	"backup/ui/config_ui"
	"backup/ui/upload_ui"
)

func Create(window fyne.Window) *container.AppTabs {
	return &container.AppTabs{Items: []*container.TabItem{
		backup_ui.NewBackupTabItem(window),
		upload_ui.NewUploadTabItem(window),
		config_ui.NewConfigTabItem(window),
	}}
}
