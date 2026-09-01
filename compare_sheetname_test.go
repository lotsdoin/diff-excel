package main

import (
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"

	"github.com/xuri/excelize/v2"
)

// safeSheetName 单元测试：超长截断、冲突加后缀、合法名原样保留
func TestSafeSheetName(t *testing.T) {
	used := make(map[string]bool)

	// 合法名原样保留
	if got := safeSheetName("Sheet1", used); got != "Sheet1" {
		t.Errorf("合法名被改写: %q", got)
	}

	// 超长名截断到 31 个 UTF-16 单元
	long := "【新】2026年6月销售明细汇总 <<【旧】2026年5月销售明细汇总"
	got := safeSheetName(long, used)
	if n := len(utf16.Encode([]rune(got))); n > 31 {
		t.Errorf("截断后仍超长: %d 单元, %q", n, got)
	}

	// 同名再来一次，必须去重且不超长
	got2 := safeSheetName(long, used)
	if got2 == got {
		t.Errorf("冲突未去重: %q", got2)
	}
	if n := len(utf16.Encode([]rune(got2))); n > 31 {
		t.Errorf("去重后超长: %d 单元, %q", n, got2)
	}
}

// 回归测试：sheet 名超长时，CompareFlexibleSheetPairs 不得静默输出空文件
func TestCompareFlexibleSheetPairsLongSheetName(t *testing.T) {
	tmpDir := t.TempDir()
	longSheet := "2026年6月销售明细汇总数据表" // DisplayName 加前缀后必超 31 字符

	makeFile := func(name, cellVal string) string {
		f := excelize.NewFile()
		defer f.Close()
		if err := f.SetSheetName("Sheet1", longSheet); err != nil {
			t.Fatalf("准备测试文件失败: %v", err)
		}
		if err := f.SetCellValue(longSheet, "A1", cellVal); err != nil {
			t.Fatalf("准备测试文件失败: %v", err)
		}
		p := filepath.Join(tmpDir, name)
		if err := f.SaveAs(p); err != nil {
			t.Fatalf("保存测试文件失败: %v", err)
		}
		return p
	}

	a := ExcelCompareApp{}
	a.srcFile = makeFile("src.xlsx", "old")
	a.cmpFile = makeFile("cmp.xlsx", "new")
	a.outExcelFile = filepath.Join(tmpDir, "out.xlsx")
	a.outLogFile = filepath.Join(tmpDir, "out.txt")
	a.highlightClr = "#FF0000"
	a.sheetPairs = []SheetPair{{
		SrcSheet:    longSheet,
		CmpSheet:    longSheet,
		DisplayName: "【新】" + longSheet + " <<【旧】" + longSheet,
	}}

	if err := a.CompareFlexibleSheetPairs(); err != nil {
		t.Fatalf("CompareFlexibleSheetPairs failed: %v", err)
	}

	if _, err := os.Stat(a.outExcelFile); err != nil {
		t.Fatalf("输出 Excel 文件未生成: %v", err)
	}
	out, err := excelize.OpenFile(a.outExcelFile)
	if err != nil {
		t.Fatalf("打开输出文件失败: %v", err)
	}
	defer out.Close()

	sheets := out.GetSheetList()
	if len(sheets) != 1 {
		t.Fatalf("输出 sheet 数不为 1: %v", sheets)
	}
	val, err := out.GetCellValue(sheets[0], "A1")
	if err != nil {
		t.Fatalf("读取输出单元格失败: %v", err)
	}
	if val != "new" {
		t.Errorf("差异单元格未写入输出（静默失败回归）: A1=%q", val)
	}
}
