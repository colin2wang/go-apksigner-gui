// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

package certificate

import (
	"fmt"
	"strings"
	"testing"

	"go-apksigner-gui/internal/dto"
)

func sampleRequest() dto.KeystoreRequest {
	return dto.KeystoreRequest{
		OutputPath: "C:\\keys\\release.jks",
		StorePass:  "123456",
		KeyPass:    "123456",
		Cert: dto.CertInfo{
			Alias:        "android-key",
			CommonName:   "Demo App",
			Organization: "Demo Inc, Ltd",
			OrgUnit:      "Dev",
			Locality:     "Beijing",
			State:        "Beijing",
			Country:      "CN",
			KeyAlgorithm: "RSA",
			KeySize:      2048,
			Validity:     3650,
		},
	}
}

func TestValidateRequest(t *testing.T) {
	req := sampleRequest()
	if err := ValidateRequest(req); err != nil {
		t.Fatalf("合法请求不应报错: %v", err)
	}
	badPath := req
	badPath.OutputPath = ""
	if err := ValidateRequest(badPath); err == nil {
		t.Fatal("未填写保存路径应报错")
	}
	shortPass := req
	shortPass.StorePass = "123"
	if err := ValidateRequest(shortPass); err == nil {
		t.Fatal("弱密码应报错")
	}
	badExt := req
	badExt.OutputPath = "release.txt"
	if err := ValidateRequest(badExt); err == nil {
		t.Fatal("非法后缀应报错")
	}
	noCN := req
	noCN.Cert.CommonName = ""
	if err := ValidateRequest(noCN); err == nil {
		t.Fatal("缺少 CN 应报错")
	}
}

func TestBuildGenArgsRSA(t *testing.T) {
	args, err := buildGenArgs(sampleRequest())
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"-genkeypair", "-keyalg RSA", "-keysize 2048", "-validity 3650", "-storetype JKS"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("参数缺少 %q，实际: %s", want, joined)
		}
	}
	if !strings.Contains(joined, "CN=Demo App") {
		t.Fatalf("DN 未包含 CN: %s", joined)
	}
	// 组织名含逗号时必须转义，避免 DN 解析错位
	if !strings.Contains(joined, `O=Demo Inc\, Ltd`) {
		t.Fatalf("DN 未转义逗号: %s", joined)
	}
	if idx := indexOf(args, "-genkeypair"); idx < 0 || args[idx+1] != "-v" {
		t.Fatal("应启用 -v 输出便于日志排查")
	}
}

func TestBuildGenArgsPKCS12AndEC(t *testing.T) {
	req := sampleRequest()
	req.OutputPath = "/tmp/release.p12"
	req.StoreType = ""
	args, _ := buildGenArgs(req)
	if !strings.Contains(strings.Join(args, " "), "-storetype PKCS12") {
		t.Fatal(".p12 应推断为 PKCS12 类型")
	}

	ecReq := sampleRequest()
	ecReq.Cert.KeyAlgorithm = "EC"
	ecArgs, err := buildGenArgs(ecReq)
	if err != nil {
		t.Fatal(err)
	}
	ecJoined := strings.Join(ecArgs, " ")
	if !strings.Contains(ecJoined, "-groupname secp256r1") {
		t.Fatalf("EC 算法应使用 -groupname: %s", ecJoined)
	}
	if strings.Contains(ecJoined, "-keysize") {
		t.Fatalf("EC 算法不应使用 -keysize: %s", ecJoined)
	}
}

func TestBuildGenArgsDefaults(t *testing.T) {
	req := dto.KeystoreRequest{
		OutputPath: "/tmp/demo.keystore",
		StorePass:  "changeit",
		Cert:       dto.CertInfo{Alias: "key0", CommonName: "App"},
	}
	args, err := buildGenArgs(req)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-validity "+fmt.Sprintf("%d", DefaultValidity)) {
		t.Fatalf("未指定有效期时应使用默认值 %d: %s", DefaultValidity, joined)
	}
	if !strings.Contains(joined, "-keysize 2048") {
		t.Fatalf("未指定密钥长度时应默认 2048: %s", joined)
	}
	if !strings.Contains(joined, "C=CN") {
		t.Fatalf("未指定国家时应默认 CN: %s", joined)
	}
}

func TestEscapeDN(t *testing.T) {
	if got := escapeDN(`A, Inc\`); got != `A\, Inc\\` {
		t.Fatalf("转义结果错误: %s", got)
	}
	if got := escapeDN("   "); got != "Unknown" {
		t.Fatalf("空值应回退 Unknown，实际 %q", got)
	}
}

func TestInferredStoreType(t *testing.T) {
	if got := inferredStoreType("a.p12"); got != "PKCS12" {
		t.Fatalf("p12 应推断为 PKCS12，实际 %s", got)
	}
	if got := inferredStoreType("a.jks"); got != "JKS" {
		t.Fatalf("jks 应推断为 JKS，实际 %s", got)
	}
}

// indexOf 返回字符串在切片中的位置。
func indexOf(list []string, target string) int {
	for i, s := range list {
		if s == target {
			return i
		}
	}
	return -1
}
