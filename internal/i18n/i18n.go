// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

// Package i18n 提供零依赖的轻量级国际化支持，集中管理用户可见文案。
// 当前支持的语言：zh-CN（默认）、en。前后端共享相同的 key 命名空间。
package i18n

import (
	"fmt"
	"strings"
)

// 支持的语言代码。
const (
	LangZhCN = "zh-CN"
	LangEn   = "en"
)

var supported = map[string]bool{
	LangZhCN: true,
	LangEn:   true,
}

// current 为当前生效的语言，由 app 启动时或用户切换时设置。
var current = LangZhCN

// SetLocale 设置当前语言；仅接受受支持的语言代码，其余按前缀归一化并回退默认。
func SetLocale(lang string) {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		current = LangZhCN
		return
	}
	if supported[lang] {
		current = lang
		return
	}
	// 归一化：zh / zh_CN / zh-cn → zh-CN；en / en_US → en。
	lower := strings.ToLower(lang)
	switch {
	case strings.HasPrefix(lower, "zh"):
		current = LangZhCN
	case strings.HasPrefix(lower, "en"):
		current = LangEn
	default:
		current = LangZhCN
	}
}

// Locale 返回当前语言代码。
func Locale() string { return current }

// T 翻译指定 key；按 当前语言 → zh-CN → key 原文 回退。
// 提供 args 时使用 fmt.Sprintf 格式化（与 %w 错误链无关，仅用于文案插值）。
func T(key string, args ...any) string {
	if msg, ok := messages[current][key]; ok && msg != "" {
		return format(msg, args...)
	}
	if msg, ok := messages[LangZhCN][key]; ok && msg != "" {
		return format(msg, args...)
	}
	return key
}

func format(msg string, args ...any) string {
	if len(args) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, args...)
}
