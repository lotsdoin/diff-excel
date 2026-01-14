package main

import (
	"diff_excel/utils"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// CompareExcelFiles 对比Excel文件
func (a *ExcelCompareApp) CompareExcelFiles() error {
	src, err := excelize.OpenFile(a.srcFile)
	if err != nil {
		a.appendLog(fmt.Sprintf("打开旧 Excel 出错: %v", err))
		return fmt.Errorf("打开旧 Excel 出错: %v", err)
	}
	defer src.Close()

	cmp, err := excelize.OpenFile(a.cmpFile)
	if err != nil {
		a.appendLog(fmt.Sprintf("打开新 Excel 出错: %v", err))
		return fmt.Errorf("打开新 Excel 出错: %v", err)
	}
	defer cmp.Close()

	srcRows, err := src.GetRows(a.srcSheet)
	if err != nil {
		a.appendLog(fmt.Sprintf("读取旧 Excel 行失败: %v", err))
		return fmt.Errorf("读取旧 Excel 行失败: %v", err)
	}
	cmpRows, err := cmp.GetRows(a.cmpSheet)
	if err != nil {
		a.appendLog(fmt.Sprintf("读取新 Excel 行失败: %v", err))
		return fmt.Errorf("读取新 Excel 行失败: %v", err)
	}

	// 创建输出文件：如果需要保持原格式，则复制原文件；否则创建新文件
	var diffF *excelize.File
	if a.keepOriginalFormat {
		// 复制原文件作为输出文件的基础
		a.appendLog("正在复制原文件格式...\n")
		if err := utils.CopyFile(a.srcFile, a.outExcelFile); err != nil {
			a.appendLog(fmt.Sprintf("复制原文件失败: %v", err))
			return fmt.Errorf("复制原文件失败: %v", err)
		}
		diffF, err = excelize.OpenFile(a.outExcelFile)
		if err != nil {
			a.appendLog(fmt.Sprintf("打开复制的文件失败: %v", err))
			return fmt.Errorf("打开复制的文件失败: %v", err)
		}
	} else {
		diffF = excelize.NewFile()
	}
	defer diffF.Close()

	diffSheet := a.srcSheet
	if !a.keepOriginalFormat {
		diffSheet = diffF.GetSheetName(0)
	}

	// 创建差异高亮样式
	style := &excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{a.highlightClr},
		},
	}
	styleID, err := diffF.NewStyle(style)
	if err != nil {
		a.appendLog(fmt.Sprintf("创建样式失败: %v", err))
		return fmt.Errorf("创建样式失败: %v", err)
	}

	maxRow := len(srcRows)
	diffMaxRow := len(cmpRows)
	if diffMaxRow > maxRow {
		maxRow = diffMaxRow
	}

	a.appendLog(fmt.Sprintf("【原始文件】%s 行数据\n", strconv.Itoa(maxRow)))
	a.appendLog(fmt.Sprintf("【对比文件】%s 行数据\n", strconv.Itoa(diffMaxRow)))

	var logBuilder strings.Builder

	a.appendLog("\n\n --------- 差异单元格 --------- \n")
	diffCount := 0
	for r := 0; r < maxRow; r++ {
		maxCol := 0
		if r < len(srcRows) && len(srcRows[r]) > maxCol {
			maxCol = len(srcRows[r])
		}
		if r < len(cmpRows) && len(cmpRows[r]) > maxCol {
			maxCol = len(cmpRows[r])
		}

		for c := 0; c < maxCol; c++ {
			var oldVal, newVal string
			if r < len(srcRows) && c < len(srcRows[r]) {
				oldVal = srcRows[r][c]
			}
			if r < len(cmpRows) && c < len(cmpRows[r]) {
				newVal = cmpRows[r][c]
			}

			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			if oldVal != newVal {
				diffCount++
				a.appendLog(fmt.Sprintf(" %s |", cell))
				logLine := fmt.Sprintf("差异单元格: %s 旧数据: %s 新数据: %s\n", cell, oldVal, newVal)
				logBuilder.WriteString(logLine)

				// 设置新值
				diffF.SetCellValue(diffSheet, cell, newVal)

				// 设置差异高亮样式
				if a.keepOriginalFormat {
					// 获取原单元格样式
					originalStyleID, _ := src.GetCellStyle(a.srcSheet, cell)
					if originalStyleID != 0 {
						// 复制原样式并添加高亮颜色
						originalStyle, _ := src.GetStyle(originalStyleID)
						if originalStyle != nil {
							// 保持原有样式，只修改背景颜色
							originalStyle.Fill = excelize.Fill{
								Type:    "pattern",
								Pattern: 1,
								Color:   []string{a.highlightClr},
							}
							combinedStyleID, _ := diffF.NewStyle(originalStyle)
							diffF.SetCellStyle(diffSheet, cell, cell, combinedStyleID)
						} else {
							diffF.SetCellStyle(diffSheet, cell, cell, styleID)
						}
					} else {
						diffF.SetCellStyle(diffSheet, cell, cell, styleID)
					}
				} else {
					diffF.SetCellStyle(diffSheet, cell, cell, styleID)
				}

				// 添加备注
				if a.showOldInComment && oldVal != "" {
					_ = diffF.AddComment(diffSheet, excelize.Comment{
						Cell:   cell,
						Author: "Diff Excel",
						Paragraph: []excelize.RichTextRun{
							{Text: "旧数据: \n", Font: &excelize.Font{Bold: true, Color: "#6c0808ff"}},
							{Text: oldVal},
						},
						Height: 40,
						Width:  180,
					})
				}
			} else {
				// 如果没有差异，且不保持原格式，则设置新值
				if !a.keepOriginalFormat {
					diffF.SetCellValue(diffSheet, cell, newVal)
				}
			}
		}
	}
	a.appendLog(fmt.Sprintf("\n\n --------- 差异数：%v -------- \n", diffCount))

	// 保存文件
	if a.keepOriginalFormat {
		// 如果保持原格式，文件已经存在，只需要保存修改
		if err := diffF.Save(); err != nil {
			a.appendLog(fmt.Sprintf("保存差异 Excel 文件失败: %v", err))
			return fmt.Errorf("保存差异 Excel 文件失败: %v", err)
		}
	} else {
		// 如果不保持原格式，则另存为新文件
		if err := diffF.SaveAs(a.outExcelFile); err != nil {
			a.appendLog(fmt.Sprintf("保存差异 Excel 文件失败: %v", err))
			return fmt.Errorf("保存差异 Excel 文件失败: %v", err)
		}
	}
	err = os.WriteFile(a.outLogFile, []byte(logBuilder.String()), 0644)
	if err != nil {
		a.appendLog(fmt.Sprintf("写入日志 TXT 文件失败: %v", err))
		return fmt.Errorf("写入日志 TXT 文件失败: %v", err)
	}
	return nil
}

