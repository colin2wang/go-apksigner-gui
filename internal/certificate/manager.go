// Package certificate 封装 keytool 能力：密钥库生成、证书信息查看、导出与格式转换。
package certificate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-apksigner-gui/internal/dto"
	"go-apksigner-gui/internal/i18n"
	"go-apksigner-gui/internal/logger"
	"go-apksigner-gui/internal/tools"
)

// Manager 证书管理器，依赖工具链提供的 keytool。
type Manager struct {
	toolMgr *tools.Manager
}

// NewManager 创建证书管理器。
func NewManager(toolMgr *tools.Manager) *Manager {
	return &Manager{toolMgr: toolMgr}
}

// DefaultValidity 默认有效期（天），约 25 年。
const DefaultValidity = 25 * 365

// keytool 获取当前可用的 keytool 工具。
func (m *Manager) keytool() (tools.Tool, error) {
	if m == nil || m.toolMgr == nil {
		return tools.Tool{}, fmt.Errorf("%s", i18n.T("cert.err.managerUninit"))
	}
	return m.toolMgr.MustTool("keytool")
}

// javaLocaleArgs 强制 keytool 输出英文（便于稳定解析）；若 keytool 不支持 -J 会忽略该参数。
var javaLocaleArgs = []string{"-J-Duser.language=en", "-J-Duser.country=US"}

// runKeytool 执行 keytool 命令并对输出脱敏后返回原始结果。
func (m *Manager) runKeytool(ctx context.Context, timeout time.Duration, args []string, secrets ...string) (*toolsRunResult, error) {
	kt, err := m.keytool()
	if err != nil {
		return nil, err
	}
	res := tools.RunTool(ctx, kt, append(append([]string{}, javaLocaleArgs...), args...), timeout, nil)
	if res == nil {
		return nil, fmt.Errorf("%s", i18n.T("cert.err.keytoolNoResult"))
	}
	if res.StartErr != nil {
		return nil, res.StartErr
	}
	out := res.Output()
	logger.Debug("keytool 输出", "args", len(args), "exit", res.ExitCode)
	if res.ExitCode != 0 {
		return &toolsRunResult{Output: out, ExitCode: res.ExitCode}, fmt.Errorf("%s", i18n.T("cert.err.keytoolExit", res.ExitCode, logger.Mask(strings.TrimSpace(res.ErrorText()), secrets...)))
	}
	return &toolsRunResult{Output: out, ExitCode: res.ExitCode}, nil
}

type toolsRunResult struct {
	Output   string
	ExitCode int
}

// ValidateRequest 校验生成密钥库的入参。
func ValidateRequest(req dto.KeystoreRequest) error {
	if strings.TrimSpace(req.OutputPath) == "" {
		return fmt.Errorf("%s", i18n.T("cert.err.noOutputPath"))
	}
	if strings.TrimSpace(req.Cert.Alias) == "" {
		return fmt.Errorf("%s", i18n.T("cert.err.noAlias"))
	}
	if len(req.StorePass) < 6 {
		return fmt.Errorf("%s", i18n.T("cert.err.storePassShort"))
	}
	if req.KeyPass == "" {
		req.KeyPass = req.StorePass
	}
	if len(req.KeyPass) < 6 {
		return fmt.Errorf("%s", i18n.T("cert.err.keyPassShort"))
	}
	if strings.TrimSpace(req.Cert.CommonName) == "" {
		return fmt.Errorf("%s", i18n.T("cert.err.noCN"))
	}
	ext := strings.ToLower(filepath.Ext(req.OutputPath))
	switch ext {
	case ".jks", ".keystore", ".p12", ".pfx":
	default:
		return fmt.Errorf("%s", i18n.T("cert.err.badExt", ext))
	}
	return nil
}

// Generate 生成密钥库，返回 keytool 原始输出。
func (m *Manager) Generate(ctx context.Context, req dto.KeystoreRequest) (string, error) {
	if err := ValidateRequest(req); err != nil {
		return "", err
	}
	args, err := buildGenArgs(req)
	if err != nil {
		return "", err
	}
	if dir := filepath.Dir(req.OutputPath); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("%s: %w", i18n.T("cert.err.createDir"), err)
		}
	}
	res, err := m.runKeytool(ctx, 2*time.Minute, args, req.StorePass, req.KeyPass)
	if err != nil {
		if res != nil {
			return res.Output, err
		}
		return "", err
	}
	return res.Output, nil
}

// buildGenArgs 构造 -genkeypair 参数，包含 DN 转义与算法处理。
func buildGenArgs(req dto.KeystoreRequest) ([]string, error) {
	c := req.Cert
	algo := strings.ToUpper(defaultString(c.KeyAlgorithm, "RSA"))
	keySize := c.KeySize
	if keySize == 0 {
		keySize = 2048
	}
	validity := c.Validity
	if validity <= 0 {
		validity = DefaultValidity
	}
	storeType := strings.ToUpper(defaultString(req.StoreType, ""))
	if storeType == "" {
		storeType = inferredStoreType(req.OutputPath)
	}
	keyPass := req.KeyPass
	if keyPass == "" {
		keyPass = req.StorePass
	}

	args := []string{
		"-genkeypair", "-v",
		"-keystore", req.OutputPath,
		"-alias", strings.TrimSpace(c.Alias),
		"-keyalg", algo,
		"-validity", fmt.Sprintf("%d", validity),
		"-storepass", req.StorePass,
		"-keypass", keyPass,
		"-dname", buildDName(c),
	}
	if storeType != "" {
		args = append(args, "-storetype", storeType)
	}
	switch algo {
	case "EC":
		args = append(args, "-groupname", "secp256r1")
	case "DSA":
		args = append(args, "-keysize", fmt.Sprintf("%d", keySize))
	default:
		if algo != "RSA" {
			return nil, fmt.Errorf("%s", i18n.T("cert.err.unsupportedAlgo", algo))
		}
		args = append(args, "-keysize", fmt.Sprintf("%d", keySize))
	}
	return args, nil
}

