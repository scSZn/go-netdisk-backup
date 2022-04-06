package upload_ui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"backup/consts"
)

var ExportUploadList *UploadList
var retryCancelFunc *context.CancelFunc

func NewUploadTabItem(window fyne.Window) *container.TabItem {
	ExportUploadList = NewUploadList(window)
	return container.NewTabItemWithIcon("上传", theme.SettingsIcon(),
		container.NewBorder(container.New(layout.NewHBoxLayout(), layout.NewSpacer(), &widget.Button{
			Text: "清除上传成功",
			OnTapped: func() {
				ExportUploadList.CleanItem(consts.UploadStatusUploaded)
			},
		}, &widget.Button{
			Text: "重试所有失败",
			OnTapped: func() {
				if retryCancelFunc != nil {
					(*retryCancelFunc)()
				}
				retryCancelFunc = ExportUploadList.RetryAll()
			},
		}, &widget.Button{
			Text: "刷新",
			OnTapped: func() {
				ExportUploadList.Refresh()
			},
		}), nil, nil, nil, container.NewVScroll(ExportUploadList)))
}
