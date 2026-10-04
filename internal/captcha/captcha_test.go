package captcha

import (
	"bytes"
	"testing"
)

func TestNewCodeIsFourDigits(t *testing.T) {
	for i := 0; i < 50; i++ {
		code := NewCode()
		if len(code) != 4 {
			t.Fatalf("验证码长度应为 4，实际 %q", code)
		}
		for _, c := range code {
			if c < '0' || c > '9' {
				t.Fatalf("验证码应仅包含数字，实际 %q", code)
			}
		}
	}
}

func TestGenerateWritesPNG(t *testing.T) {
	var buf bytes.Buffer
	if err := Generate(&buf, "1234"); err != nil {
		t.Fatalf("生成验证码失败: %v", err)
	}
	b := buf.Bytes()
	if len(b) < 8 {
		t.Fatalf("输出过短: %d 字节", len(b))
	}
	// PNG 魔数
	pngMagic := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	if !bytes.Equal(b[:8], pngMagic) {
		t.Fatalf("输出不是合法 PNG: % x", b[:8])
	}
}
