package signer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-apksigner-gui/internal/dto"
	"go-apksigner-gui/internal/fileutil"
	"go-apksigner-gui/internal/i18n"
	"go-apksigner-gui/internal/tools"
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
			cb.line("stderr", i18n.T("sign.err.cleanTempFail", err))
		}
	}()

	initialSteps := []string{}
	if opts.RemoveOldSign {
		initialSteps = append(initialSteps, i18n.T("sign.step.remove"))
	}
	if opts.ZipAlignFirst {
		initialSteps = append(initialSteps, i18n.T("sign.step.zipalign"))
	}
	allSteps := append(append(initialSteps, i18n.T("sign.step.sign")), i18n.T("sign.step.verify"))
	for _, name := range allSteps {
		cb.step(name, "pending", "")
	}

	inputAPK := opts.InputAPK

	// 步骤 1：移除旧签名（避免重复签名导致安装失败）
	if opts.RemoveOldSign {
		cb.step(i18n.T("sign.step.remove"), "running", inputAPK)
		unsigned := filepath.Join(workDir, "unsigned.apk")
		removed, err := m.RemoveSignature(ctx, inputAPK, unsigned)
		if err != nil {
			cb.step(i18n.T("sign.step.remove"), "error", err.Error())
			return nil, fmt.Errorf("%s: %w", i18n.T("sign.err.removeStepFail"), err)
		}
		inputAPK = unsigned
		cb.step(i18n.T("sign.step.remove"), "success", i18n.T("sign.detail.removed", removed))
	}

	// 步骤 2：zipalign 对齐（必须在签名之前执行）
	if opts.ZipAlignFirst {
		cb.step(i18n.T("sign.step.zipalign"), "running", inputAPK)
		aligned := filepath.Join(workDir, "aligned.apk")
		out, err := m.ZipAlign(ctx, inputAPK, aligned, cb)
		if err != nil {
			cb.step(i18n.T("sign.step.zipalign"), "error", err.Error())
			return nil, err
		}
		inputAPK = aligned
		cb.step(i18n.T("sign.step.zipalign"), "success", trimOutput(out))
	}

	// 步骤 3：apksigner sign
	outputAPK := opts.OutputAPK
	if strings.TrimSpace(outputAPK) == "" {
		outputAPK = DefaultOutputPath(opts.InputAPK)
	}
	if err := os.MkdirAll(filepath.Dir(outputAPK), 0o755); err != nil {
		cb.step(i18n.T("sign.step.sign"), "error", err.Error())
		return nil, fmt.Errorf("%s: %w", i18n.T("sign.err.createOutDir"), err)
	}
	cb.step(i18n.T("sign.step.sign"), "running", outputAPK)

	apksigner, err := m.mustTool("apksigner")
	if err != nil {
		cb.step(i18n.T("sign.step.sign"), "error", err.Error())
		return nil, err
	}
	signOpts := opts
	signOpts.OutputAPK = outputAPK
	res := tools.RunTool(ctx, apksigner, BuildSignArgs(signOpts, inputAPK), signTimeout, cb.line)
	if res == nil {
		cb.step(i18n.T("sign.step.sign"), "error", i18n.T("sign.err.apksignerNoResult"))
		return nil, fmt.Errorf("%s", i18n.T("sign.err.apksignerNoResult"))
	}
	if res.StartErr != nil {
		cb.step(i18n.T("sign.step.sign"), "error", res.StartErr.Error())
		return nil, res.StartErr
	}
	if res.TimedOut {
		cb.step(i18n.T("sign.step.sign"), "error", i18n.T("sign.err.signTimeout"))
		return nil, fmt.Errorf("%s", i18n.T("sign.err.signTimeoutDetail", signTimeout))
	}
	if res.ExitCode != 0 {
		detail := strings.TrimSpace(res.ErrorText())
		cb.step(i18n.T("sign.step.sign"), "error", detail)
		return nil, fmt.Errorf("%s", i18n.T("sign.err.signFail", res.ExitCode, cb.mask(detail)))
	}
	cb.step(i18n.T("sign.step.sign"), "success", i18n.T("sign.msg.signDone"))

	// 步骤 4：验证签名结果
	cb.step(i18n.T("sign.step.verify"), "running", outputAPK)
	verifyResult, err := m.Verify(ctx, outputAPK)
	if err != nil {
		cb.step(i18n.T("sign.step.verify"), "error", err.Error())
		return nil, err
	}
	result.Verify = verifyResult
	result.OutputPath = outputAPK
	result.FileSize = fileutil.FileSize(outputAPK)
	result.SignVersion = VerifySummary(*verifyResult)

	if !verifyResult.Verified {
		cb.step(i18n.T("sign.step.verify"), "error", i18n.T("sign.err.verifyNotPass"))
		result.Success = false
		result.Message = i18n.T("sign.msg.verifyFailDetail")
		return result, nil
	}
	cb.step(i18n.T("sign.step.verify"), "success", i18n.T("sign.detail.verified", result.SignVersion))
	result.Success = true
	result.Message = i18n.T("sign.msg.success", result.SignVersion)
	return result, nil
}

// validateSignOptions 校验签名入参。
func validateSignOptions(opts dto.SignOptions) error {
	if strings.TrimSpace(opts.InputAPK) == "" {
		return fmt.Errorf("%s", i18n.T("sign.err.noApk"))
	}
	if !fileutil.Exists(opts.InputAPK) {
		return fmt.Errorf("%s", i18n.T("sign.err.apkNotExist", opts.InputAPK))
	}
	if strings.TrimSpace(opts.Keystore) == "" {
		return fmt.Errorf("%s", i18n.T("sign.err.noKeystore"))
	}
	if !fileutil.Exists(opts.Keystore) {
		return fmt.Errorf("%s", i18n.T("sign.err.keystoreNotExist", opts.Keystore))
	}
	if opts.StorePass == "" {
		return fmt.Errorf("%s", i18n.T("sign.err.noStorePass"))
	}
	if !opts.V1Enabled && !opts.V2Enabled && !opts.V3Enabled && !opts.V4Enabled {
		return fmt.Errorf("%s", i18n.T("sign.err.noScheme"))
	}
	if opts.MinSDK > 0 && opts.MaxSDK > 0 && opts.MinSDK > opts.MaxSDK {
		return fmt.Errorf("%s", i18n.T("sign.err.sdkOrder"))
	}
	return nil
}

// trimOutput 压缩长输出用于步骤摘要展示。
func trimOutput(s string) string {
	s = strings.TrimSpace(s)
	lines := strings.Split(s, "\n")
	if len(lines) > 5 {
		return strings.Join(lines[:5], "\n") + fmt.Sprintf("\n...%s", i18n.T("sign.detail.truncated", len(lines)))
	}
	return s
}