// CompareMultipleSheets 对比多个Sheet
func (a *ExcelCompareApp) CompareMultipleSheets() error {
	src, err := excelize.OpenFile(a.srcFile)
	if err != nil {
		a.appendLog(fmt.Sprintf("打开源 Excel 出错: %v", err))
		return fmt.Errorf("打开源 Excel 出错: %v", err)
	}
	defer src.Close()

	cmp, err := excelize.OpenFile(a.cmpFile)
	if err != nil {
		a.appendLog(fmt.Sprintf("打开对比 Excel 出错: %v", err))
		return fmt.Errorf("打开对比 Excel 出错: %v", err)
	}
	defer cmp.Close()

	// 创建输出文件
	var diffF *excelize.File
	if a.keepOriginalFormat {
		// 复制原文件作为基础
		a.appendLog("正在复制原文件格式...\n")
		if err := utils.CopyFile(a.srcFile, a.outExcelFile); err != nil {
			a.appendLog(fmt.Sprintf("复制原文件失败: %v", err))
			return fmt.Errorf("复制原文件失败: %v", err)
		}
		diffF, err = excelize.OpenFile(a.outExcelFile)
		if err != nil {
			a.appendLog(fmt.Sprintf("打开复制的文件失败: %v", err))
			return fmt.Errorf("打开复制的文件失败: %v", err)
		}
	} else {
		diffF = excelize.NewFile()
	}
	defer diffF.Close()

	// 创建差异高亮样式
	style := &excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{a.highlightClr},
		},
	}
	styleID, err := diffF.NewStyle(style)
	if err != nil {
		a.appendLog(fmt.Sprintf("创建样式失败: %v", err))
		return fmt.Errorf("创建样式失败: %v", err)
	}

	// 样式缓存：避免重复创建相同样式（优化性能）
	styleCache := make(map[int]int)

	var allLogBuilder strings.Builder
	totalDiffCount := 0

	a.appendLog(fmt.Sprintf("开始对比 %d 个Sheet...\n", len(a.sheetMappings)))

	// 对比每个映射的Sheet
	for srcSheetName, cmpSheetName := range a.sheetMappings {
		a.appendLog(fmt.Sprintf("\n正在对比 %s → %s ...\n", srcSheetName, cmpSheetName))

		// 读取Sheet数据
		srcRows, err := src.GetRows(srcSheetName)
		if err != nil {
			a.appendLog(fmt.Sprintf("读取源 Excel Sheet '%s' 失败: %v\n", srcSheetName, err))
			continue
		}
		cmpRows, err := cmp.GetRows(cmpSheetName)
		if err != nil {
			a.appendLog(fmt.Sprintf("读取对比 Excel Sheet '%s' 失败: %v\n", cmpSheetName, err))
			continue
		}

		// 确定输出 Sheet 名称
		diffSheetName := srcSheetName
		if !a.keepOriginalFormat {
			// 如果不保持格式，则创建新Sheet（除了第一个Sheet）
			if srcSheetName != "Sheet1" {
				diffF.NewSheet(srcSheetName)
			}
			diffSheetName = srcSheetName
		}

		// 对比单元格 - 收集差异单元格批量处理
		sheetDiffCount := 0
		var sheetLogBuilder strings.Builder
		var diffCells []struct {
			cell    string
			oldVal  string
			newVal  string
			styleID int
		}

		maxRow := len(srcRows)
		if len(cmpRows) > maxRow {
			maxRow = len(cmpRows)
		}

		for r := 0; r < maxRow; r++ {
			maxCol := 0
			if r < len(srcRows) && len(srcRows[r]) > maxCol {
				maxCol = len(srcRows[r])
			}
			if r < len(cmpRows) && len(cmpRows[r]) > maxCol {
				maxCol = len(cmpRows[r])
			}

			for c := 0; c < maxCol; c++ {
				var oldVal, newVal string
				if r < len(srcRows) && c < len(srcRows[r]) {
					oldVal = srcRows[r][c]
				}
				if r < len(cmpRows) && c < len(cmpRows[r]) {
					newVal = cmpRows[r][c]
				}

				cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
				if oldVal != newVal {
					sheetDiffCount++
					logLine := fmt.Sprintf("[%s] %s: %s → %s\n", srcSheetName, cell, oldVal, newVal)
					sheetLogBuilder.WriteString(logLine)

					var finalStyleID = styleID

					// 设置高亮样式（使用缓存优化）
					if a.keepOriginalFormat {
						originalStyleID, _ := src.GetCellStyle(srcSheetName, cell)
						if originalStyleID != 0 {
							// 检查缓存
							if cachedStyleID, ok := styleCache[originalStyleID]; ok {
								finalStyleID = cachedStyleID
							} else {
								originalStyle, _ := src.GetStyle(originalStyleID)
								if originalStyle != nil {
									originalStyle.Fill = excelize.Fill{
										Type:    "pattern",
										Pattern: 1,
										Color:   []string{a.highlightClr},
									}
									combinedStyleID, _ := diffF.NewStyle(originalStyle)
									styleCache[originalStyleID] = combinedStyleID
									finalStyleID = combinedStyleID
								}
							}
						}
					}

					diffCells = append(diffCells, struct {
						cell    string
						oldVal  string
						newVal  string
						styleID int
					}{
						cell:    cell,
						oldVal:  oldVal,
						newVal:  newVal,
						styleID: finalStyleID,
					})
				} else {
					// 如果没有差异，且不保持原格式，则设置新值
					if !a.keepOriginalFormat {
						diffF.SetCellValue(diffSheetName, cell, newVal)
					}
				}
			}
		}

		// 批量应用差异单元格修改
		for _, dc := range diffCells {
			// 设置新值
			diffF.SetCellValue(diffSheetName, dc.cell, dc.newVal)
			// 设置高亮样式
			diffF.SetCellStyle(diffSheetName, dc.cell, dc.cell, dc.styleID)

			// 添加备注（如果需要）
			if a.showOldInComment && dc.oldVal != "" {
				_ = diffF.AddComment(diffSheetName, excelize.Comment{
					Cell:   dc.cell,
					Author: "Diff Excel",
					Paragraph: []excelize.RichTextRun{
						{Text: "旧数据: \n", Font: &excelize.Font{Bold: true, Color: "#6c0808ff"}},
						{Text: dc.oldVal},
					},
					Height: 40,
					Width:  180,
				})
			}
		}

		totalDiffCount += sheetDiffCount
		a.appendLog(fmt.Sprintf("Sheet '%s' 差异数: %d\n", srcSheetName, sheetDiffCount))
		allLogBuilder.WriteString(fmt.Sprintf("\n=== Sheet: %s ===\n", srcSheetName))
		allLogBuilder.WriteString(sheetLogBuilder.String())
	}

	a.appendLog(fmt.Sprintf("\n\n --------- 总差异数：%d -------- \n", totalDiffCount))

	// 保存文件
	if a.keepOriginalFormat {
		if err := diffF.Save(); err != nil {
			a.appendLog(fmt.Sprintf("保存差异 Excel 文件失败: %v", err))
			return fmt.Errorf("保存差异 Excel 文件失败: %v", err)
		}
	} else {
		if err := diffF.SaveAs(a.outExcelFile); err != nil {
			a.appendLog(fmt.Sprintf("保存差异 Excel 文件失败: %v", err))
			return fmt.Errorf("保存差异 Excel 文件失败: %v", err)
		}
	}

	// 写入日志文件
	err = os.WriteFile(a.outLogFile, []byte(allLogBuilder.String()), 0644)
	if err != nil {
		a.appendLog(fmt.Sprintf("写入日志 TXT 文件失败: %v", err))
		return fmt.Errorf("写入日志 TXT 文件失败: %v", err)
	}

	return nil
}

