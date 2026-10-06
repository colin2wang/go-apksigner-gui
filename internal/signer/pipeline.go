package signer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-apksigner-gui/internal/dto"
	"go-apksigner-gui/internal/fileutil"
	"go-apksigner-gui/internal/tools"
)

// 流水线步骤名称（中文），前端按此顺序展示步骤条。
const (
	stepRemoveSign = "移除旧签名"
	stepZipAlign   = "zipalign 对齐"
	stepSign       = "APK 签名"
	stepVerify     = "签名验证"
)

// Sign 执行完整签名流水线：去签 → 对齐 → 签名 → 验证。
// 任一步骤失败都会通过 dto.Step 上报，并返回带失败原因的错误。
func (m *Manager) Sign(ctx context.Context, opts dto.SignOptions, cb *Callbacks) (*dto.SignResult, error) {
	if err := validateSignOptions(opts); err != nil {
		return nil, err
	}
	result := &dto.SignResult{}

	workDir, err := m.tmpDir()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.RemoveAll(workDir); err != nil {
			cb.line("stderr", fmt.Sprintf("清理临时目录失败: %v", err))
		}
	}()

	initialSteps := []string{}
	if opts.RemoveOldSign {
		initialSteps = append(initialSteps, stepRemoveSign)
	}
	if opts.ZipAlignFirst {
		initialSteps = append(initialSteps, stepZipAlign)
	}
	allSteps := append(append(initialSteps, stepSign), stepVerify)
	for _, name := range allSteps {
		cb.step(name, "pending", "")
	}

	inputAPK := opts.InputAPK

	// 步骤 1：移除旧签名（避免重复签名导致安装失败）
	if opts.RemoveOldSign {
		cb.step(stepRemoveSign, "running", inputAPK)
		unsigned := filepath.Join(workDir, "unsigned.apk")
		removed, err := m.RemoveSignature(ctx, inputAPK, unsigned)
		if err != nil {
			cb.step(stepRemoveSign, "error", err.Error())
			return nil, fmt.Errorf("移除旧签名失败: %w", err)
		}
		inputAPK = unsigned
		cb.step(stepRemoveSign, "success", fmt.Sprintf("已剔除 %d 个签名文件", removed))
	}

	// 步骤 2：zipalign 对齐（必须在签名之前执行）
	if opts.ZipAlignFirst {
		cb.step(stepZipAlign, "running", inputAPK)
		aligned := filepath.Join(workDir, "aligned.apk")
		out, err := m.ZipAlign(ctx, inputAPK, aligned, cb)
		if err != nil {
			cb.step(stepZipAlign, "error", err.Error())
			return nil, err
		}
		inputAPK = aligned
		cb.step(stepZipAlign, "success", trimOutput(out))
	}

	// 步骤 3：apksigner sign
	outputAPK := opts.OutputAPK
	if strings.TrimSpace(outputAPK) == "" {
		outputAPK = DefaultOutputPath(opts.InputAPK)
	}
	if err := os.MkdirAll(filepath.Dir(outputAPK), 0o755); err != nil {
		cb.step(stepSign, "error", err.Error())
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}
	cb.step(stepSign, "running", outputAPK)

	apksigner, err := m.mustTool("apksigner")
	if err != nil {
		cb.step(stepSign, "error", err.Error())
		return nil, err
	}
	signOpts := opts
	signOpts.OutputAPK = outputAPK
	res := tools.RunTool(ctx, apksigner, BuildSignArgs(signOpts, inputAPK), signTimeout, cb.line)
	if res == nil {
		cb.step(stepSign, "error", "apksigner 未返回结果")
		return nil, fmt.Errorf("apksigner 未返回结果")
	}
	if res.StartErr != nil {
		cb.step(stepSign, "error", res.StartErr.Error())
		return nil, res.StartErr
	}
	if res.TimedOut {
		cb.step(stepSign, "error", "签名超时")
		return nil, fmt.Errorf("apksigner sign 执行超时（超过 %v）", signTimeout)
	}
	if res.ExitCode != 0 {
		detail := strings.TrimSpace(res.ErrorText())
		cb.step(stepSign, "error", detail)
		return nil, fmt.Errorf("签名失败(退出码 %d): %s", res.ExitCode, cb.mask(detail))
	}
	cb.step(stepSign, "success", "签名完成")

	// 步骤 4：验证签名结果
	cb.step(stepVerify, "running", outputAPK)
	verifyResult, err := m.Verify(ctx, outputAPK)
	if err != nil {
		cb.step(stepVerify, "error", err.Error())
		return nil, err
	}
	result.Verify = verifyResult
	result.OutputPath = outputAPK
	result.FileSize = fileutil.FileSize(outputAPK)
	result.SignVersion = VerifySummary(*verifyResult)

	if !verifyResult.Verified {
		cb.step(stepVerify, "error", "apksigner 验证未通过")
		result.Success = false
		result.Message = "签名完成但校验未通过，请查看日志了解详情"
		return result, nil
	}
	cb.step(stepVerify, "success", "已验证: "+result.SignVersion)
	result.Success = true
	result.Message = fmt.Sprintf("签名成功，包含方案 %s", result.SignVersion)
	return result, nil
}

// validateSignOptions 校验签名入参。
func validateSignOptions(opts dto.SignOptions) error {
	if strings.TrimSpace(opts.InputAPK) == "" {
		return fmt.Errorf("请选择待签名的 APK 文件")
	}
	if !fileutil.Exists(opts.InputAPK) {
		return fmt.Errorf("APK 文件不存在: %s", opts.InputAPK)
	}
	if strings.TrimSpace(opts.Keystore) == "" {
		return fmt.Errorf("请选择密钥库文件")
	}
	if !fileutil.Exists(opts.Keystore) {
		return fmt.Errorf("密钥库不存在: %s", opts.Keystore)
	}
	if opts.StorePass == "" {
		return fmt.Errorf("请填写密钥库密码")
	}
	if !opts.V1Enabled && !opts.V2Enabled && !opts.V3Enabled && !opts.V4Enabled {
		return fmt.Errorf("请至少启用一种签名方案（V1/V2/V3/V4）")
	}
	if opts.MinSDK > 0 && opts.MaxSDK > 0 && opts.MinSDK > opts.MaxSDK {
		return fmt.Errorf("最低 SDK 不能大于最高 SDK")
	}
	return nil
}

// trimOutput 压缩长输出用于步骤摘要展示。
func trimOutput(s string) string {
	s = strings.TrimSpace(s)
	lines := strings.Split(s, "\n")
	if len(lines) > 5 {
		return strings.Join(lines[:5], "\n") + fmt.Sprintf("\n...（共 %d 行）", len(lines))
	}
	return s
}
