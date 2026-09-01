package main

import "testing"

// 回归测试：updateSheetMappings 的一对一映射必须按 sheet 在文件中的顺序确定性配对，
// 不受 Go map 迭代序随机化影响（多跑几轮放大随机性）。
func TestUpdateSheetMappingsDeterministicOrder(t *testing.T) {
	a := ExcelCompareApp{
		srcSheets: []string{"S1", "S2", "S3"},
		cmpSheets: []string{"C1", "C2", "C3"},
		selectedSrcSheets: map[string]bool{
			"S1": true, "S2": true, "S3": true,
		},
		selectedCmpSheets: map[string]bool{
			"C1": true, "C2": true, "C3": true,
		},
	}

	want := map[string]string{"S1": "C1", "S2": "C2", "S3": "C3"}
	for i := 0; i < 50; i++ {
		a.updateSheetMappings()
		if len(a.sheetMappings) != len(want) {
			t.Fatalf("第 %d 轮映射数错误: got %v", i, a.sheetMappings)
		}
		for src, cmp := range want {
			if a.sheetMappings[src] != cmp {
				t.Fatalf("第 %d 轮映射乱序: got %v, want %v", i, a.sheetMappings, want)
			}
		}
	}
}
