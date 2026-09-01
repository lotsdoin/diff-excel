package utils

import "testing"

// NormalizeColorCode：#RGB 简写必须展开为 #RRGGBB（excelize 只接受 6 位色值），
// 已是 6 位或非简写输入原样返回。
func TestNormalizeColorCode(t *testing.T) {
	cases := []struct{ in, want string }{
		{"#F00", "#FF0000"},
		{"#0aB", "#00aaBB"},
		{"#FF0000", "#FF0000"},
		{"#123456", "#123456"},
	}
	for _, c := range cases {
		if got := NormalizeColorCode(c.in); got != c.want {
			t.Errorf("NormalizeColorCode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
