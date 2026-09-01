package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func OpenFile(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default: // Linux
		cmd = exec.Command("xdg-open", path)
	}
	cmd.Start()
}

func OpenDir(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	case "windows":
		cmd = exec.Command("explorer", "/select,", path)
	default: // Linux
		cmd = exec.Command("xdg-open", path)
	}
	cmd.Start()
}

func GetExeDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exePath)
	return dir, nil
}

// NormalizeColorCode 把 #RGB 简写展开为 #RRGGBB。
// excelize 的 fill 颜色是 "FF"+hex 直拼成 ARGB，3 位简写会生成 5 位非法色值，
// 高亮静默失效，因此写入样式前必须展开为 6 位。
func NormalizeColorCode(s string) string {
	if len(s) == 4 && s[0] == '#' {
		return string([]byte{'#', s[1], s[1], s[2], s[2], s[3], s[3]})
	}
	return s
}

func IsValidColorCode(s string) bool {
	if len(s) != 7 && len(s) != 4 {
		return false
	}
	if s[0] != '#' {
		return false
	}
	for _, c := range s[1:] {
		if !strings.Contains("0123456789abcdefABCDEF", string(c)) {
			return false
		}
	}
	return true
}
