// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

// Package apkinfo 解析 APK 基础信息：优先使用 aapt2 dump badging，
// 工具缺失时回退到内置的二进制 AndroidManifest(AXML) 解析。
package apkinfo

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go-apksigner-gui/internal/dto"
	"go-apksigner-gui/internal/fileutil"
	"go-apksigner-gui/internal/i18n"
	"go-apksigner-gui/internal/logger"
	"go-apksigner-gui/internal/tools"
)

// SignatureChecker 判断是否已签名的能力（由 signer.Manager 提供）。
type SignatureChecker interface {
	IsSigned(ctx context.Context, apkPath string) bool
}

// Parser APK 信息解析器。
type Parser struct {
	toolMgr  *tools.Manager
	verifier SignatureChecker
}

// NewParser 创建解析器。
func NewParser(toolMgr *tools.Manager, verifier SignatureChecker) *Parser {
	return &Parser{toolMgr: toolMgr, verifier: verifier}
}

var (
	packagePattern = regexp.MustCompile(`package:\s*name='([^']*)'\s*versionCode='([^']*)'\s*versionName='([^']*)'`)
	sdkPattern     = regexp.MustCompile(`sdkVersion:'([^']*)'`)
	targetPattern  = regexp.MustCompile(`targetSdkVersion:'([^']*)'`)
	labelPattern   = regexp.MustCompile(`application-label(?:-[^:]*)?:'([^']*)'`)
)

// Parse 解析 APK 信息，返回结构化结果。
func (p *Parser) Parse(ctx context.Context, apkPath string) (dto.APKInfo, error) {
	info := dto.APKInfo{Path: apkPath, FileSize: fileutil.FileSize(apkPath)}
	if !fileutil.Exists(apkPath) {
		return info, fmt.Errorf("%s", i18n.T("sign.err.apkNotExist", apkPath))
	}
	if p != nil && p.verifier != nil {
		info.IsSigned = p.verifier.IsSigned(ctx, apkPath)
	}

	if parsed, err := p.parseWithAapt2(ctx, apkPath); err == nil {
		parsed.IsSigned = info.IsSigned
		parsed.FileSize = info.FileSize
		return parsed, nil
	} else {
		logger.Warn("aapt2 解析失败，回退内置解析", "error", err.Error())
	}

	parsed, err := ParseManifest(apkPath)
	if err != nil {
		// 兜底失败也要返回已知字段，避免前端无内容可展示
		info.Source = "unknown"
		return info, err
	}
	parsed.IsSigned = info.IsSigned
	parsed.FileSize = info.FileSize
	return parsed, nil
}

// parseWithAapt2 调用 aapt2 dump badging 并解析输出。
func (p *Parser) parseWithAapt2(ctx context.Context, apkPath string) (dto.APKInfo, error) {
	if p == nil || p.toolMgr == nil {
		return dto.APKInfo{}, fmt.Errorf("%s", i18n.T("apkinfo.err.managerUninit"))
	}
	aapt2Tool, err := p.toolMgr.MustTool("aapt2")
	if err != nil {
		return dto.APKInfo{}, err
	}
	res := tools.RunTool(ctx, aapt2Tool, []string{"dump", "badging", apkPath}, 2*time.Minute, nil)
	if res == nil || res.StartErr != nil {
		return dto.APKInfo{}, fmt.Errorf("%s", i18n.T("apkinfo.err.aapt2Fail"))
	}
	if res.ExitCode != 0 {
		return dto.APKInfo{}, fmt.Errorf("%s", i18n.T("apkinfo.err.aapt2Exit", res.ExitCode, strings.TrimSpace(res.ErrorText())))
	}
	info := ParseBadging(res.Output())
	info.Path = apkPath
	info.Source = "aapt2"
	if info.PackageName == "" {
		return info, fmt.Errorf("%s", i18n.T("apkinfo.err.noPackageInBadging"))
	}
	info.RawOutput = ""
	return info, nil
}

// ParseBadging 解析 aapt2 dump badging 的纯文本内容。
func ParseBadging(output string) dto.APKInfo {
	info := dto.APKInfo{}
	text := strings.ReplaceAll(output, "\r\n", "\n")
	if m := packagePattern.FindStringSubmatch(text); len(m) == 4 {
		info.PackageName = m[1]
		info.VersionCode = m[2]
		info.VersionName = m[3]
	}
	if m := sdkPattern.FindStringSubmatch(text); len(m) == 2 {
		info.MinSDK = m[1]
	}
	if m := targetPattern.FindStringSubmatch(text); len(m) == 2 {
		info.TargetSDK = m[1]
	}
	if m := labelPattern.FindStringSubmatch(text); len(m) == 2 {
		info.Label = m[1]
	}
	return info
}
