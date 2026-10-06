package logger

import (
	"strings"
	"testing"
)

func TestMaskPassArg(t *testing.T) {
	got := Mask(`apksigner sign --ks-pass pass:123456 --out a.apk`)
	if strings.Contains(got, "123456") {
		t.Fatalf("apksigner 密码未脱敏: %s", got)
	}
	if !strings.Contains(got, "pass:") {
		t.Fatalf("应保留参数名便于排查: %s", got)
	}
}

func TestMaskKeytoolStorePass(t *testing.T) {
	got := Mask(`-list -v -keystore demo.jks -storepass my-secret -keypass other-pass`)
	if strings.Contains(got, "my-secret") || strings.Contains(got, "other-pass") {
		t.Fatalf("keytool 密码未脱敏: %s", got)
	}
}

func TestMaskExplicitSecrets(t *testing.T) {
	got := Mask("使用 changeit 连接 keystore", "changeit")
	if strings.Contains(got, "changeit") {
		t.Fatalf("显式密文未脱敏: %s", got)
	}
	if got != "使用 ****** 连接 keystore" {
		t.Fatalf("脱敏占位符不符: %s", got)
	}
}

func TestMaskKeepsNormalText(t *testing.T) {
	text := "apksigner verify --print-certs demo.apk"
	if got := Mask(text); got != text {
		t.Fatalf("普通命令不应被改写: %s", got)
	}
}

func TestRecentRing(t *testing.T) {
	for i := 0; i < 5; i++ {
		Info("message")
	}
	if got := Recent(0); len(got) != 5 {
		t.Fatalf("Recent(0) 应返回全部日志，实际 %d", len(got))
	}
	if got := Recent(2); len(got) != 2 {
		t.Fatalf("Recent(2) 应返回 2 条，实际 %d", len(got))
	}
}

func TestRecentKeepsLatestOrder(t *testing.T) {
	Info("first")
	Info("second")
	got := Recent(2)
	if got[0].Message != "first" || got[1].Message != "second" {
		t.Fatalf("日志顺序错误: %+v", got)
	}
}