// buildDName 构造 DN 字符串，转义逗号与反斜杠避免解析错误。
func buildDName(c dto.CertInfo) string {
	parts := []string{
		"CN=" + escapeDN(c.CommonName),
		"OU=" + escapeDN(defaultString(c.OrgUnit, "Dev")),
		"O=" + escapeDN(defaultString(c.Organization, c.CommonName)),
		"L=" + escapeDN(defaultString(c.Locality, "Unknown")),
		"ST=" + escapeDN(defaultString(c.State, c.Locality)),
		"C=" + escapeDN(defaultString(c.Country, "CN")),
	}
	return strings.Join(parts, ", ")
}

// escapeDN 转义 DN 中的特殊字符。
func escapeDN(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "Unknown"
	}
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "+", `\+`)
	return s
}

// inferredStoreType 依据扩展名推断密钥库类型。
func inferredStoreType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".p12", ".pfx":
		return "PKCS12"
	default:
		return "JKS"
	}
}

// List 查看密钥库详情。
func (m *Manager) List(ctx context.Context, path, storePass string) (*dto.KeystoreInfo, error) {
	if path == "" {
		return nil, fmt.Errorf("%s", i18n.T("cert.err.noKeystore"))
	}
	if storePass == "" {
		return nil, fmt.Errorf("%s", i18n.T("cert.err.noStorePass"))
	}
	args := []string{"-list", "-v", "-keystore", path, "-storepass", storePass}
	res, err := m.runKeytool(ctx, time.Minute, args, storePass)
	if err != nil {
		return nil, err
	}
	info := ParseKeytoolList(res.Output)
	info.Path = path
	if info.Type == "" {
		info.Type = inferredStoreType(path)
	}
	if len(info.Entries) == 0 {
		return &info, fmt.Errorf("%s", i18n.T("cert.err.noEntry"))
	}
	info.Raw = ""
	return &info, nil
}

// Aliases 返回密钥库中的全部别名。
func (m *Manager) Aliases(ctx context.Context, path, storePass string) ([]string, error) {
	info, err := m.List(ctx, path, storePass)
	if err != nil {
		return nil, err
	}
	aliases := make([]string, 0, len(info.Entries))
	for _, e := range info.Entries {
		aliases = append(aliases, e.Alias)
	}
	return aliases, nil
}

// Export 导出证书（PEM 格式 .cer/.pem）。
func (m *Manager) Export(ctx context.Context, keystorePath, storePass, alias, outputPath string) (string, error) {
	if keystorePath == "" || outputPath == "" {
		return "", fmt.Errorf("%s", i18n.T("cert.err.exportPaths"))
	}
	if dir := filepath.Dir(outputPath); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	args := []string{
		"-exportcert", "-rfc",
		"-keystore", keystorePath,
		"-storepass", storePass,
		"-alias", alias,
		"-file", outputPath,
	}
	res, err := m.runKeytool(ctx, time.Minute, args, storePass)
	out := ""
	if res != nil {
		out = res.Output
	}
	if err != nil {
		return out, err
	}
	return out, nil
}

// Convert 密钥库格式转换（JKS <-> PKCS12）。
func (m *Manager) Convert(ctx context.Context, srcPath, srcPass, dstPath, dstPass, dstType string) (string, error) {
	if srcPath == "" || dstPath == "" {
		return "", fmt.Errorf("%s", i18n.T("cert.err.convertPaths"))
	}
	if srcPath == dstPath {
		return "", fmt.Errorf("%s", i18n.T("cert.err.samePath"))
	}
	dstType = strings.ToUpper(defaultString(dstType, inferredStoreType(dstPath)))
	args := []string{
		"-importkeystore",
		"-srckeystore", srcPath,
		"-srcstorepass", srcPass,
		"-destkeystore", dstPath,
		"-deststorepass", dstPass,
		"-deststoretype", dstType,
		"-noprompt",
	}
	res, err := m.runKeytool(ctx, 2*time.Minute, args, srcPass, dstPass)
	out := ""
	if res != nil {
		out = res.Output
	}
	if err != nil {
		return out, err
	}
	return out, nil
}

// DeleteAlias 删除密钥库中的别名。
func (m *Manager) DeleteAlias(ctx context.Context, path, storePass, alias string) (string, error) {
	if alias == "" {
		return "", fmt.Errorf("%s", i18n.T("cert.err.noAliasDel"))
	}
	args := []string{"-delete", "-keystore", path, "-storepass", storePass, "-alias", alias}
	res, err := m.runKeytool(ctx, time.Minute, args, storePass)
	out := ""
	if res != nil {
		out = res.Output
	}
	return out, err
}

func defaultString(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return strings.TrimSpace(s)
}
