package backup_ui

import (
	"context"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"backup/internal/dao"
	"backup/internal/scanner"
	"backup/pkg/database"
	"backup/pkg/logger"
	"backup/pkg/util"
	util_ui "backup/ui/util"
)

const tipsMd = `1. 从百度网盘开放平台获取AppKey和AppSecret
2. 在配置界面配置相应的AppKey和AppSecret，以及备份保存路径
3. 点击配置界面的获取access_token，打开浏览器获取授权码
4. 将浏览器的授权码粘贴到弹窗中，即可获取access_token
5. 接下来即可通过备份管理界面进行备份文件`

func NewBackupTabItem(window fyne.Window) *container.TabItem {
	return container.NewTabItemWithIcon("备份管理", theme.DocumentIcon(),
		NewBackupPathConfig(window).buildUI(),
	)
}

// 备份路径相关配置卡片
type BackupPathConfig struct {
	contentPicker dialog.Dialog

	fileChoice          *widget.Button
	directoryChoice     *widget.Button
	addBackupPathButton *widget.Button
	tipsButton          *widget.Button
	tipsText            *widget.RichText

	window fyne.Window

	fileDialog      *dialog.FileDialog
	directoryDialog *dialog.FileDialog

	backupList *BackupPathList
}

func NewBackupPathConfig(window fyne.Window) *BackupPathConfig {
	return &BackupPathConfig{
		window: window,
	}
}

func (c *BackupPathConfig) OnFileSelect(uri fyne.URIReadCloser, err error) {
	if err != nil {
		logger.Logger.WithError(err).Error("open backup path fail")
		util_ui.ShowErrorDialog("打开文件窗口失败", c.window)
		return
	}
	if uri == nil {
		return
	}
	path := uri.URI().Path()
	affected, err := c.backupList.AddItem(filepath.Clean(uri.URI().Path()))
	if err != nil {
		logger.Logger.WithError(err).WithField("path", uri.URI().Path()).Error("add backup path fail")
		util_ui.ShowErrorDialog("添加备份文件失败", c.window)
		return
	}

	if affected == 0 {
		util_ui.ShowErrorDialog("备份目录/文件已存在", c.window)
		return
	}

	_scanner, err := scanner.NewScanner(util.NewContext(), path)
	scanner.Manager.Add(_scanner)
	go _scanner.ScanAndUpload()
}

func (c *BackupPathConfig) OnDirSelect(uri fyne.ListableURI, err error) {
	if err != nil {
		logger.Logger.WithError(err).Error("open backup path fail")
		util_ui.ShowErrorDialog("打开文件窗口失败", c.window)
		return
	}
	if uri == nil {
		return
	}
	path := uri.Path()
	affected, err := c.backupList.AddItem(filepath.Clean(uri.Path()))
	if err != nil {
		logger.Logger.WithError(err).WithField("path", uri.Path()).Error("add backup path fail")
		util_ui.ShowErrorDialog("添加备份目录失败", c.window)
		return
	}

	if affected == 0 {
		util_ui.ShowErrorDialog("备份目录/文件已存在", c.window)
		return
	}
	_scanner, err := scanner.NewScanner(util.NewContext(), path)
	scanner.Manager.Add(_scanner)
	go _scanner.ScanAndUpload()
}

func (c *BackupPathConfig) openFileDialog() {
	c.contentPicker.Hide()
	c.fileDialog.Resize(util_ui.WindowSizeToDialog(c.window.Canvas().Size()))
	c.fileDialog.Show()
	c.fileDialog.Refresh()
}

func (c *BackupPathConfig) openDirDialog() {
	c.contentPicker.Hide()
	c.directoryDialog.Resize(util_ui.WindowSizeToDialog(c.window.Canvas().Size()))
	c.directoryDialog.Show()
	c.directoryDialog.Refresh()
}

func (c *BackupPathConfig) buildUI() *fyne.Container {
	c.fileChoice = &widget.Button{Text: "文件", Icon: theme.FileIcon(), OnTapped: c.openFileDialog}
	c.directoryChoice = &widget.Button{Text: "目录", Icon: theme.FolderOpenIcon(), OnTapped: c.openDirDialog}

	c.fileDialog = dialog.NewFileOpen(c.OnFileSelect, c.window)
	c.directoryDialog = dialog.NewFolderOpen(c.OnDirSelect, c.window)

	choiceContent := container.New(layout.NewVBoxLayout(), c.fileChoice, c.directoryChoice)
	c.contentPicker = dialog.NewCustom("请选择备份的类型", "取消", choiceContent, c.window)

	c.addBackupPathButton = &widget.Button{Text: "添加上传文件", Icon: theme.ContentAddIcon(), OnTapped: func() {
		c.contentPicker.Show()
	}}
	c.tipsText = widget.NewRichTextFromMarkdown(tipsMd)
	c.tipsButton = &widget.Button{
		Text:       "操作指南",
		Importance: widget.MediumImportance,
		OnTapped: func() {
			dialog.NewCustom("操作指南", "确认", c.tipsText, c.window).Show()
		},
	}

	backupPathDao := dao.NewBackupPathDao(context.Background(), database.DB)
	var items []string
	for _, pathModel := range backupPathDao.GetAll() {
		items = append(items, pathModel.AbsPath)
	}
	c.backupList = NewBackupPathList(items, c.window)
	content := container.NewBorder(c.addBackupPathButton, c.tipsButton, nil, nil, c.backupList)
	return content
}
