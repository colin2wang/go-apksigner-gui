// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

// Package tools 负责外部工具链（apksigner / zipalign / aapt2 / keytool）的
// 探测、版本索引、下载安装与代理镜像配置。
package tools

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"go-apksigner-gui/internal/dto"
	"go-apksigner-gui/internal/i18n"
)

// Mirrors 内置镜像源清单（含国内可用站点）。
var Mirrors = []dto.MirrorSource{
	{Name: "Google 官方", BaseURL: "https://dl.google.com/android/repository/", Note: "需要可访问 Google 服务"},
	{Name: "腾讯云镜像", BaseURL: "https://mirrors.cloud.tencent.com/AndroidSDK/", Note: "国内推荐"},
	{Name: "清华大学镜像", BaseURL: "https://mirrors.tuna.tsinghua.edu.cn/android/repository/", Note: "国内推荐"},
	{Name: "大连东软镜像", BaseURL: "https://mirrors.neusoft.edu.cn/android/repository/", Note: "教育网推荐"},
	{Name: "中科大镜像", BaseURL: "https://mirrors.ustc.edu.cn/android/repository/", Note: "国内备选"},
}

// DefaultMirror 返回默认镜像源。
func DefaultMirror() dto.MirrorSource { return Mirrors[0] }

// FindMirror 按 BaseURL 查找镜像源，未命中时返回默认源。
func FindMirror(baseURL string) dto.MirrorSource {
	for _, m := range Mirrors {
		if m.BaseURL == baseURL {
			return m
		}
	}
	if baseURL != "" {
		return dto.MirrorSource{Name: "自定义", BaseURL: baseURL, Note: "自定义地址"}
	}
	return DefaultMirror()
}

// NewHTTPClient 依据代理设置构造 HTTP 客户端。
func NewHTTPClient(proxy dto.ProxyConfig, timeout time.Duration) (*http.Client, error) {
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		TLSHandshakeTimeout: 15 * time.Second,
	}
	if proxy.Enabled {
		scheme := proxy.Type
		if scheme == "" {
			scheme = "http"
		}
		proxyURL := &url.URL{Scheme: scheme, Host: fmt.Sprintf("%s:%d", proxy.Host, proxy.Port)}
		if proxy.Username != "" {
			proxyURL.User = url.UserPassword(proxy.Username, proxy.Password)
		}
		if _, err := url.Parse(proxyURL.String()); err != nil {
			return nil, fmt.Errorf("%s: %w", i18n.T("tools.err.proxyAddrInvalid"), err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return &http.Client{Transport: transport, Timeout: timeout}, nil
}
