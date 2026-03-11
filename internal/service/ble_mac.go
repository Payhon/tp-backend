package service

import (
	"strings"
	"unicode"
)

// normalizeBleMac12ForStore 归一化 BLE MAC（12 位 HEX，无分隔符）。
// 兼容输入中出现分隔符、0x 前缀，以及尾部补零（例如 "...:00:00:00:00"）。
func normalizeBleMac12ForStore(input string) (string, bool) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", false
	}
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		s = s[2:]
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	hex := b.String()
	if len(hex) < 12 {
		return "", false
	}
	for len(hex) > 12 && strings.HasSuffix(hex, "00") {
		hex = hex[:len(hex)-2]
	}
	if len(hex) != 12 {
		return "", false
	}
	return hex, true
}
