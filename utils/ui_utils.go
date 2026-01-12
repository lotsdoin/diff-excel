package utils

import (
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// MakeWideMultiLineEntry 创建宽Entry组件，禁用编辑（响应式版本，会横向扩展）
func MakeWideMultiLineEntry(defaultInput, placeholder string) (*widget.Entry, fyne.CanvasObject) {
	e := widget.NewMultiLineEntry()
	e.Wrapping = fyne.TextWrapWord
	e.SetMinRowsVisible(2) // 让输入框高度能显示两行
	e.Disable()            // 只读
	e.SetText(defaultInput)
	e.SetPlaceHolder(placeholder)

	// 直接返回 Entry，让调用处的布局控制其扩展行为
	// Entry 作为基础组件，会被 Border、VBox 等容器自动扩展
	return e, e
}

// ParseHexColor 解析 hex 颜色
func ParseHexColor(s string) color.Color {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return color.Black
	}
	r, _ := strconv.ParseUint(s[0:2], 16, 8)
	g, _ := strconv.ParseUint(s[2:4], 16, 8)
	b, _ := strconv.ParseUint(s[4:6], 16, 8)
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

// MakeColorSelector 创建带彩色方块选择器的颜色输入框
func MakeColorSelector(onChange func(string)) fyne.CanvasObject {
	// 常用颜色列表
	colors := []string{
		"#FF0000", "#00FF00", "#0000FF",
		"#800080", "#A52A2A",
		"#008000", "#FF6347",
		"#008080", "#FFD700",
	}

	var colorButtons []fyne.CanvasObject
	for _, c := range colors {
		colorCode := c
		// 彩色背景方块 - 使用合理的固定尺寸
		rect := canvas.NewRectangle(ParseHexColor(c))
		rect.SetMinSize(fyne.NewSize(30, 30))
		// 透明按钮覆盖在方块上
		btn := widget.NewButton("", func() {
			if onChange != nil {
				onChange(colorCode)
			}
		})
		btn.Importance = widget.LowImportance // 去掉高亮样式

		// 把按钮和颜色块叠加
		colorButtons = append(colorButtons, container.NewMax(rect, btn))
	}

	return container.NewHBox(
		container.NewHBox(colorButtons...),
	)
}

// SetBlue 设置按钮为蓝色样式
func SetBlue(b *widget.Button) {
	b.Importance = widget.HighImportance
}

// SetTitle 创建标题文本
func SetTitle(str string, size float32) *canvas.Text {
	text := canvas.NewText(str, &color.NRGBA{R: 255, G: 255, B: 255, A: 255}) // 纯黑色
	text.TextStyle = fyne.TextStyle{Bold: true}
	text.TextSize = size
	return text
}
