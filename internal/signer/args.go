// Package signer 封装 zipalign 对齐、apksigner 签名与验证，以及完整签名流水线。
package signer

import (
	"fmt"
	"strings"

	"go-apksigner-gui/internal/dto"
)

// BuildZipAlignArgs 构造 zipalign 参数：强制覆盖 + 4 字节对齐。
func BuildZipAlignArgs(input, output string) []string {
	return []string{"-f", "-v", "4", input, output}
}

// BuildSignArgs 构造 apksigner sign 参数（入参不含待签名 APK，便于流水线替换中间产物）。
func BuildSignArgs(opts dto.SignOptions, input string) []string {
	args := []string{"sign"}

	if strings.TrimSpace(opts.Keystore) != "" {
		args = append(args, "--ks", opts.Keystore)
	}
	if strings.TrimSpace(opts.Alias) != "" {
		args = append(args, "--ks-key-alias", strings.TrimSpace(opts.Alias))
	}
	if opts.StorePass != "" {
		args = append(args, "--ks-pass", "pass:"+opts.StorePass)
	}
	if opts.KeyPass != "" {
		args = append(args, "--key-pass", "pass:"+opts.KeyPass)
	}
	if strings.TrimSpace(opts.OutputAPK) != "" {
		args = append(args, "--out", opts.OutputAPK)
	}
	if opts.MinSDK > 0 {
		args = append(args, "--min-sdk-version", fmt.Sprintf("%d", opts.MinSDK))
	}
	if opts.MaxSDK > 0 {
		args = append(args, "--max-sdk-version", fmt.Sprintf("%d", opts.MaxSDK))
	}
	args = append(args,
		"--v1-signing-enabled", boolArg(opts.V1Enabled),
		"--v2-signing-enabled", boolArg(opts.V2Enabled),
		"--v3-signing-enabled", boolArg(opts.V3Enabled),
		"--v4-signing-enabled", boolArg(opts.V4Enabled),
	)
	args = append(args, input)
	return args
}

// BuildVerifyArgs 构造 apksigner verify 参数。
func BuildVerifyArgs(apkPath string, printCerts bool) []string {
	args := []string{"verify", "--verbose"}
	if printCerts {
		args = append(args, "--print-certs")
	}
	return append(args, apkPath)
}

// DefaultOutputPath 未指定输出路径时的默认命名：xxx-signed.apk。
func DefaultOutputPath(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	ext := ".apk"
	if idx := strings.LastIndex(input, "."); idx > strings.LastIndexAny(input, `/\`) {
		ext = input[idx:]
		return input[:idx] + "-signed" + ext
	}
	return input + "-signed" + ext
}

// boolArg 将布尔值转为 apksigner 需要的字符串。
func boolArg(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
