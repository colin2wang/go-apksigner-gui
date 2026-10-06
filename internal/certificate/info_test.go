package certificate

import "testing"

const englishOutput = `Keystore type: PKCS12
Keystore provider: SUN

Your keystore contains 1 entry

Alias name: android-key
Creation date: Oct 6, 2026
Entry type: PrivateKeyEntry
Certificate chain length: 1
Certificate[1]:
Owner: CN=Test App, OU=Dev, O=TestOrg, L=Beijing, ST=Beijing, C=CN
Issuer: CN=Test App, OU=Dev, O=TestOrg, L=Beijing, ST=Beijing, C=CN
Serial number: 3f2a1b
Valid from: Mon Oct 06 12:00:00 CST 2026 until: Wed Oct 04 12:00:00 CST 2051
Certificate fingerprints:
	 SHA1: AA:BB:CC:DD
	 SHA256: 11:22:33:44:55
Signature algorithm name: SHA256withRSA
Subject Public Key Algorithm: 2048-bit RSA key
Version: 3`

const chineseOutput = `密钥库类型: JKS
密钥库提供方: SUN

您的密钥库包含 1 个条目

别名: release-key
创建日期: 2026-10-6
条目类型: PrivateKeyEntry
证书链长度: 1
证书[1]:
所有者: CN=Demo App, OU=Dev, O=Demo, L=Shanghai, ST=Shanghai, C=CN
签发人: CN=Demo App, OU=Dev, O=Demo, L=Shanghai, ST=Shanghai, C=CN
序列号: abc123
有效期: 从 Mon Oct 06 12:00:00 CST 2026 至 Wed Oct 04 12:00:00 CST 2051
证书指纹:
	 MD5:  66:77:88
	 SHA1: 99:AA:BB
	 SHA256: CC:DD:EE
签名算法名称: SHA256withRSA
主体公共密钥算法: 2048 位 RSA 密钥
版本: 3`

func TestParseKeytoolListEnglish(t *testing.T) {
	info := ParseKeytoolList(englishOutput)
	if info.Type != "PKCS12" {
		t.Fatalf("密钥库类型解析错误: %q", info.Type)
	}
	if len(info.Entries) != 1 {
		t.Fatalf("期望解析出 1 个条目，实际 %d", len(info.Entries))
	}
	e := info.Entries[0]
	if e.Alias != "android-key" {
		t.Fatalf("别名解析错误: %q", e.Alias)
	}
	if e.Owner != "CN=Test App, OU=Dev, O=TestOrg, L=Beijing, ST=Beijing, C=CN" {
		t.Fatalf("所有者解析错误: %q", e.Owner)
	}
	if e.Serial != "3f2a1b" {
		t.Fatalf("序列号解析错误: %q", e.Serial)
	}
	if e.SHA1 != "AA:BB:CC:DD" || e.SHA256 != "11:22:33:44:55" {
		t.Fatalf("指纹解析错误: SHA1=%q SHA256=%q", e.SHA1, e.SHA256)
	}
	if e.ValidFrom != "Mon Oct 06 12:00:00 CST 2026" {
		t.Fatalf("有效期起解析错误: %q", e.ValidFrom)
	}
	if e.ValidTo != "Wed Oct 04 12:00:00 CST 2051" {
		t.Fatalf("有效期止解析错误: %q", e.ValidTo)
	}
	if e.EntryType != "PrivateKeyEntry" {
		t.Fatalf("条目类型解析错误: %q", e.EntryType)
	}
	if e.KeyAlgo != "2048-bit RSA key" {
		t.Fatalf("公钥算法解析错误: %q", e.KeyAlgo)
	}
}

func TestParseKeytoolListChinese(t *testing.T) {
	info := ParseKeytoolList(chineseOutput)
	if info.Type != "JKS" {
		t.Fatalf("密钥库类型解析错误: %q", info.Type)
	}
	if len(info.Entries) != 1 {
		t.Fatalf("中文输出也应解析出 1 个条目，实际 %d", len(info.Entries))
	}
	e := info.Entries[0]
	if e.Alias != "release-key" {
		t.Fatalf("别名解析错误: %q", e.Alias)
	}
	if e.MD5 != "66:77:88" || e.SHA1 != "99:AA:BB" || e.SHA256 != "CC:DD:EE" {
		t.Fatalf("指纹解析错误: MD5=%q SHA1=%q SHA256=%q", e.MD5, e.SHA1, e.SHA256)
	}
	if e.ValidFrom != "Mon Oct 06 12:00:00 CST 2026" || e.ValidTo != "Wed Oct 04 12:00:00 CST 2051" {
		t.Fatalf("有效期解析错误: %q ~ %q", e.ValidFrom, e.ValidTo)
	}
	if e.Owner == "" || e.Issuer == "" {
		t.Fatal("所有者/签发人不应为空")
	}
}

func TestParseKeytoolListMultipleEntries(t *testing.T) {
	combined := englishOutput + "\n\nAlias name: second-key\nOwner: CN=Second\nCertificate fingerprints:\n\t SHA256: FF:FF\n"
	info := ParseKeytoolList(combined)
	if len(info.Entries) != 2 {
		t.Fatalf("期望 2 个条目，实际 %d", len(info.Entries))
	}
	if info.Entries[1].Alias != "second-key" || info.Entries[1].SHA256 != "FF:FF" {
		t.Fatalf("第二个条目解析错误: %+v", info.Entries[1])
	}
}

func TestCountEntries(t *testing.T) {
	if got := CountEntries(englishOutput); got != 1 {
		t.Fatalf("英文条目数应为 1，实际 %d", got)
	}
	if got := CountEntries(chineseOutput); got != 1 {
		t.Fatalf("中文条目数应为 1，实际 %d", got)
	}
	if got := CountEntries("nothing here"); got != -1 {
		t.Fatalf("无匹配时应返回 -1，实际 %d", got)
	}
}