// CompareFlexibleSheetPairs 灵活的多Sheet对比功能，支持手动选择多对Sheet进行对比（优化版）
func (a *ExcelCompareApp) CompareFlexibleSheetPairs() error {
	if len(a.sheetPairs) == 0 {
		return fmt.Errorf("没有设置Sheet对比对")
	}

	src, err := excelize.OpenFile(a.srcFile)
	if err != nil {
		a.appendLog(fmt.Sprintf("打开源 Excel 出错: %v", err))
		return fmt.Errorf("打开源 Excel 出错: %v", err)
	}
	defer src.Close()

	cmp, err := excelize.OpenFile(a.cmpFile)
	if err != nil {
		a.appendLog(fmt.Sprintf("打开对比 Excel 出错: %v", err))
		return fmt.Errorf("打开对比 Excel 出错: %v", err)
	}
	defer cmp.Close()

	// 创建输出文件
	diffF := excelize.NewFile()
	defer diffF.Close()

	// 创建差异高亮样式
	style := &excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{a.highlightClr},
		},
	}
	styleID, err := diffF.NewStyle(style)
	if err != nil {
		a.appendLog(fmt.Sprintf("创建样式失败: %v", err))
		return fmt.Errorf("创建样式失败: %v", err)
	}

	// 全局样式缓存：跨所有 Sheet 复用样式
	globalStyleCache := make(map[int]int)

	a.appendLog(fmt.Sprintf("开始灵活对比 %d 个Sheet对..\n", len(a.sheetPairs)))

	var allLogBuilder strings.Builder
	totalDiffCount := 0
	firstSheet := true

	// 逐个处理 Sheet 对
	for _, pair := range a.sheetPairs {
		a.appendLog(fmt.Sprintf("\n正在对比: %s ...\n", pair.DisplayName))

		// 读取 Sheet 数据
		srcRows, err := src.GetRows(pair.SrcSheet)
		if err != nil {
			a.appendLog(fmt.Sprintf("读取源 Excel Sheet '%s' 失败: %v\n", pair.SrcSheet, err))
			continue
		}
		cmpRows, err := cmp.GetRows(pair.CmpSheet)
		if err != nil {
			a.appendLog(fmt.Sprintf("读取对比 Excel Sheet '%s' 失败: %v\n", pair.CmpSheet, err))
			continue
		}

		// 确定输出 Sheet 名称
		diffSheetName := pair.DisplayName

		// 创建输出 Sheet
		if firstSheet {
			diffF.SetSheetName("Sheet1", diffSheetName)
			firstSheet = false
		} else {
			diffF.NewSheet(diffSheetName)
		}

		// 如果启用格式保持，复制 Sheet 格式和数据
		if a.keepOriginalFormat {
			if err := a.copySheetContentOptimized(cmp, pair.CmpSheet, diffF, diffSheetName); err != nil {
				a.appendLog(fmt.Sprintf("复制新文件 Sheet 格式失败: %v\n", err))
			}
		} else {
			// 不保持格式时，只填充基础数据
			for r, row := range cmpRows {
				for c, cellValue := range row {
					cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
					diffF.SetCellValue(diffSheetName, cell, cellValue)
				}
			}
		}

		// 对比单元格并收集差异
		sheetDiffCount := 0
		var sheetLogBuilder strings.Builder
		var diffCells []struct {
			cell            string
			newVal          string
			oldVal          string
			originalStyleID int
		}

		maxRow := len(srcRows)
		if len(cmpRows) > maxRow {
			maxRow = len(cmpRows)
		}

		for r := 0; r < maxRow; r++ {
			maxCol := 0
			if r < len(srcRows) && len(srcRows[r]) > maxCol {
				maxCol = len(srcRows[r])
			}
			if r < len(cmpRows) && len(cmpRows[r]) > maxCol {
				maxCol = len(cmpRows[r])
			}

			for c := 0; c < maxCol; c++ {
				var oldVal, newVal string
				if r < len(srcRows) && c < len(srcRows[r]) {
					oldVal = srcRows[r][c]
				}
				if r < len(cmpRows) && c < len(cmpRows[r]) {
					newVal = cmpRows[r][c]
				}

				if oldVal != newVal {
					sheetDiffCount++
					cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
					logLine := fmt.Sprintf("[%s] %s: %s → %s\n", pair.DisplayName, cell, oldVal, newVal)
					sheetLogBuilder.WriteString(logLine)

					var originalStyleID int
					if a.keepOriginalFormat {
						originalStyleID, _ = cmp.GetCellStyle(pair.CmpSheet, cell)
					}

					diffCells = append(diffCells, struct {
						cell            string
						newVal          string
						oldVal          string
						originalStyleID int
					}{
						cell:            cell,
						newVal:          newVal,
						oldVal:          oldVal,
						originalStyleID: originalStyleID,
					})
				}
			}
		}

		// 批量应用差异单元格修改
		for _, dc := range diffCells {
			// 设置新值
			diffF.SetCellValue(diffSheetName, dc.cell, dc.newVal)

			// 设置高亮样式
			finalStyleID := styleID
			if a.keepOriginalFormat && dc.originalStyleID != 0 {
				// 检查全局样式缓存
				if cachedStyleID, ok := globalStyleCache[dc.originalStyleID]; ok {
					finalStyleID = cachedStyleID
				} else {
					// 创建新的合并样式
					originalStyle, _ := cmp.GetStyle(dc.originalStyleID)
					if originalStyle != nil {
						originalStyle.Fill = excelize.Fill{
							Type:    "pattern",
							Pattern: 1,
							Color:   []string{a.highlightClr},
						}
						combinedStyleID, _ := diffF.NewStyle(originalStyle)
						globalStyleCache[dc.originalStyleID] = combinedStyleID
						finalStyleID = combinedStyleID
					}
				}
			}
			diffF.SetCellStyle(diffSheetName, dc.cell, dc.cell, finalStyleID)

			// 添加备注（如果需要）
			if a.showOldInComment && dc.oldVal != "" {
				_ = diffF.AddComment(diffSheetName, excelize.Comment{
					Cell:   dc.cell,
					Author: "Diff Excel",
					Paragraph: []excelize.RichTextRun{
						{Text: "旧数据: \n", Font: &excelize.Font{Bold: true, Color: "#6c0808ff"}},
						{Text: dc.oldVal},
					},
					Height: 40,
					Width:  180,
				})
			}
		}

		totalDiffCount += sheetDiffCount
		a.appendLog(fmt.Sprintf("Sheet对 '%s' 差异数: %d\n", pair.DisplayName, sheetDiffCount))
		allLogBuilder.WriteString(fmt.Sprintf("\n=== %s ===\n", pair.DisplayName))
		allLogBuilder.WriteString(sheetLogBuilder.String())
	}

	a.appendLog(fmt.Sprintf("\n\n --------- 总差异数：%d -------- \n", totalDiffCount))

	// 保存文件
	if err := diffF.SaveAs(a.outExcelFile); err != nil {
		a.appendLog(fmt.Sprintf("保存差异 Excel 文件失败: %v", err))
		return fmt.Errorf("保存差异 Excel 文件失败: %v", err)
	}

	// 写入日志文件
	err = os.WriteFile(a.outLogFile, []byte(allLogBuilder.String()), 0644)
	if err != nil {
		a.appendLog(fmt.Sprintf("写入日志 TXT 文件失败: %v", err))
		return fmt.Errorf("写入日志 TXT 文件失败: %v", err)
	}

	return nil
}

