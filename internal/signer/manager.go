package signer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-apksigner-gui/internal/dto"
	"go-apksigner-gui/internal/fileutil"
	"go-apksigner-gui/internal/i18n"
	"go-apksigner-gui/internal/logger"
	"go-apksigner-gui/internal/tools"
)

// Manager 签名管理器，依赖工具链提供 apksigner / zipalign。
type Manager struct {
	toolMgr *tools.Manager
}

// NewManager 创建签名管理器。
func NewManager(toolMgr *tools.Manager) *Manager {
	return &Manager{toolMgr: toolMgr}
}

// 各类操作的超时时间：签名可能因大包耗时较久。
const (
	zipAlignTimeout = 5 * time.Minute
	signTimeout     = 15 * time.Minute
	verifyTimeout   = 2 * time.Minute
)

// Callbacks 任务回调：步骤状态与实时日志（日志自动脱敏）。
type Callbacks struct {
	OnStep  func(dto.Step)
	OnLog   func(stream, text string)
	Secrets []string
}

// step 推送步骤状态。
func (c *Callbacks) step(name, status, detail string) {
	logger.Info("流水线步骤", "name", name, "status", status, "detail", logger.Mask(detail, c.Secrets...))
	if c != nil && c.OnStep != nil {
		c.OnStep(dto.Step{Name: name, Status: status, Detail: detail})
	}
}

// log line 推送实时日志。
func (c *Callbacks) line(stream, text string) {
	if c != nil && c.OnLog != nil && text != "" {
		c.OnLog(stream, c.mask(text))
	}
}

// mask 对输出中的密码片段脱敏。
func (c *Callbacks) mask(text string) string {
	secrets := []string{}
	if c != nil {
		secrets = c.Secrets
	}
	return logger.Mask(text, secrets...)
}

// MustTool 获取工具链中的可执行文件。
func (m *Manager) mustTool(name string) (tools.Tool, error) {
	if m == nil || m.toolMgr == nil {
		return tools.Tool{}, fmt.Errorf("%s", i18n.T("sign.err.managerUninit"))
	}
	return m.toolMgr.MustTool(name)
}

// RemoveSignature 剔除 APK 中的 META-INF 目录，得到未签名 APK。
func (m *Manager) RemoveSignature(ctx context.Context, input, output string) (int, error) {
	if !fileutil.Exists(input) {
		return 0, fmt.Errorf("%s", i18n.T("sign.err.apkNotExist", input))
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return 0, err
	}
	removed, err := fileutil.RemoveEntries(input, output, []string{"META-INF/"})
	if err != nil {
		return removed, fmt.Errorf("%s: %w", i18n.T("sign.err.removeFail"), err)
	}
	logger.Info("已移除旧签名", "removed", removed, "output", output)
	return removed, nil
}

// ZipAlign 执行 4 字节对齐，返回工具输出。
func (m *Manager) ZipAlign(ctx context.Context, input, output string, cb *Callbacks) (string, error) {
	tool, err := m.mustTool("zipalign")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return "", err
	}
	res := tools.RunTool(ctx, tool, BuildZipAlignArgs(input, output), zipAlignTimeout, cb.line)
	if res == nil {
		return "", fmt.Errorf("%s", i18n.T("sign.err.zipalignNoResult"))
	}
	if res.StartErr != nil {
		return "", res.StartErr
	}
	if res.TimedOut {
		return res.Output(), fmt.Errorf("%s", i18n.T("sign.err.zipalignTimeout"))
	}
	if res.ExitCode != 0 {
		return res.Output(), fmt.Errorf("%s", i18n.T("sign.err.zipalignFail", res.ExitCode, strings.TrimSpace(res.ErrorText())))
	}
	return res.Output(), nil
}

// Verify 验证 APK 签名：验证失败属于有效结果（verified=false），仅在无法执行时返回 error。
func (m *Manager) Verify(ctx context.Context, apkPath string) (*dto.VerifyResult, error) {
	tool, err := m.mustTool("apksigner")
	if err != nil {
		return nil, err
	}
	if !fileutil.Exists(apkPath) {
		return nil, fmt.Errorf("%s", i18n.T("sign.err.apkNotExist", apkPath))
	}
	res := tools.RunTool(ctx, tool, BuildVerifyArgs(apkPath, true), verifyTimeout, nil)
	if res == nil {
		return nil, fmt.Errorf("%s", i18n.T("sign.err.verifyNoResult"))
	}
	if res.StartErr != nil {
		return nil, res.StartErr
	}
	if res.TimedOut {
		return nil, fmt.Errorf("%s", i18n.T("sign.err.verifyTimeout"))
	}
	result := ParseVerifyOutput(res.Output())
	return &result, nil
}

// IsSigned 判断 APK 是否已签名（供 APK 信息解析使用）。
func (m *Manager) IsSigned(ctx context.Context, apkPath string) bool {
	res, err := m.Verify(ctx, apkPath)
	if err != nil || res == nil {
		return false
	}
	return res.Verified
}

// tmpDir 创建本次任务的临时目录。
func (m *Manager) tmpDir() (string, error) {
	base := filepath.Join(os.TempDir(), "go-apksigner-gui", "tmp")
	if m != nil && m.toolMgr != nil {
		base = filepath.Join(m.toolMgr.ToolsDir(), "tmp")
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(base, "task-")
}
