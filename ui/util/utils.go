package util

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// WindowSizeToDialog scales the window size to a suitable dialog size.
func WindowSizeToDialog(s fyne.Size) fyne.Size {
	return fyne.NewSize(s.Width*0.8, s.Height*0.8)
}

func ShowErrorDialog(errmsg string, window fyne.Window) {
	dialog.NewError(errors.New(errmsg), window).Show()
}

func ShowInfoDialog(info string, window fyne.Window) {
	dialog.NewInformation("Info", info, window).Show()
}
