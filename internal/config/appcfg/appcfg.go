// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

// Package appcfg 提供应用可外部化的配置加载：从 app.yaml 读取，
// 缺失或解析失败时回退内置默认值（与 app.yaml 同款），保证离线可用。
package appcfg

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"go-apksigner-gui/internal/config"
	"go-apksigner-gui/internal/dto"
)

// SdkManagerConfig 描述 command-line tools（含 sdkmanager）的下载地址。
type SdkManagerConfig struct {
	WindowsURL    string `yaml:"windowsUrl"`
	WindowsSHA256 string `yaml:"windowsSha256"`
	LinuxURL      string `yaml:"linuxUrl"`
	LinuxSHA256   string `yaml:"linuxSha256"`
	MacURL        string `yaml:"macUrl"`
	MacSHA256     string `yaml:"macSha256"`
}

// URLForOS 返回当前系统的 cmdline-tools 下载地址。
func (c SdkManagerConfig) URLForOS(os string) string {
	switch os {
	case "windows":
		return c.WindowsURL
	case "linux":
		return c.LinuxURL
	case "darwin":
		return c.MacURL
	}
	return ""
}

// SHA256ForOS 返回当前系统的校验和（可能为空）。
func (c SdkManagerConfig) SHA256ForOS(os string) string {
	switch os {
	case "windows":
		return c.WindowsSHA256
	case "linux":
		return c.LinuxSHA256
	case "darwin":
		return c.MacSHA256
	}
	return ""
}

// BuildToolsConfig 描述 build-tools 安装相关配置。
type BuildToolsConfig struct {
	DefaultVersion string   `yaml:"defaultVersion"`
	Versions       []string `yaml:"versions"`
}

// ProxyTestConfig 代理连通性测试目标。
type ProxyTestConfig struct {
	Target         string `yaml:"target"`
	TimeoutSeconds int    `yaml:"timeoutSeconds"`
}

// DownloadConfig 下载相关参数。
type DownloadConfig struct {
	TimeoutSeconds int `yaml:"timeoutSeconds"`
	Retries        int `yaml:"retries"`
}

// Config 外部化配置根结构。
type Config struct {
	SdkManager SdkManagerConfig   `yaml:"sdkmanager"`
	BuildTools BuildToolsConfig   `yaml:"buildTools"`
	Mirrors    []dto.MirrorSource `yaml:"mirrors"`
	ProxyTest  ProxyTestConfig    `yaml:"proxyTest"`
	Download   DownloadConfig     `yaml:"download"`
}

// DefaultConfig 返回内置默认值（与 app.yaml 同款）。
func DefaultConfig() *Config {
	return &Config{
		SdkManager: SdkManagerConfig{
			WindowsURL: "https://dl.google.com/android/repository/commandlinetools-win-15859902_latest.zip",
		},
		BuildTools: BuildToolsConfig{
			DefaultVersion: "36.1.0",
			Versions:       []string{"36.1.0", "35.0.1", "34.0.0", "33.0.2"},
		},
		Mirrors: []dto.MirrorSource{
			{Name: "Google 官方", BaseURL: "https://dl.google.com/android/repository/", Note: "需要可访问 Google 服务"},
			{Name: "腾讯云镜像", BaseURL: "https://mirrors.cloud.tencent.com/AndroidSDK/", Note: "国内推荐"},
			{Name: "清华大学镜像", BaseURL: "https://mirrors.tuna.tsinghua.edu.cn/android/repository/", Note: "国内推荐"},
			{Name: "大连东软镜像", BaseURL: "https://mirrors.neusoft.edu.cn/android/repository/", Note: "教育网推荐"},
			{Name: "中科大镜像", BaseURL: "https://mirrors.ustc.edu.cn/android/repository/", Note: "国内备选"},
		},
		ProxyTest: ProxyTestConfig{Target: "https://www.google.com", TimeoutSeconds: 15},
		Download:  DownloadConfig{TimeoutSeconds: 1800, Retries: 3},
	}
}

// Load 依次尝试 paths 中的文件，读取首个存在且可解析的文件并覆盖默认值；
// 全部缺失或解析失败时返回内置默认值（绝不阻断启动）。
func Load(paths ...string) *Config {
	cfg := DefaultConfig()
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			continue
		}
		return cfg
	}
	return cfg
}

// DefaultPaths 返回常见的 app.yaml 查找路径：exe 同目录、配置目录、当前工作目录。
func DefaultPaths() []string {
	paths := []string{}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "app.yaml"))
	}
	if dir := config.Dir(); dir != "" {
		paths = append(paths, filepath.Join(dir, "app.yaml"))
	}
	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, "app.yaml"))
	}
	return paths
}
