// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

package signer

import (
	"strings"
	"testing"

	"go-apksigner-gui/internal/dto"
)

const signedOutput = `Verifies
Verified using v1 scheme (JAR signing): true
Verified using v2 scheme (APK Signature Scheme v2): true
Verified using v3 scheme (APK Signature Scheme v3): false
Number of signers: 1
Signer #1 certificate DN: CN=Demo App, OU=Dev, O=Demo, L=Beijing, ST=Beijing, C=CN
Signer #1 certificate SHA-256 digest: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
Signer #1 certificate SHA-1 digest: bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
Signer #1 certificate MD5 digest: cccccccccccccccccccccccccccccccc
WARNING: META-INF/xxx not protected by signature. A cross-version verification failure could occur.`

const unsignedOutput = `DOES NOT VERIFY
ERROR: JAR signer DEMO.RSA: JAR signature META-INF/DEMO.SF indicates the APK is signed using APK Signature Scheme v2, but no such signature was found. Signature stripped?
Verified using v1 scheme (JAR signing): false
Verified using v2 scheme (APK Signature Scheme v2): NOT signed`

func TestBuildSignArgs(t *testing.T) {
	opts := dto.SignOptions{
		InputAPK:  "in.apk",
		OutputAPK: "out.apk",
		Keystore:  "demo.jks",
		Alias:     "demo-key",
		StorePass: "123456",
		KeyPass:   "654321",
		MinSDK:    24,
		MaxSDK:    34,
		V1Enabled: true,
		V2Enabled: true,
		V3Enabled: true,
	}
	args := BuildSignArgs(opts, "stage.apk")
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"sign",
		"--ks demo.jks",
		"--ks-key-alias demo-key",
		"--ks-pass pass:123456",
		"--key-pass pass:654321",
		"--out out.apk",
		"--min-sdk-version 24",
		"--max-sdk-version 34",
		"--v1-signing-enabled true",
		"--v2-signing-enabled true",
		"--v3-signing-enabled true",
		"--v4-signing-enabled false",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("参数缺少 %q，实际: %s", want, joined)
		}
	}
	if args[len(args)-1] != "stage.apk" {
		t.Fatalf("待签名 APK 必须作为最后一个参数传递，实际: %s", args[len(args)-1])
	}
}

func TestBuildSignArgsOmitsEmpty(t *testing.T) {
	opts := dto.SignOptions{InputAPK: "a.apk", Keystore: "k.jks", StorePass: "pwd"}
	joined := strings.Join(BuildSignArgs(opts, "a.apk"), " ")
	for _, unwanted := range []string{"--ks-key-alias", "--key-pass", "--out", "--min-sdk-version", "--max-sdk-version"} {
		if strings.Contains(joined, unwanted) {
			t.Fatalf("未填写的可选项不应出现在参数中: %s -> %s", unwanted, joined)
		}
	}
}

func TestBuildZipAlignArgs(t *testing.T) {
	args := BuildZipAlignArgs("a.apk", "b.apk")
	if len(args) != 5 || args[2] != "4" || args[3] != "a.apk" || args[4] != "b.apk" {
		t.Fatalf("zipalign 参数错误: %v", args)
	}
}

func TestBuildVerifyArgs(t *testing.T) {
	args := BuildVerifyArgs("a.apk", true)
	if strings.Join(args, " ") != "verify --verbose --print-certs a.apk" {
		t.Fatalf("verify 参数错误: %v", args)
	}
	args = BuildVerifyArgs("a.apk", false)
	if strings.Contains(strings.Join(args, " "), "--print-certs") {
		t.Fatal("未开启打印证书时不该带 --print-certs")
	}
}

func TestParseVerifyOutputSigned(t *testing.T) {
	res := ParseVerifyOutput(signedOutput)
	if !res.Verified {
		t.Fatal("Verifies 应判定为验证通过")
	}
	if !res.V1Signed || !res.V2Signed {
		t.Fatal("v1/v2 应为已签名")
	}
	if res.V3Signed {
		t.Fatal("v3 明确为 false，不应判定为已签名")
	}
	if len(res.Certs) != 1 {
		t.Fatalf("期望解析出 1 个签名证书，实际 %d", len(res.Certs))
	}
	if !strings.HasPrefix(res.Certs[0].SHA256, "aaaa") || res.Certs[0].SHA1 == "" || res.Certs[0].MD5 == "" {
		t.Fatalf("证书摘要解析错误: %+v", res.Certs[0])
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("应收集 1 条告警，实际 %d", len(res.Warnings))
	}
}

func TestParseVerifyOutputUnsigned(t *testing.T) {
	res := ParseVerifyOutput(unsignedOutput)
	if res.Verified {
		t.Fatal("DOES NOT VERIFY 应判定为未通过")
	}
	if res.V1Signed || res.V2Signed {
		t.Fatal("未签名 APK 不应标记为已签名")
	}
	if VerifySummary(res) != "无有效签名" {
		t.Fatalf("摘要应为「无有效签名」，实际 %q", VerifySummary(res))
	}
}

func TestVerifySummary(t *testing.T) {
	res := dto.VerifyResult{V1Signed: true, V2Signed: true, V3Signed: true}
	if got := VerifySummary(res); got != "v1+v2+v3" {
		t.Fatalf("摘要错误: %s", got)
	}
}

func TestDefaultOutputPath(t *testing.T) {
	cases := map[string]string{
		`C:\apps\demo.apk`:    `C:\apps\demo-signed.apk`,
		`/home/u/demo.apk`:    `/home/u/demo-signed.apk`,
		`/home/u/demo.apks`:   `/home/u/demo-signed.apks`,
		`/home/u/demo`:        `/home/u/demo-signed.apk`,
		`/home/u/1.0/a.b.apk`: `/home/u/1.0/a.b-signed.apk`,
	}
	for in, want := range cases {
		if got := DefaultOutputPath(in); got != want {
			t.Fatalf("DefaultOutputPath(%s) = %s，期望 %s", in, got, want)
		}
	}
	if got := DefaultOutputPath("  "); got != "" {
		t.Fatalf("空输入应返回空字符串，实际 %q", got)
	}
}
