package backup_ui

import (
	"context"
	"image/color"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"backup/internal/dao"
	"backup/internal/model"
	"backup/internal/scanner"
	"backup/pkg/database"
	"backup/pkg/logger"
	"backup/ui/util"
)

type BackupPathList struct {
	widget.List

	items []string

	window fyne.Window
}

func NewBackupPathList(item []string, window fyne.Window) *BackupPathList {
	list := &BackupPathList{
		items:  item,
		window: window,
	}
	list.List.Length = list.Length
	list.List.CreateItem = list.CreateItem
	list.List.UpdateItem = list.UpdateItem
	list.List.OnSelected = list.OnSelected
	list.List.OnUnselected = list.OnUnselected

	list.ExtendBaseWidget(list)
	return list
}

func (l *BackupPathList) CreateRenderer() fyne.WidgetRenderer {
	return l.List.CreateRenderer()
}

func (l *BackupPathList) Length() int {
	return len(l.items)
}

func (l *BackupPathList) CreateItem() fyne.CanvasObject {
	text := &canvas.Text{
		Text:     "",
		TextSize: fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText),
		Color:    color.Black,
	}
	text.Move(fyne.NewPos(theme.Padding()/2, theme.Padding()/2))
	button := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		if err := l.DeleteItem(text.Text); err != nil {
			logger.Logger.WithField("abs_path", text.Text).WithError(err).Error("delete item fail")
			util.ShowErrorDialog("删除备份路径失败", l.window)
		}
	})

	return container.New(layout.NewHBoxLayout(), text, layout.NewSpacer(), button)
}

func (l *BackupPathList) UpdateItem(id widget.ListItemID, item fyne.CanvasObject) {
	c := item.(*fyne.Container)
	c.Objects[0].(*canvas.Text).Text = l.items[id]
}

func (l *BackupPathList) OnSelected(id widget.ListItemID) {
	l.List.Unselect(id)
}

func (l *BackupPathList) OnUnselected(id widget.ListItemID) {

}

func (l *BackupPathList) AddItem(item string) (int64, error) {
	stat, err := os.Stat(item)
	if err != nil {
		logger.Logger.WithField("abs_path", item).Errorf("get stat fail, err: ")
		return 0, err
	}

	path := &model.BackupPath{
		AbsPath: item,
		IsDir:   stat.IsDir(),
	}

	affected, err := dao.NewBackupPathDao(context.Background(), database.DB).Add(path)

	if err != nil {
		return 0, err
	}

	l.items = append(l.items, item)

	l.Refresh()
	return affected, nil
}

func (l *BackupPathList) DeleteItem(nowItem string) error {
	nowItem = filepath.Clean(nowItem)

	transaction := database.DB.Begin()
	fileInfoDao := dao.NewFileInfoDao(context.Background(), transaction)
	backupPathDao := dao.NewBackupPathDao(context.Background(), transaction)

	err := backupPathDao.Delete(nowItem)
	if err != nil {
		transaction.Rollback()
		return err
	}

	err = fileInfoDao.DeleteAllByPrefix(nowItem)
	if err != nil {
		transaction.Rollback()
		return err
	}
	transaction.Commit()

	newItems := make([]string, 0, len(l.items))
	for _, item := range l.items {
		if item == nowItem {
			continue
		}
		newItems = append(newItems, item)
	}

	l.items = newItems
	scanner.Manager.Remove(nowItem)
	//upload_ui.ExportUploadList.ClearItemPrefix(nowItem)

	l.Refresh()
	return nil
}
