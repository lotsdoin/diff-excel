package main

import (
	"diff_excel/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// SheetPair 表示一对要对比的Sheet
type SheetPair struct {
	SrcSheet    string // 源文件Sheet名
	CmpSheet    string // 对比文件Sheet名
	DisplayName string // 显示名称（用于结果文件）
}

// ExcelCompareApp Excel对比应用程序结构体
type ExcelCompareApp struct {
	myApp              fyne.App
	myWindow           fyne.Window
	logRich            *widget.Entry
	srcFile            string
	cmpFile            string
	sheetMappings      map[string]string // 源Sheet -> 目标Sheet 的映射（保持兼容性）
	sheetPairs         []SheetPair       // 新的Sheet对比组列表
	outDir             string
	outExcelFile       string
	outLogFile         string
	highlightClr       string
	showOldInComment   bool
	keepOriginalFormat bool
	multiSheetMode     bool // 是否启用多Sheet模式

	// 兼容单Sheet模式的字段
	srcSheet string
	cmpSheet string

	// Sheet数据
	srcSheets         []string
	cmpSheets         []string
	selectedSrcSheets map[string]bool
	selectedCmpSheets map[string]bool

	// UI组件
	srcEntry            *widget.Entry
	srcEntryBox         fyne.CanvasObject
	cmpEntry            *widget.Entry
	cmpEntryBox         fyne.CanvasObject
	outDirEntry         *widget.Entry
	outDirBox           fyne.CanvasObject
	srcSheetSelect      *widget.Select // 单Sheet模式
	cmpSheetSelect      *widget.Select // 单Sheet模式
	srcSheetMultiSelect *widget.List   // 多Sheet模式
	cmpSheetMultiSelect *widget.List   // 多Sheet模式
	sheetMappingList    *widget.List
	multiSheetCheckbox  *widget.Check
	colorEntry          *widget.Entry
	commentCheckbox     *widget.Check
	formatCheckbox      *widget.Check

	// 新的Sheet对比组UI组件
	currentSrcSheetSelect *widget.Select // 当前选择的源Sheet
	currentCmpSheetSelect *widget.Select // 当前选择的对比Sheet
	sheetPairsList        *widget.List   // Sheet对比组列表
	addPairButton         *widget.Button // 添加对比组按钮
	autoMatchButton       *widget.Button // 自动匹配所有Sheet按钮

	// 功能函数
	srcUploadFunc func()
	cmpUploadFunc func()

	// 异步操作支持
	isComparing bool     // 是否正在比较中
	logChan    chan string // 日志通道，用于从 goroutine 安全地更新 UI
}

// NewExcelCompareApp 创建新的Excel对比应用程序
func NewExcelCompareApp(w, h float32) *ExcelCompareApp {
	// 获取当前程序运行目录作为默认输出目录
	currentDir, err := os.Getwd()
	if err != nil {
		currentDir = "."
	}

	a := &ExcelCompareApp{
		myApp:             app.NewWithID("com.zbuzhi.diffexcel"),
		highlightClr:      "#FF0000",
		sheetMappings:     make(map[string]string),
		selectedSrcSheets: make(map[string]bool),
		selectedCmpSheets: make(map[string]bool),
		multiSheetMode:    false,
		outDir:            currentDir, // 设置默认输出目录
		logChan:           make(chan string, 100), // 缓冲 100 条日志
	}

	a.myWindow = a.myApp.NewWindow("Excel 对比")
	// 设置初始窗口大小，增加宽度以更好地显示文件路径
	a.myWindow.Resize(fyne.NewSize(w+100, h))
	// 居中显示窗口
	a.myWindow.CenterOnScreen()

	// 初始化只读日志框
	a.logRich = widget.NewMultiLineEntry()
	a.logRich.Wrapping = fyne.TextWrapWord
	a.logRich.Disable() // 只读模式，禁止手动输入

	// 启动日志处理 goroutine（在 logRich 初始化之后）
	go func() {
		for text := range a.logChan {
			// 在主线程中执行 UI 更新
			fyne.Do(func() {
				a.logRich.Append(text)
			})
		}
	}()

	a.initUI()
	return a
}

// appendLog 追加日志：通过 channel 发送到 UI 线程（线程安全）
func (a *ExcelCompareApp) appendLog(text string) {
	fmt.Print(text) // 保留终端输出
	select {
	case a.logChan <- text:
	default:
		// 通道满时丢弃（避免阻塞对比流程）
	}
}

// appendLogUI 追加日志：仅从UI线程调用（更快）
func (a *ExcelCompareApp) appendLogUI(text string) {
	fmt.Print(text)
	a.logRich.Append(text)
}

// compareFunc 执行对比操作的主函数（异步执行）
func (a *ExcelCompareApp) compareFunc() {
	if a.isComparing {
		dialog.ShowInformation("提示", "正在对比中，请稍候...", a.myWindow)
		return
	}

	if a.srcFile == "" || a.cmpFile == "" {
		dialog.ShowError(fmt.Errorf("请先选择源 Excel 和对比 Excel 文件"), a.myWindow)
		return
	}

	// 检查模式和选择
	if a.multiSheetMode {
		// 多Sheet模式检查
		if len(a.sheetMappings) == 0 {
			dialog.ShowError(fmt.Errorf("多Sheet模式下请选择要对比的Sheet"), a.myWindow)
			return
		}
	} else {
		// 检查是否有Sheet对比组（灵活多Sheet对比）
		if len(a.sheetPairs) == 0 {
			dialog.ShowError(fmt.Errorf("请先添加Sheet对比组"), a.myWindow)
			return
		}
		a.appendLogUI("使用灵活多Sheet对比模式\n")
	}

	if a.outDir == "" {
		dialog.ShowError(fmt.Errorf("请选择输出目录"), a.myWindow)
		return
	}

	a.highlightClr = strings.TrimSpace(a.colorEntry.Text)
	if !utils.IsValidColorCode(a.highlightClr) {
		dialog.ShowError(fmt.Errorf("颜色格式错误，需形如 #RRGGBB 或 #RGB"), a.myWindow)
		return
	}
	timeNow := time.Now().Format("2006.01.02_15_04_05")

	// 在输出目录下创建时间命名的子目录
	timeDirName := time.Now().Format("20060102_15.04.05")
	timeDir := filepath.Join(a.outDir, timeDirName)
	if err := os.MkdirAll(timeDir, 0755); err != nil {
		dialog.ShowError(fmt.Errorf("创建输出目录失败: %v", err), a.myWindow)
		return
	}

	a.outExcelFile = filepath.Join(timeDir, fmt.Sprintf("diff_excel_%s.xlsx", timeNow))
	a.outLogFile = filepath.Join(timeDir, fmt.Sprintf("diff_log_%s.txt", timeNow))

	// 创建进度对话框
	progressDlg := dialog.NewCustom("正在对比", "取消（后台继续）",
		container.NewVBox(
			widget.NewLabel("正在对比Excel文件，请稍候..."),
			widget.NewProgressBarInfinite(),
		),
		a.myWindow,
	)

	a.isComparing = true
	progressDlg.Show()

	// 在goroutine中执行对比
	go func() {
		defer func() { a.isComparing = false }()

		startTime := time.Now()
		a.appendLog("\n====== [" + time.Now().Format("2006.01.02 15:04:05") + "] 开始======\n")

		var err error
		if a.multiSheetMode {
			err = a.CompareMultipleSheets()
		} else {
			// 使用灵活多Sheet对比功能
			err = a.CompareFlexibleSheetPairs()
		}

		elapsed := time.Since(startTime)

		// 关闭进度对话框
		progressDlg.Hide()

		if err != nil {
			a.appendLog(fmt.Sprintf("错误：%v\n", err))
			dialog.ShowError(err, a.myWindow)
		} else {
			a.appendLog(fmt.Sprintf("Excel文件: %s\n日志文件: %s\n", a.outExcelFile, a.outLogFile))
			a.appendLog(fmt.Sprintf("耗时: %.2f秒\n", elapsed.Seconds()))
			a.appendLog("====== [" + time.Now().Format("2006.01.02 15:04:05") + "] 结束======\n\n")
			a.Success()
		}
	}()
}

// Success 显示对比完成的成功对话框
func (a *ExcelCompareApp) Success() {
	content := container.NewVBox(
		// widget.NewLabel("对比完成！输出文件："),
		widget.NewHyperlink(a.outExcelFile, nil),
		widget.NewHyperlink(a.outLogFile, nil),
		widget.NewActivity(),
		container.NewHBox(
			widget.NewButton("打开文件", func() { utils.OpenFile(a.outExcelFile) }),
			widget.NewButton("打开文件所在目录", func() { utils.OpenDir(a.outDir) }),
		),
	)
	dialog.ShowCustom("对比完成", "关闭", content, a.myWindow)
}

// Run 运行应用程序
func (a *ExcelCompareApp) Run() {
	a.myWindow.ShowAndRun()
}
