package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type ChineseTheme struct{}

var CustomTheme fyne.Theme = &ChineseTheme{}

// return bundled font resource
// ResourceSourceHanSansTtf 即是 bundle.go 文件中 var 的变量名
func (m ChineseTheme) Font(s fyne.TextStyle) fyne.Resource {
	return resourceAlibabaPuHuiTiMediumTtf
}
func (*ChineseTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(n, v)
}

func (*ChineseTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (*ChineseTheme) Size(n fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(n)
}
