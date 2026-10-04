package fetch

import "testing"

func TestValidateURLRejectsUnsafe(t *testing.T) {
	cases := []string{
		"http://127.0.0.1/",
		"http://localhost/",
		"http://10.0.0.1/",
		"http://192.168.1.1/",
		"http://169.254.1.1/",
		"file:///etc/passwd",
		"ftp://example.com/",
		"",
	}
	for _, u := range cases {
		if err := ValidateURL(u); err == nil {
			t.Errorf("期望拒绝 %q，但通过了校验", u)
		}
	}
}

func TestValidateURLAcceptsPublic(t *testing.T) {
	if err := ValidateURL("http://example.com/"); err != nil {
		t.Errorf("期望接受 http://example.com/，实际: %v", err)
	}
	if err := ValidateURL("https://example.com/path?q=1"); err != nil {
		t.Errorf("期望接受 https://example.com/path，实际: %v", err)
	}
}

func TestGetURLPath(t *testing.T) {
	cases := []struct {
		ref, base, want string
	}{
		{"/favicon.ico", "http://a.com/page", "http://a.com/favicon.ico"},
		{"icon.png", "http://a.com/dir/page", "http://a.com/dir/icon.png"},
		{"//cdn.a.com/i.png", "https://a.com/", "https://cdn.a.com/i.png"},
		{"http://b.com/x.png", "http://a.com/", "http://b.com/x.png"},
	}
	for _, c := range cases {
		if got := GetURLPath(c.ref, c.base); got != c.want {
			t.Errorf("GetURLPath(%q,%q)=%q, 期望 %q", c.ref, c.base, got, c.want)
		}
	}
}

func TestGetHeadInvalidURL(t *testing.T) {
	res := GetHead("not-a-url", t.TempDir(), false)
	if code, _ := res["code"].(int); code == 200 {
		t.Errorf("非法 URL 不应返回 200: %#v", res)
	}
}