// copySheetContentOptimized 优化版：复制Sheet内容到新Sheet（减少重复操作）
func (a *ExcelCompareApp) copySheetContentOptimized(srcFile *excelize.File, srcSheet string, dstFile *excelize.File, dstSheet string) error {
	// 读取源Sheet的所有数据
	rows, err := srcFile.GetRows(srcSheet)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		return nil
	}

	// 计算实际使用的列数（只处理有数据的列）
	maxCol := 0
	for _, row := range rows {
		if len(row) > maxCol {
			maxCol = len(row)
		}
	}

	// 复制列宽（只复制实际使用的列）
	for c := 0; c < maxCol; c++ {
		colName, _ := excelize.ColumnNumberToName(c + 1)
		width, _ := srcFile.GetColWidth(srcSheet, colName)
		if width > 0 {
			dstFile.SetColWidth(dstSheet, colName, colName, width)
		}
	}

	// 复制行高
	for r := 1; r <= len(rows); r++ {
		height, _ := srcFile.GetRowHeight(srcSheet, r)
		if height > 0 {
			dstFile.SetRowHeight(dstSheet, r, height)
		}
	}

	// 复制合并单元格
	mergedCells, _ := srcFile.GetMergeCells(srcSheet)
	for _, mergeCell := range mergedCells {
		dstFile.MergeCell(dstSheet, mergeCell.GetStartAxis(), mergeCell.GetEndAxis())
	}

	// 样式缓存：避免重复创建相同样式
	styleCache := make(map[int]int)

	// 复制数据到目标Sheet
	for r, row := range rows {
		for c := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			cellValue := row[c]

			// 设置单元格值
			dstFile.SetCellValue(dstSheet, cell, cellValue)

			// 复制样式（使用缓存）
			styleID, _ := srcFile.GetCellStyle(srcSheet, cell)
			if styleID != 0 {
				// 检查缓存
				if newStyleID, ok := styleCache[styleID]; ok {
					dstFile.SetCellStyle(dstSheet, cell, cell, newStyleID)
				} else {
					// 创建新样式并缓存
					style, _ := srcFile.GetStyle(styleID)
					if style != nil {
						newStyleID, _ := dstFile.NewStyle(style)
						styleCache[styleID] = newStyleID
						dstFile.SetCellStyle(dstSheet, cell, cell, newStyleID)
					}
				}
			}
		}
	}

	return nil
}
