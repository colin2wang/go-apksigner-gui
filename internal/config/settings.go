// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

// Package config 负责应用设置的持久化：JSON 文件 + 原子写入。
// 说明：密钥库密码、代理外的任何凭据都不落盘。
package config

import (
	"crypto/rand"
	"encoding/base64"
	"path/filepath"
)

// Settings 全局设置。
type Settings struct {
	ToolsDir     string            `json:"toolsDir"`
	LastKeystore string            `json:"lastKeystore"`
	LastAlias    string            `json:"lastAlias"`
	LastApk      string            `json:"lastApk"`
	DefaultV1    bool              `json:"defaultV1"`
	DefaultV2    bool              `json:"defaultV2"`
	DefaultV3    bool              `json:"defaultV3"`
	DefaultV4    bool              `json:"defaultV4"`
	RecentFiles  []string          `json:"recentFiles"`
	RecentAlias  []string          `json:"recentAliases"`
	Mirror       MirrorSetting     `json:"mirror"`
	Proxy        ProxySetting      `json:"proxy"`
	AndroidSDK   string            `json:"androidSdk"`
	Language     string            `json:"language"` // 界面语言：zh-CN（默认）| en
	CleanOnExit  bool              `json:"cleanOnExit"`
	Tools        map[string]string `json:"tools,omitempty"` // 手工指定的工具路径
	// CryptoKey 本地主密钥（base64 编码的 32 字节），用于加密保存的密钥库密码。
	CryptoKey string `json:"cryptoKey,omitempty"`
	// SavedPasswords 按密钥库绝对路径保存的加密密码（base64 密文）。
	SavedPasswords map[string]string `json:"savedPasswords,omitempty"`
}

// MirrorSetting 当前选中的镜像源。
type MirrorSetting struct {
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
}

// ProxySetting 下载代理设置。
type ProxySetting struct {
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type"` // http|https|socks5
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Defaults 返回默认设置。
func Defaults() Settings {
	return Settings{
		ToolsDir:       DefaultToolsDir(),
		DefaultV1:      true,
		DefaultV2:      true,
		DefaultV3:      true,
		DefaultV4:      false,
		RecentFiles:    []string{},
		RecentAlias:    []string{},
		SavedPasswords: map[string]string{},
		Language:       "zh-CN",
		Mirror: MirrorSetting{
			Name:    "Google 官方",
			BaseURL: "https://dl.google.com/android/repository/",
		},
	}
}

// DefaultToolsDir 返回默认工具目录：用户主目录下的 go-apksigner-gui/tools。
func DefaultToolsDir() string {
	base := Dir()
	return filepath.Join(base, "tools")
}

// RecentLimit 保留的最近文件数量。
const RecentLimit = 10

// generateCryptoKey 生成 32 字节随机主密钥的 base64 表示。
func generateCryptoKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("config: 生成主密钥失败: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(b)
}

// AddRecent 追加最近使用的文件（去重、限长）。
func AddRecent(list []string, path string) []string {
	if path == "" {
		return list
	}
	out := []string{path}
	for _, item := range list {
		if item == path {
			continue
		}
		out = append(out, item)
		if len(out) >= RecentLimit {
			break
		}
	}
	return out
}
