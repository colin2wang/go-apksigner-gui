// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

package signer

import (
	"regexp"
	"strings"

	"go-apksigner-gui/internal/dto"
)

var (
	schemePattern = regexp.MustCompile(`(?i)Verified using v([1-4]) scheme[^:]*:?\s*(true|false|NOT signed)?`)
	dnPattern     = regexp.MustCompile(`(?i)Signer #\d+ certificate DN:\s*(.+)`)
	sha256Pattern = regexp.MustCompile(`(?i)Signer #\d+ certificate SHA-256 digest:\s*(.+)`)
	sha1Pattern   = regexp.MustCompile(`(?i)Signer #\d+ certificate SHA-1 digest:\s*(.+)`)
	md5Pattern    = regexp.MustCompile(`(?i)Signer #\d+ certificate MD5 digest:\s*(.+)`)
)

// ParseVerifyOutput 解析 apksigner verify --verbose --print-certs 的输出。
func ParseVerifyOutput(output string) dto.VerifyResult {
	result := dto.VerifyResult{RawOutput: output}
	text := strings.ReplaceAll(output, "\r\n", "\n")

	result.Verified = strings.Contains(text, "Verifies") && !strings.Contains(strings.ToUpper(text), "DOES NOT VERIFY")

	for _, m := range schemePattern.FindAllStringSubmatch(text, -1) {
		enabled := true
		switch strings.ToLower(strings.TrimSpace(m[2])) {
		case "true":
			enabled = true
		case "false", "not signed":
			enabled = false
		}
		switch m[1] {
		case "1":
			result.V1Signed = enabled
		case "2":
			result.V2Signed = enabled
		case "3":
			result.V3Signed = enabled
		case "4":
			result.V4Signed = enabled
		}
	}

	certs := map[int]dto.SignerCert{}
	for _, m := range dnPattern.FindAllStringSubmatch(text, -1) {
		i := len(certs)
		c := certs[i]
		c.DN = strings.TrimSpace(m[1])
		certs[i] = c
	}
	collect := func(pattern *regexp.Regexp, set func(int, string)) {
		for idx, m := range pattern.FindAllStringSubmatch(text, -1) {
			set(idx, strings.TrimSpace(m[1]))
		}
	}
	collect(sha256Pattern, func(i int, v string) {
		c := certs[i]
		c.SHA256 = v
		certs[i] = c
	})
	collect(sha1Pattern, func(i int, v string) {
		c := certs[i]
		c.SHA1 = v
		certs[i] = c
	})
	collect(md5Pattern, func(i int, v string) {
		c := certs[i]
		c.MD5 = v
		certs[i] = c
	})

	if len(certs) > 0 {
		result.Certs = make([]dto.SignerCert, 0, len(certs))
		for i := 0; i < len(certs); i++ {
			if c, ok := certs[i]; ok {
				result.Certs = append(result.Certs, c)
			}
		}
	}

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trimmed), "WARNING") {
			result.Warnings = append(result.Warnings, trimmed)
		}
	}
	return result
}

// VerifySummary 生成便于展示的签名方案摘要。
func VerifySummary(v dto.VerifyResult) string {
	parts := []string{}
	if v.V1Signed {
		parts = append(parts, "v1")
	}
	if v.V2Signed {
		parts = append(parts, "v2")
	}
	if v.V3Signed {
		parts = append(parts, "v3")
	}
	if v.V4Signed {
		parts = append(parts, "v4")
	}
	if len(parts) == 0 {
		return "无有效签名"
	}
	return strings.Join(parts, "+")
}
