package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"go-apksigner-gui/internal/dto"
	"go-apksigner-gui/internal/i18n"
)

// DefaultProxyTestTarget 代理连通性测试的默认目标（Google 主页）。
const DefaultProxyTestTarget = "https://www.google.com"

// TestProxyConnection 通过给定代理访问目标 URL，验证代理连通性并返回结构化结果。
// 即使 proxy.Enabled 为 false，只要填写了主机与端口，也会临时按该代理测试，
// 方便用户在保存设置前验证刚输入的代理是否可用。target 为空时回退默认目标，
// timeout 为 0 时回退默认 15s。
func TestProxyConnection(proxy dto.ProxyConfig, target string, timeout time.Duration) dto.TaskResult {
	if target == "" {
		target = DefaultProxyTestTarget
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	// 未启用但已填写主机+端口时，强制按该代理测试（用户在保存前就想验证刚输入的代理）。
	test := proxy
	if !test.Enabled && test.Host != "" && test.Port != 0 {
		test.Enabled = true
	}
	usingProxy := test.Enabled

	client, err := NewHTTPClient(test, 20*time.Second)
	if err != nil {
		return dto.TaskResult{Success: false, Message: i18n.T("tools.proxy.invalid", err.Error()), Detail: err.Error()}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return dto.TaskResult{Success: false, Message: err.Error()}
	}
	req.Header.Set("User-Agent", "go-apksigner-gui/1.0")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		reason := i18n.T("tools.proxy.unreachable")
		if !usingProxy {
			reason = i18n.T("tools.proxy.directFail")
		}
		return dto.TaskResult{
			Success: false,
			Message: fmt.Sprintf("%s: %s", reason, err.Error()),
			Detail:  err.Error(),
		}
	}
	defer resp.Body.Close()
	// 排空响应体以复用底层连接，避免连接泄漏。
	_, _ = io.Copy(io.Discard, resp.Body)
	elapsed := time.Since(start)

	if resp.StatusCode >= 400 {
		return dto.TaskResult{
			Success: false,
			Message: i18n.T("tools.proxy.httpStatus", resp.StatusCode),
			Detail:  resp.Status,
		}
	}

	scope := i18n.T("tools.proxy.viaProxy")
	if !usingProxy {
		scope = i18n.T("tools.proxy.direct")
	}
	return dto.TaskResult{
		Success: true,
		Message: i18n.T("tools.proxy.result", scope, target, resp.StatusCode, elapsed.Milliseconds()),
	}
}
