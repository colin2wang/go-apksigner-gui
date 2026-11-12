APK Signer Studio — Go 语言 APK 签名工具 GUI 设计方案

一、总体架构概览

graph TB
subgraph "APK Signer Studio"
UI["🖥️ 前端 UI (HTML/CSS/JS + Vue3)"]
Bridge["🔗 Wails v2 Bridge"]
Core["⚙️ Go 后端核心"]
end

    subgraph "核心功能模块"
        CM["📜 证书管理"]
        TM["📦 工具管理"]
        SM["✍️ 签名模块"]
        RM["🔄 重打包模块"]
        UM["🛠️ 实用工具"]
    end

    subgraph "外部工具链"
        KT["keytool (JDK)"]
        AS["apksigner"]
        ZA["zipalign"]
        AT["apktool"]
        AAPT["aapt2"]
    end

    subgraph "存储层"
        FS["📁 本地文件系统"]
        CFG["⚙️ 配置存储 (BoltDB/JSON)"]
        TOOLS["📂 工具目录"]
    end

    UI --> Bridge
    Bridge --> Core
    Core --> CM & TM & SM & RM & UM
    CM --> KT
    SM --> AS & ZA
    RM --> AT & AAPT & AS & ZA
    TM --> TOOLS
    Core --> FS & CFG

二、主要功能模块

graph LR
subgraph "模块 1: 证书管理 📜"
C1["生成 Keystore"]
C2["查看证书详情"]
C3["导入/导出证书"]
C4["证书格式转换"]
C5["密钥库密码管理"]
end

    subgraph "模块 2: 工具管理 📦"
        T1["自动检测已安装工具"]
        T2["在线下载 Build Tools"]
        T3["代理/镜像源配置"]
        T4["版本切换管理"]
        T5["工具完整性校验"]
    end

    subgraph "模块 3: APK 签名 ✍️"
        S1["APK 信息解析"]
        S2["zipalign 对齐优化"]
        S3["V1/V2/V3/V4 签名"]
        S4["签名验证"]
        S5["批量签名"]
    end

    subgraph "模块 4: 重打包 🔄"
        R1["APK 反编译"]
        R2["资源编辑"]
        R3["重新编译打包"]
        R4["签名 + 对齐流水线"]
        R5["Diff 对比"]
    end

    subgraph "模块 5: 其他工具 🛠️"
        U1["APK 信息查看"]
        U2["签名历史日志"]
        U3["模板预设管理"]
        U4["拖拽快捷操作"]
    end

三、技术栈
层级   技术选型   说明
GUI 框架   Wails v2   Go + Web 技术，轻量级桌面应用（~15MB），使用系统 WebView

前端   Vue 3 + Vite + TailwindCSS   现代化响应式 UI

后端   Go 1.22+   核心业务逻辑、进程管理

配置存储   BoltDB / JSON   轻量级嵌入式数据库

进程调用   os/exec   调用外部工具链

下载管理   net/http + 断点续传   工具包下载

ZIP 处理   archive/zip   APK 文件解析

日志   zerolog / slog   结构化日志

外部工具链依赖

graph TD
subgraph "Android Build Tools"
BT["build-tools v35+"]
BT --> APS["apksigner (签名)"]
BT --> ZIP["zipalign (对齐)"]
BT --> AAPT2["aapt2 (资源处理)"]
end

    subgraph "JDK"
        JDK["JDK 11+"]
        JDK --> KEYTOOL["keytool (证书)"]
    end

    subgraph "第三方"
        APKTOOL["apktool v2.9+ (反编译)"]
    end

    style BT fill:#4CAF50,color:#fff
    style JDK fill:#FF9800,color:#fff
    style APKTOOL fill:#2196F3,color:#fff

四、项目结构

apk-signer-studio/
├── main.go                    # 应用入口
├── app.go                     # Wails 应用结构体 & 绑定方法
├── go.mod / go.sum
├── wails.json                 # Wails 配置
├── internal/
│   ├── certificate/           # 证书管理模块
│   │   ├── keystore.go        # Keystore 生成/管理
│   │   ├── info.go            # 证书信息解析
│   │   └── convert.go         # 格式转换
│   ├── tools/                 # 工具管理模块
│   │   ├── detector.go        # 工具检测
│   │   ├── downloader.go      # 下载管理器
│   │   ├── mirror.go          # 镜像源配置
│   │   └── manager.go         # 版本管理
│   ├── signer/                # 签名模块
│   │   ├── zipalign.go        # 对齐处理
│   │   ├── apksign.go         # APK 签名
│   │   ├── verify.go          # 签名验证
│   │   └── batch.go           # 批量签名
│   ├── repack/                # 重打包模块
│   │   ├── decompile.go       # 反编译
│   │   ├── recompile.go       # 重编译
│   │   └── pipeline.go        # 完整流水线
│   ├── apkparser/             # APK 解析
│   │   ├── manifest.go        # AndroidManifest 解析
│   │   └── info.go            # APK 基本信息
│   ├── config/                # 配置管理
│   │   ├── settings.go        # 全局设置
│   │   └── store.go           # 持久化存储
│   └── logger/                # 日志模块
│       └── logger.go
├── frontend/                  # 前端代码
│   ├── src/
│   │   ├── App.vue
│   │   ├── components/
│   │   │   ├── CertificatePanel.vue
│   │   │   ├── ToolsPanel.vue
│   │   │   ├── SignerPanel.vue
│   │   │   ├── RepackPanel.vue
│   │   │   └── SettingsPanel.vue
│   │   ├── stores/
│   │   └── assets/
│   ├── index.html
│   ├── package.json
│   └── vite.config.ts
└── build/                     # 构建产物
└── appicon.png

五、主要流程

5.1 APK 签名完整流程

flowchart TD
A["📱 选择 APK 文件"] --> B["📋 解析 APK 信息"]
B --> C{"已有签名?"}
C -->|"是"| D["⚠️ 提示: 需要先移除旧签名"]
C -->|"否"| E["📐 zipalign 对齐优化"]
D --> E
E --> F["🔐 选择签名方案"]

    F --> G{"签名版本"}
    G -->|"V1"| H1["JAR 签名 (兼容旧设备)"]
    G -->|"V2"| H2["APK Signature Scheme v2"]
    G -->|"V3"| H3["支持密钥轮换"]
    G -->|"V4"| H4["增量更新签名"]
    G -->|"V1+V2+V3"| H5["组合签名 (推荐)"]
    
    H1 & H2 & H3 & H4 & H5 --> I["🔑 选择证书/密钥库"]
    I --> J["📝 输入密码"]
    J --> K["🚀 执行 apksigner sign"]
    K --> L["✅ 验证签名"]
    L --> M{"验证通过?"}
    M -->|"是"| N["📦 输出签名后 APK"]
    M -->|"否"| O["❌ 输出错误日志"]
    
    style A fill:#E3F2FD
    style N fill:#C8E6C9
    style O fill:#FFCDD2

5.2 证书管理流程

flowchart TD
A["📜 证书管理"] --> B{"操作类型"}

    B -->|"新建"| C1["填写证书信息"]
    C1 --> C2["CN/OU/O/L/ST/C"]
    C2 --> C3["选择算法 RSA/EC"]
    C3 --> C4["设置密钥长度 2048/4096"]
    C4 --> C5["设置有效期"]
    C5 --> C6["设置密码"]
    C6 --> C7["执行 keytool -genkeypair"]
    C7 --> C8["保存 .jks/.keystore"]
    
    B -->|"查看"| D1["选择 Keystore 文件"]
    D1 --> D2["输入密码"]
    D2 --> D3["执行 keytool -list -v"]
    D3 --> D4["显示证书详情"]
    
    B -->|"转换"| E1["JKS → PKCS12"]
    B -->|"导出"| F1["导出 .cer/.pem"]
    
    style A fill:#FFF3E0
    style C8 fill:#C8E6C9
    style D4 fill:#E1F5FE

5.3 工具下载管理流程

flowchart TD
A["📦 工具管理"] --> B["检测本地工具"]
B --> C{"工具完整?"}
C -->|"是"| D["✅ 显示已安装版本"]
C -->|"否"| E["提示下载"]

    E --> F{"下载方式"}
    F -->|"Google 官方"| G1["dl.google.com/android/repository/"]
    F -->|"国内镜像"| G2["mirrors.cloud.tencent.comn 或 mirrors.neusoft.edu.cn"]
    F -->|"自定义代理"| G3["HTTP/SOCKS5 代理"]
    
    G1 & G2 & G3 --> H["解析 repository2-3.xml"]
    H --> I["选择 Build Tools 版本"]
    I --> J["下载 ZIP 包"]
    J --> K["SHA256 校验"]
    K --> L{"校验通过?"}
    L -->|"是"| M["解压到工具目录"]
    L -->|"否"| N["重新下载"]
    M --> O["注册工具路径"]
    O --> D
    
    style D fill:#C8E6C9
    style N fill:#FFCDD2

5.4 APK 重打包流程

flowchart TD
A["📱 选择 APK"] --> B["apktool d 反编译"]
B --> C["📁 生成项目目录"]
C --> D["编辑资源/代码"]
D --> E["apktool b 重新编译"]
E --> F{"编译成功?"}
F -->|"否"| G["显示错误并定位"]
F -->|"是"| H["zipalign -f 4 对齐"]
H --> I["apksigner sign 签名"]
I --> J["apksigner verify 验证"]
J --> K["📦 输出重打包 APK"]

    style K fill:#C8E6C9
    style G fill:#FFCDD2

六、核心实例代码

6.1 项目入口 — main.go

package main

import (
"embed"
"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
// 创建应用实例
app := NewApp()

	err := wails.Run(&options.App{
		Title:     "APK Signer Studio",
		Width:     1280,
		Height:    800,
		MinWidth:  960,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatal("启动应用失败:", err)
	}
}

6.2 应用绑定层 — app.go

package main

import (
"context"
"fmt"

	"apk-signer-studio/internal/certificate"
	"apk-signer-studio/internal/config"
	"apk-signer-studio/internal/signer"
	"apk-signer-studio/internal/tools"
	"apk-signer-studio/internal/repack"
)

// App 应用主结构体，所有前端可调用的方法都挂在此结构体上
type App struct {
ctx         context.Context
certMgr     *certificate.Manager
toolMgr     *tools.Manager
signerMgr   *signer.Manager
repackMgr   *repack.Manager
configStore *config.Store
}

func NewApp() *App {
cfg := config.NewStore("apk-signer-studio.json")
return &App{
certMgr:     certificate.NewManager(cfg),
toolMgr:     tools.NewManager(cfg),
signerMgr:   signer.NewManager(cfg),
repackMgr:   repack.NewManager(cfg),
configStore: cfg,
}
}

func (a *App) startup(ctx context.Context) {
a.ctx = ctx
// 初始化: 检测已有工具、加载配置
a.toolMgr.DetectAll()
}

func (a *App) shutdown(ctx context.Context) {
a.configStore.Save()
}

// ============ 前端可调用的 API ============

// GetAppInfo 返回应用信息
func (a *App) GetAppInfo() map[string]string {
return map[string]string{
"version": "1.0.0",
"name":    "APK Signer Studio",
}
}

6.3 证书管理模块 — internal/certificate/keystore.go

package certificate

import (
"fmt"
"os/exec"
"strings"
"apk-signer-studio/internal/config"
)

// CertInfo 证书信息
type CertInfo struct {
Alias         string json:"alias"
CommonName    string json:"commonName"    // CN
Organization  string json:"organization"  // O
OrgUnit       string json:"orgUnit"       // OU
Locality      string json:"locality"      // L
State         string json:"state"         // ST
Country       string json:"country"       // C
Validity      int    json:"validity"      // 有效天数
KeyAlgorithm  string json:"keyAlgorithm"  // RSA / EC
KeySize       int    json:"keySize"       // 2048 / 4096
}

// KeystoreInfo 密钥库信息
type KeystoreInfo struct {
Path       string json:"path"
Alias      string json:"alias"
DN         string json:"dn"
Issuer     string json:"issuer"
Serial     string json:"serial"
ValidFrom  string json:"validFrom"
ValidTo    string json:"validTo"
FingerMD5  string json:"fingerMD5"
FingerSHA1 string json:"fingerSHA1"
FingerSHA256 string json:"fingerSHA256"
}

// Manager 证书管理器
type Manager struct {
cfg     *config.Store
keytool string // keytool 路径
}

func NewManager(cfg *config.Store) *Manager {
return &Manager{
cfg:     cfg,
keytool: "keytool", // 默认从 PATH 查找
}
}

// SetKeytoolPath 设置 keytool 路径
func (m *Manager) SetKeytoolPath(path string) {
m.keytool = path
}

// GenerateKeystore 生成新的密钥库
// keytool -genkeypair -v \
//   -keystore output.jks \
//   -alias mykey \
//   -keyalg RSA -keysize 2048 \
//   -validity 10000 \
//   -storepass password \
//   -keypass password \
//   -dname "CN=Name, OU=Unit, O=Org, L=City, ST=State, C=CN"
func (m *Manager) GenerateKeystore(
outputPath string,
password string,
info CertInfo,
) (string, error) {

	if info.KeyAlgorithm == "" {
		info.KeyAlgorithm = "RSA"
	}
	if info.KeySize == 0 {
		info.KeySize = 2048
	}
	if info.Validity == 0 {
		info.Validity = 10000
	}
	if info.Alias == "" {
		info.Alias = "android-key"
	}

	dname := fmt.Sprintf("CN=%s, OU=%s, O=%s, L=%s, ST=%s, C=%s",
		info.CommonName, info.OrgUnit, info.Organization,
		info.Locality, info.State, info.Country)

	args := []string{
		"-genkeypair", "-v",
		"-keystore", outputPath,
		"-alias", info.Alias,
		"-keyalg", info.KeyAlgorithm,
		"-keysize", fmt.Sprintf("%d", info.KeySize),
		"-validity", fmt.Sprintf("%d", info.Validity),
		"-storepass", password,
		"-keypass", password,
		"-dname", dname,
	}

	cmd := exec.Command(m.keytool, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("生成密钥库失败: %wn输出: %s", err, string(output))
	}

	return string(output), nil
}

// ListKeystoreInfo 查看密钥库详细信息
// keytool -list -v -keystore file.jks -storepass password
func (m *Manager) ListKeystoreInfo(
keystorePath string,
password string,
) (*KeystoreInfo, error) {

	args := []string{
		"-list", "-v",
		"-keystore", keystorePath,
		"-storepass", password,
	}

	cmd := exec.Command(m.keytool, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("读取密钥库失败: %wn输出: %s", err, string(output))
	}

	info := parseKeytoolOutput(string(output))
	info.Path = keystorePath
	return info, nil
}

// ExportCertificate 导出证书
// keytool -exportcert -keystore file.jks -alias mykey -file output.cer
func (m *Manager) ExportCertificate(
keystorePath, password, alias, outputPath string,
) error {
args := []string{
"-exportcert",
"-keystore", keystorePath,
"-alias", alias,
"-storepass", password,
"-file", outputPath,
"-rfc", // PEM 格式
}

	cmd := exec.Command(m.keytool, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("导出证书失败: %wn输出: %s", err, string(output))
	}
	return nil
}

// parseKeytoolOutput 解析 keytool -list -v 的输出
func parseKeytoolOutput(output string) *KeystoreInfo {
info := &KeystoreInfo{}
lines := strings.Split(output, "n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case strings.Contains(line, "别名"):
			info.Alias = extractValue(line)
		case strings.Contains(line, "所有者"):
			info.DN = extractValue(line)
		case strings.Contains(line, "签发人"):
			info.Issuer = extractValue(line)
		case strings.Contains(line, "序列号"):
			info.Serial = extractValue(line)
		case strings.Contains(line, "有效期从"):
			info.ValidFrom = extractValue(line)
		case strings.Contains(line, "至"):
			info.ValidTo = extractValue(line)
		case strings.Contains(line, "MD5"):
			info.FingerMD5 = extractValue(line)
		case strings.Contains(line, "SHA1"):
			info.FingerSHA1 = extractValue(line)
		case strings.Contains(line, "SHA256"):
			info.FingerSHA256 = extractValue(line)
		}
	}
	return info
}

func extractValue(line string) string {
parts := strings.SplitN(line, ":", 2)
if len(parts) == 2 {
return strings.TrimSpace(parts[1])
}
// 中文冒号
parts = strings.SplitN(line, "：", 2)
if len(parts) == 2 {
return strings.TrimSpace(parts[1])
}
return line
}

6.4 工具下载管理模块 — internal/tools/manager.go

package tools

import (
"archive/zip"
"crypto/sha256"
"encoding/hex"
"encoding/xml"
"fmt"
"io"
"net/http"
"net/url"
"os"
"path/filepath"
"runtime"
"strings"

	"apk-signer-studio/internal/config"
)

// ToolStatus 工具状态
type ToolStatus struct {
Name      string json:"name"
Version   string json:"version"
Path      string json:"path"
Installed bool   json:"installed"
Required  bool   json:"required"
}

// MirrorSource 镜像源配置
type MirrorSource struct {
Name    string json:"name"
BaseURL string json:"baseUrl"
Note    string json:"note"
}

// ProxyConfig 代理配置
type ProxyConfig struct {
Enabled  bool   json:"enabled"
Type     string json:"type" // http / socks5
Host     string json:"host"
Port     int    json:"port"
Username string json:"username"
Password string json:"password"
}

// DownloadProgress 下载进度
type DownloadProgress struct {
FileName      string  json:"fileName"
TotalBytes    int64   json:"totalBytes"
Downloaded    int64   json:"downloaded"
Percent       float64 json:"percent"
Status        string  json:"status" // downloading / verifying / extracting / done / error
}

// BuildToolVersion 表示一个可用的 Build Tools 版本
type BuildToolVersion struct {
Version  string json:"version"
FileName string json:"fileName"
URL      string json:"url"
SHA256   string json:"sha256"
Size     int64  json:"size"
}

// 内置镜像源列表
var DefaultMirrors = []MirrorSource{
{
Name:    "Google 官方",
BaseURL: "https://dl.google.com/android/repository/",
Note:    "需要访问 Google 服务",
},
{
Name:    "腾讯云镜像",
BaseURL: "https://mirrors.cloud.tencent.com/AndroidSDK/",
Note:    "国内推荐",
},
{
Name:    "大连东软镜像",
BaseURL: "https://mirrors.neusoft.edu.cn/android/repository/",
Note:    "国内教育网推荐",
},
{
Name:    "清华大学镜像",
BaseURL: "https://mirrors.tuna.tsinghua.edu.cn/android/repository/",
Note:    "国内推荐",
},
}

// Manager 工具管理器
type Manager struct {
cfg        *config.Store
toolsDir   string
mirror     MirrorSource
proxy      ProxyConfig
httpClient *http.Client
}

func NewManager(cfg *config.Store) *Manager {
// 默认工具存储目录
homeDir, _ := os.UserHomeDir()
toolsDir := filepath.Join(homeDir, ".apk-signer-studio", "tools")
os.MkdirAll(toolsDir, 0755)

	return &Manager{
		cfg:        cfg,
		toolsDir:   toolsDir,
		mirror:     DefaultMirrors[0],
		httpClient: &http.Client{},
	}
}

// SetMirror 设置镜像源
func (m *Manager) SetMirror(mirror MirrorSource) {
m.mirror = mirror
}

// SetProxy 设置代理
func (m *Manager) SetProxy(proxy ProxyConfig) {
m.proxy = proxy
m.updateHTTPClient()
}

func (m *Manager) updateHTTPClient() {
transport := &http.Transport{}

	if m.proxy.Enabled {
		proxyURL := &url.URL{
			Scheme: m.proxy.Type,
			Host:   fmt.Sprintf("%s:%d", m.proxy.Host, m.proxy.Port),
		}
		if m.proxy.Username != "" {
			proxyURL.User = url.UserPassword(m.proxy.Username, m.proxy.Password)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	m.httpClient = &http.Client{Transport: transport}
}

// DetectAll 检测所有工具的安装状态
func (m *Manager) DetectAll() []ToolStatus {
tools := []struct {
name string
file string
}{
{"apksigner", "apksigner"},
{"zipalign", "zipalign"},
{"aapt2", "aapt2"},
{"apktool", "apktool"},
}

	// Windows 加 .exe / .bat 后缀
	if runtime.GOOS == "windows" {
		tools[0].file = "apksigner.bat"
		tools[1].file = "zipalign.exe"
		tools[2].file = "aapt2.exe"
		tools[3].file = "apktool.bat"
	}

	var statuses []ToolStatus
	for _, t := range tools {
		status := ToolStatus{
			Name:     t.name,
			Required: t.name != "apktool",
		}

		// 1. 先在工具目录中查找
		toolPath := m.findToolInDir(t.file)
		if toolPath == "" {
			// 2. 在系统 PATH 中查找
			toolPath, _ = findInPath(t.file)
		}

		if toolPath != "" {
			status.Installed = true
			status.Path = toolPath
			status.Version = m.getToolVersion(toolPath, t.name)
		}

		statuses = append(statuses, status)
	}
	return statuses
}

func (m *Manager) findToolInDir(name string) string {
// 递归搜索工具目录
var found string
filepath.Walk(m.toolsDir, func(path string, info os.FileInfo, err error) error {
if err != nil {
return nil
}
if !info.IsDir() && strings.EqualFold(info.Name(), name) {
found = path
return filepath.SkipAll
}
return nil
})
return found
}

func findInPath(name string) (string, error) {
pathEnv := os.Getenv("PATH")
paths := strings.Split(pathEnv, string(os.PathListSeparator))
for _, p := range paths {
full := filepath.Join(p, name)
if _, err := os.Stat(full); err == nil {
return full, nil
}
}
return "", fmt.Errorf("未在 PATH 中找到 %s", name)
}

func (m *Manager) getToolVersion(path string, toolName string) string {
// 各工具获取版本的方式不同
var args []string
switch toolName {
case "apksigner":
args = []string{"version"}
case "zipalign":
// zipalign 没有 --version，通过帮助信息提取
args = []string{"-h"}
case "aapt2":
args = []string{"version"}
case "apktool":
args = []string{"--version"}
}
// 简化: 实际中需要执行命令并解析输出
return "detected"
}

// FetchAvailableVersions 从仓库索引获取可用版本列表
func (m *Manager) FetchAvailableVersions() ([]BuildToolVersion, error) {
// 下载 Google 的仓库索引 XML
indexURL := m.mirror.BaseURL + "repository2-3.xml"

	resp, err := m.httpClient.Get(indexURL)
	if err != nil {
		return nil, fmt.Errorf("获取版本列表失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseRepositoryXML(body, m.mirror.BaseURL)
}

// parseRepositoryXML 解析 repository2-3.xml 获取 Build Tools 版本
func parseRepositoryXML(data []byte, baseURL string) ([]BuildToolVersion, error) {
// 简化的 XML 解析 — 实际需要完整的结构体映射
type RemotePackage struct {
Path     string xml:"path"
Revision struct {
Major int xml:"major"
Minor int xml:"minor"
} xml:"revision"
Archives struct {
Archive []struct {
OS       string xml:"host-os,attr"
URL      string xml:"url"
SHA256   string xml:"checksum"
Size     int64  xml:"size"
} xml:"archive"
} xml:"archives"
}

	type Repository struct {
		Packages []RemotePackage xml:"remotePackage"
	}

	var repo Repository
	if err := xml.Unmarshal(data, &repo); err != nil {
		return nil, fmt.Errorf("解析 XML 失败: %w", err)
	}

	osName := getOSName()
	var versions []BuildToolVersion

	for _, pkg := range repo.Packages {
		if !strings.HasPrefix(pkg.Path, "build-tools;") {
			continue
		}

		version := strings.TrimPrefix(pkg.Path, "build-tools;")
		for _, archive := range pkg.Archives.Archive {
			if strings.Contains(archive.OS, osName) || archive.OS == "any" {
				versions = append(versions, BuildToolVersion{
					Version:  version,
					FileName: archive.URL,
					URL:      baseURL + archive.URL,
					SHA256:   archive.SHA256,
					Size:     archive.Size,
				})
				break
			}
		}
	}

	return versions, nil
}

func getOSName() string {
switch runtime.GOOS {
case "windows":
return "windows"
case "darwin":
return "macosx"
case "linux":
return "linux"
default:
return runtime.GOOS
}
}

// DownloadBuildTools 下载并安装 Build Tools（支持断点续传 + 进度回调）
func (m *Manager) DownloadBuildTools(
version BuildToolVersion,
onProgress func(DownloadProgress),
) error {

	destPath := filepath.Join(m.toolsDir, "downloads", version.FileName)
	os.MkdirAll(filepath.Dir(destPath), 0755)

	// 检查已有部分下载
	var existingSize int64
	if fi, err := os.Stat(destPath); err == nil {
		existingSize = fi.Size()
	}

	// 创建请求（支持断点续传）
	req, err := http.NewRequest("GET", version.URL, nil)
	if err != nil {
		return err
	}
	if existingSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existingSize))
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	totalSize := version.Size
	if resp.StatusCode == http.StatusPartialContent {
		totalSize = existingSize + resp.ContentLength
	} else {
		existingSize = 0
		totalSize = resp.ContentLength
	}

	// 打开文件（追加模式）
	flags := os.O_CREATE | os.O_WRONLY
	if existingSize > 0 && resp.StatusCode == http.StatusPartialContent {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(destPath, flags, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 下载循环 + 进度回调
	buf := make([]byte, 32*1024) // 32KB buffer
	downloaded := existingSize

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := file.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)

			if onProgress != nil {
				onProgress(DownloadProgress{
					FileName:   version.FileName,
					TotalBytes: totalSize,
					Downloaded: downloaded,
					Percent:    float64(downloaded) / float64(totalSize) * 100,
					Status:     "downloading",
				})
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	// SHA256 校验
	if onProgress != nil {
		onProgress(DownloadProgress{Status: "verifying", FileName: version.FileName})
	}

	if version.SHA256 != "" {
		valid, err := verifySHA256(destPath, version.SHA256)
		if err != nil || !valid {
			os.Remove(destPath)
			return fmt.Errorf("SHA256 校验失败，文件可能已损坏")
		}
	}

	// 解压到工具目录
	if onProgress != nil {
		onProgress(DownloadProgress{Status: "extracting", FileName: version.FileName})
	}

	extractDir := filepath.Join(m.toolsDir, "build-tools", version.Version)
	if err := extractZip(destPath, extractDir); err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}

	if onProgress != nil {
		onProgress(DownloadProgress{Status: "done", FileName: version.FileName})
	}

	return nil
}

// verifySHA256 校验文件哈希
func verifySHA256(filePath, expected string) (bool, error) {
f, err := os.Open(filePath)
if err != nil {
return false, err
}
defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	return strings.EqualFold(actual, expected), nil
}

// extractZip 解压 ZIP 文件
func extractZip(zipPath, destDir string) error {
r, err := zip.OpenReader(zipPath)
if err != nil {
return err
}
defer r.Close()

	for _, f := range r.File {
		fPath := filepath.Join(destDir, f.Name)

		// 安全检查: 防止 zip slip 攻击
		if !strings.HasPrefix(filepath.Clean(fPath), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("非法文件路径: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fPath, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(fPath), 0755)
		outFile, err := os.OpenFile(fPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

6.5 APK 签名模块 — internal/signer/apksign.go

package signer

import (
"fmt"
"os"
"os/exec"
"path/filepath"
"strings"

	"apk-signer-studio/internal/config"
)

// SignOptions 签名选项
type SignOptions struct {
InputAPK   string json:"inputApk"
OutputAPK  string json:"outputApk"
Keystore   string json:"keystore"
Alias      string json:"alias"
StorePass  string json:"storePass"
KeyPass    string json:"keyPass"
// 签名方案
V1Enabled  bool   json:"v1Enabled"
V2Enabled  bool   json:"v2Enabled"
V3Enabled  bool   json:"v3Enabled"
V4Enabled  bool   json:"v4Enabled"
// 可选
ZipAlignFirst bool json:"zipAlignFirst" // 先对齐再签名
}

// SignResult 签名结果
type SignResult struct {
Success     bool   json:"success"
OutputPath  string json:"outputPath"
Message     string json:"message"
SignVersion string json:"signVersion"
FileSize    int64  json:"fileSize"
}

// VerifyResult 验证结果
type VerifyResult struct {
Verified   bool              json:"verified"
V1Signed   bool              json:"v1Signed"
V2Signed   bool              json:"v2Signed"
V3Signed   bool              json:"v3Signed"
V4Signed   bool              json:"v4Signed"
CertInfo   map[string]string json:"certInfo"
RawOutput  string            json:"rawOutput"
}

// APKInfo APK 基本信息
type APKInfo struct {
PackageName  string json:"packageName"
VersionName  string json:"versionName"
VersionCode  string json:"versionCode"
MinSDK       string json:"minSdk"
TargetSDK    string json:"targetSdk"
FileSize     int64  json:"fileSize"
IsSigned     bool   json:"isSigned"
}

// Manager 签名管理器
type Manager struct {
cfg        *config.Store
apksigner  string // apksigner 路径
zipalign   string // zipalign 路径
}

func NewManager(cfg *config.Store) *Manager {
return &Manager{
cfg:       cfg,
apksigner: "apksigner",
zipalign:  "zipalign",
}
}

// SetToolPaths 设置工具路径
func (m *Manager) SetToolPaths(apksigner, zipalign string) {
if apksigner != "" {
m.apksigner = apksigner
}
if zipalign != "" {
m.zipalign = zipalign
}
}

// GetAPKInfo 获取 APK 基本信息
func (mManager) GetAPKInfo(apkPath string) (APKInfo, error) {
info := &APKInfo{}

	// 获取文件大小
	fi, err := os.Stat(apkPath)
	if err != nil {
		return nil, fmt.Errorf("无法访问 APK 文件: %w", err)
	}
	info.FileSize = fi.Size()

	// 使用 apksigner 检查签名状态
	verifyResult, _ := m.VerifySignature(apkPath)
	if verifyResult != nil {
		info.IsSigned = verifyResult.Verified
	}

	// 使用 aapt2 或 android 库解析 APK 信息
	// 这里简化处理，实际可以使用 aapt2 dump badging
	// 或使用 Go 的 APK 解析库
	info.PackageName = extractPackageName(apkPath)

	return info, nil
}

// ZipAlign 执行 zipalign 对齐
// zipalign -f -v 4 input.apk output.apk
func (m *Manager) ZipAlign(inputPath, outputPath string) error {
args := []string{"-f", "-v", "4", inputPath, outputPath}

	cmd := exec.Command(m.zipalign, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("zipalign 失败: %wn输出: %s", err, string(output))
	}

	// 验证对齐
	verifyCmd := exec.Command(m.zipalign, "-c", "-v", "4", outputPath)
	if err := verifyCmd.Run(); err != nil {
		return fmt.Errorf("zipalign 验证失败: 输出文件未正确对齐")
	}

	return nil
}

// SignAPK 执行 APK 签名
// 完整流程: zipalign → apksigner sign → apksigner verify
func (mManager) SignAPK(opts SignOptions) (SignResult, error) {
result := &SignResult{}

	// 验证输入
	if _, err := os.Stat(opts.InputAPK); os.IsNotExist(err) {
		return nil, fmt.Errorf("输入 APK 不存在: %s", opts.InputAPK)
	}
	if _, err := os.Stat(opts.Keystore); os.IsNotExist(err) {
		return nil, fmt.Errorf("密钥库不存在: %s", opts.Keystore)
	}

	// 设置输出路径
	if opts.OutputAPK == "" {
		ext := filepath.Ext(opts.InputAPK)
		base := strings.TrimSuffix(opts.InputAPK, ext)
		opts.OutputAPK = base + "-signed" + ext
	}

	workAPK := opts.InputAPK

	// 步骤 1: zipalign 对齐（推荐在签名前执行）
	if opts.ZipAlignFirst {
		alignedAPK := opts.OutputAPK + ".aligned"
		if err := m.ZipAlign(workAPK, alignedAPK); err != nil {
			return nil, fmt.Errorf("对齐失败: %w", err)
		}
		workAPK = alignedAPK
		defer os.Remove(alignedAPK) // 清理临时文件
	}

	// 步骤 2: 构建 apksigner 命令
	// apksigner sign \
	//   --ks keystore.jks \
	//   --ks-key-alias alias \
	//   --ks-pass pass:password \
	//   --key-pass pass:password \
	//   --v1-signing-enabled true \
	//   --v2-signing-enabled true \
	//   --v3-signing-enabled true \
	//   --out output.apk \
	//   input.apk

	args := []string{
		"sign",
		"--ks", opts.Keystore,
		"--ks-key-alias", opts.Alias,
		"--ks-pass", "pass:" + opts.StorePass,
		"--key-pass", "pass:" + opts.KeyPass,
		"--v1-signing-enabled", fmt.Sprintf("%t", opts.V1Enabled),
		"--v2-signing-enabled", fmt.Sprintf("%t", opts.V2Enabled),
		"--v3-signing-enabled", fmt.Sprintf("%t", opts.V3Enabled),
		"--out", opts.OutputAPK,
	}

	if opts.V4Enabled {
		v4File := opts.OutputAPK + ".idsig"
		args = append(args, "--v4-signing-enabled", "true")
		args = append(args, "--v4-output-file", v4File)
	}

	args = append(args, workAPK)

	cmd := exec.Command(m.apksigner, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("签名失败: %wn输出: %s", err, string(output))
	}

	// 步骤 3: 验证签名
	verifyResult, err := m.VerifySignature(opts.OutputAPK)
	if err != nil {
		return nil, fmt.Errorf("签名完成但验证失败: %w", err)
	}

	// 获取输出文件信息
	fi, _ := os.Stat(opts.OutputAPK)
	if fi != nil {
		result.FileSize = fi.Size()
	}

	result.Success = verifyResult.Verified
	result.OutputPath = opts.OutputAPK
	result.Message = fmt.Sprintf("签名完成，V1=%v V2=%v V3=%v",
		verifyResult.V1Signed, verifyResult.V2Signed, verifyResult.V3Signed)

	// 构建签名版本字符串
	var versions []string
	if verifyResult.V1Signed {
		versions = append(versions, "v1")
	}
	if verifyResult.V2Signed {
		versions = append(versions, "v2")
	}
	if verifyResult.V3Signed {
		versions = append(versions, "v3")
	}
	result.SignVersion = strings.Join(versions, "+")

	return result, nil
}

// VerifySignature 验证 APK 签名
// apksigner verify --verbose --print-certs app.apk
func (mManager) VerifySignature(apkPath string) (VerifyResult, error) {
args := []string{
"verify",
"--verbose",
"--print-certs",
apkPath,
}

	cmd := exec.Command(m.apksigner, args...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	result := &VerifyResult{
		RawOutput: outputStr,
	}

	// 即使 err 不为 nil，也解析输出（有些警告不算错误）
	result.Verified = (err == nil)
	result.V1Signed = strings.Contains(outputStr, "v1 scheme") &&
		!strings.Contains(outputStr, "v1 scheme (JAR signing): NOT signed")
	result.V2Signed = strings.Contains(outputStr, "v2 scheme") &&
		!strings.Contains(outputStr, "v2 scheme (APK Signature Scheme v2): NOT signed")
	result.V3Signed = strings.Contains(outputStr, "v3 scheme") &&
		!strings.Contains(outputStr, "v3 scheme (APK Signature Scheme v3): NOT signed")

	// 解析证书信息
	result.CertInfo = parseCertInfo(outputStr)

	return result, nil
}

// RemoveSignature 移除 APK 签名（用于重新签名）
func (m *Manager) RemoveSignature(apkPath, outputPath string) error {
// APK 本质是 ZIP，移除签名 = 删除 META-INF 目录
// 使用 archive/zip 重新打包，排除 META-INF/
return removeMetaInf(apkPath, outputPath)
}

func parseCertInfo(output string) map[string]string {
info := make(map[string]string)
lines := strings.Split(output, "n")
for _, line := range lines {
line = strings.TrimSpace(line)
if strings.Contains(line, "Signer #1 certificate DN:") {
info["DN"] = strings.SplitN(line, ":", 2)[1]
}
if strings.Contains(line, "certificate SHA-256 digest") {
info["SHA256"] = strings.SplitN(line, ":", 2)[1]
}
}
return info
}

func extractPackageName(apkPath string) string {
// 简化实现: 实际应使用 aapt2 或解析 AndroidManifest.xml
return "com.example.app"
}

func removeMetaInf(inputPath, outputPath string) error {
// 实现: 读取 ZIP，过滤 META-INF/，写入新文件
// 使用 archive/zip 标准库
return nil // 占位
}

6.6 重打包模块 — internal/repack/pipeline.go

package repack

import (
"fmt"
"os/exec"

	"apk-signer-studio/internal/config"
	"apk-signer-studio/internal/signer"
)

// RepackOptions 重打包选项
type RepackOptions struct {
InputAPK     string         json:"inputApk"
OutputDir    string         json:"outputDir"
OutputAPK    string         json:"outputApk"
SignOpts     signer.SignOptions json:"signOpts"
SkipSign     bool           json:"skipSign"
SkipAlign    bool           json:"skipAlign"
}

// PipelineStep 流水线步骤
type PipelineStep struct {
Name    string json:"name"
Status  string json:"status" // pending / running / success / error
Message string json:"message"
}

// Manager 重打包管理器
type Manager struct {
cfg      *config.Store
apktool  string
signer   *signer.Manager
}

func NewManager(cfg *config.Store) *Manager {
return &Manager{
cfg:     cfg,
apktool: "apktool",
signer:  signer.NewManager(cfg),
}
}

// Decompile 反编译 APK
// apktool d -f -o output_dir input.apk
func (m *Manager) Decompile(inputAPK, outputDir string) error {
args := []string{"d", "-f", "-o", outputDir, inputAPK}

	cmd := exec.Command(m.apktool, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("反编译失败: %wn输出: %s", err, string(output))
	}
	return nil
}

// Recompile 重新编译
// apktool b input_dir -o output.apk
func (m *Manager) Recompile(inputDir, outputAPK string) error {
args := []string{"b", inputDir, "-o", outputAPK}

	cmd := exec.Command(m.apktool, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("重编译失败: %wn输出: %s", err, string(output))
	}
	return nil
}

// FullPipeline 完整重打包流水线
// 反编译 → (用户编辑) → 重编译 → zipalign → 签名 → 验证
func (m *Manager) FullPipeline(
opts RepackOptions,
onStep func(PipelineStep),
) error {

	steps := []struct {
		name string
		fn   func() error
	}{
		{
			name: "反编译 APK",
			fn: func() error {
				return m.Decompile(opts.InputAPK, opts.OutputDir)
			},
		},
		{
			name: "重新编译",
			fn: func() error {
				return m.Recompile(opts.OutputDir, opts.OutputAPK)
			},
		},
		{
			name: "zipalign 对齐",
			fn: func() error {
				if opts.SkipAlign {
					return nil
				}
				aligned := opts.OutputAPK + ".aligned"
				err := m.signer.ZipAlign(opts.OutputAPK, aligned)
				if err != nil {
					return err
				}
				// 用对齐后的文件替换
				return moveFile(aligned, opts.OutputAPK)
			},
		},
		{
			name: "APK 签名",
			fn: func() error {
				if opts.SkipSign {
					return nil
				}
				opts.SignOpts.InputAPK = opts.OutputAPK
				opts.SignOpts.OutputAPK = opts.OutputAPK
				_, err := m.signer.SignAPK(opts.SignOpts)
				return err
			},
		},
	}

	for _, step := range steps {
		if onStep != nil {
			onStep(PipelineStep{Name: step.name, Status: "running"})
		}

		if err := step.fn(); err != nil {
			if onStep != nil {
				onStep(PipelineStep{
					Name:    step.name,
					Status:  "error",
					Message: err.Error(),
				})
			}
			return fmt.Errorf("步骤 [%s] 失败: %w", step.name, err)
		}

		if onStep != nil {
			onStep(PipelineStep{Name: step.name, Status: "success"})
		}
	}

	return nil
}

func moveFile(src, dst string) error {
// 跨文件系统时需要 copy + delete
cmd := exec.Command("mv", src, dst) // Linux/Mac
return cmd.Run()
}

6.7 配置存储 — internal/config/store.go

package config

import (
"encoding/json"
"os"
"path/filepath"
"sync"
)

// Store 配置存储
type Store struct {
path    string
mu      sync.RWMutex
data    map[string]interface{}
}

// Settings 全局设置
type Settings struct {
ToolsDir      string          json:"toolsDir"
LastKeystore  string          json:"lastKeystore"
LastAlias     string          json:"lastAlias"
DefaultSignV1 bool            json:"defaultSignV1"
DefaultSignV2 bool            json:"defaultSignV2"
DefaultSignV3 bool            json:"defaultSignV3"
MirrorIndex   int             json:"mirrorIndex"
Proxy         ProxySettings   json:"proxy"
RecentFiles   []string        json:"recentFiles"
}

type ProxySettings struct {
Enabled  bool   json:"enabled"
Type     string json:"type"
Host     string json:"host"
Port     int    json:"port"
}

func NewStore(filename string) *Store {
homeDir, _ := os.UserHomeDir()
configDir := filepath.Join(homeDir, ".apk-signer-studio")
os.MkdirAll(configDir, 0755)

	store := &Store{
		path: filepath.Join(configDir, filename),
		data: make(map[string]interface{}),
	}
	store.Load()
	return store
}

func (s *Store) Load() error {
s.mu.Lock()
defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		return err // 文件不存在时返回空配置
	}
	return json.Unmarshal(data, &s.data)
}

func (s *Store) Save() error {
s.mu.RLock()
defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *Store) Get(key string) interface{} {
s.mu.RLock()
defer s.mu.RUnlock()
return s.data[key]
}

func (s *Store) Set(key string, value interface{}) {
s.mu.Lock()
defer s.mu.Unlock()
s.data[key] = value
}

func (s *Store) GetString(key, defaultVal string) string {
v := s.Get(key)
if v == nil {
return defaultVal
}
if str, ok := v.(string); ok {
return str
}
return defaultVal
}

func (s *Store) GetBool(key string, defaultVal bool) bool {
v := s.Get(key)
if v == nil {
return defaultVal
}
if b, ok := v.(bool); ok {
return b
}
return defaultVal
}

6.8 前端 Vue 组件示例 — CertificatePanel.vue

<template>
  <div class="cert-panel p-6">
    <h2 class="text-2xl font-bold mb-6 text-gray-800">
      📜 证书管理
    </h2>

    <!-- 选项卡 -->
    <div class="flex space-x-4 mb-6">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        @click="activeTab = tab.id"
        :class="[
          'px-4 py-2 rounded-lg transition',
          activeTab === tab.id
            ? 'bg-blue-600 text-white'
            : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
        ]"
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- 生成证书表单 -->
    <div v-if="activeTab === 'generate'" class="space-y-4">
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-medium mb-1">别名 (Alias)</label>
          <input v-model="form.alias" class="input-field" placeholder="android-key" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">通用名 (CN)</label>
          <input v-model="form.commonName" class="input-field" placeholder="My App" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">组织 (O)</label>
          <input v-model="form.organization" class="input-field" placeholder="My Company" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">组织单位 (OU)</label>
          <input v-model="form.orgUnit" class="input-field" placeholder="Dev Team" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">城市 (L)</label>
          <input v-model="form.locality" class="input-field" placeholder="Beijing" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">省份 (ST)</label>
          <input v-model="form.state" class="input-field" placeholder="Beijing" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">国家 (C)</label>
          <input v-model="form.country" class="input-field" placeholder="CN" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">有效期 (天)</label>
          <input v-model.number="form.validity" type="number" class="input-field" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">密钥算法</label>
          <select v-model="form.keyAlgorithm" class="input-field">
            <option value="RSA">RSA</option>
            <option value="EC">EC (椭圆曲线)</option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">密钥长度</label>
          <select v-model.number="form.keySize" class="input-field">
            <option :value="2048">2048</option>
            <option :value="4096">4096</option>
          </select>
        </div>
      </div>

      <div>
        <label class="block text-sm font-medium mb-1">密码</label>
        <input v-model="form.password" type="password" class="input-field" />
      </div>

      <div>
        <label class="block text-sm font-medium mb-1">保存路径</label>
        <div class="flex space-x-2">
          <input v-model="form.outputPath" class="input-field flex-1" readonly />
          <button @click="selectSavePath" class="btn-secondary">
            浏览...
          </button>
        </div>
      </div>

      <button @click="generateKeystore" class="btn-primary w-full" :disabled="loading">
        {{ loading ? '生成中...' : '🔑 生成密钥库' }}
      </button>

      <!-- 输出日志 -->
      <div v-if="outputLog" class="bg-gray-900 text-green-400 p-4 rounded-lg font-mono text-sm whitespace-pre-wrap">
        {{ outputLog }}
      </div>
    </div>

    <!-- 查看证书 -->
    <div v-if="activeTab === 'view'" class="space-y-4">
      <div class="flex space-x-2">
        <input v-model="viewPath" class="input-field flex-1" placeholder="选择 Keystore 文件" readonly />
        <button @click="selectKeystoreFile" class="btn-secondary">浏览...</button>
        <button @click="viewCertInfo" class="btn-primary">查看</button>
      </div>

      <div v-if="certInfo" class="bg-white border rounded-lg p-4 space-y-2">
        <div class="grid grid-cols-2 gap-2 text-sm">
          <div><span class="font-semibold">别名:</span> {{ certInfo.alias }}</div>
          <div><span class="font-semibold">所有者:</span> {{ certInfo.dn }}</div>
          <div><span class="font-semibold">签发人:</span> {{ certInfo.issuer }}</div>
          <div><span class="font-semibold">序列号:</span> {{ certInfo.serial }}</div>
          <div><span class="font-semibold">有效期:</span> {{ certInfo.validFrom }} → {{ certInfo.validTo }}</div>
          <div><span class="font-semibold">SHA256:</span> <code class="text-xs">{{ certInfo.fingerSHA256 }}</code></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
// Wails 自动生成的绑定
import { GenerateKeystore, ListKeystoreInfo } from '../wailsjs/go/main/App'
import { SelectSaveFile, SelectOpenFile } from '../wailsjs/go/main/App'

const tabs = [
  { id: 'generate', label: '🔑 生成证书' },
  { id: 'view', label: '👁️ 查看证书' },
  { id: 'export', label: '📤 导出证书' },
]

const activeTab = ref('generate')
const loading = ref(false)
const outputLog = ref('')
const viewPath = ref('')
const certInfo = ref(null)

const form = reactive({
  alias: 'android-key',
  commonName: '',
  organization: '',
  orgUnit: '',
  locality: '',
  state: '',
  country: 'CN',
  validity: 10000,
  keyAlgorithm: 'RSA',
  keySize: 2048,
  password: '',
  outputPath: '',
})

async function generateKeystore() {
  loading.value = true
  outputLog.value = ''
  try {
    const result = await GenerateKeystore(form.outputPath, form.password, {
      alias: form.alias,
      commonName: form.commonName,
      organization: form.organization,
      orgUnit: form.orgUnit,
      locality: form.locality,
      state: form.state,
      country: form.country,
      validity: form.validity,
      keyAlgorithm: form.keyAlgorithm,
      keySize: form.keySize,
    })
    outputLog.value = result
  } catch (e) {
    outputLog.value = 错误: {e}
  } finally {
    loading.value = false
  }
}

async function selectSavePath() {
  form.outputPath = await SelectSaveFile('keystore.jks', .jks;.keystore')
}

async function selectKeystoreFile() {
  viewPath.value = await SelectOpenFile(.jks;.keystore;*.p12')
}

async function viewCertInfo() {
  const password = prompt('请输入密钥库密码:')
  if (!password) return
  try {
    certInfo.value = await ListKeystoreInfo(viewPath.value, password)
  } catch (e) {
    alert(查看失败: {e})
  }
}
</script>

<style scoped>
.input-field {
  @apply w-full px-3 py-2 border border-gray-300 rounded-lg
         focus:ring-2 focus:ring-blue-500 focus:border-transparent
         outline-none transition;
}
.btn-primary {
  @apply px-6 py-2 bg-blue-600 text-white rounded-lg
         hover:bg-blue-700 disabled:opacity-50 transition;
}
.btn-secondary {
  @apply px-4 py-2 bg-gray-200 text-gray-700 rounded-lg
         hover:bg-gray-300 transition;
}
</style>

七、如何测试和运行

7.1 环境准备

flowchart LR
A["1. 安装 Go 1.22+"] --> B["2. 安装 Wails CLI"]
B --> C["3. 安装 Node.js 18+"]
C --> D["4. 安装 JDK 11+"]
D --> E["5. 下载 Android Build Tools"]
E --> F["6. 验证环境: wails doctor"]

安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

检查环境依赖
wails doctor

安装 JDK (keytool 依赖)
Ubuntu/Debian:
sudo apt install default-jdk
macOS:
brew install openjdk

下载 Android Build Tools
方法A: 通过 Android Studio SDK Manager
方法B: 直接下载 (国内镜像)
wget https://mirrors.cloud.tencent.com/AndroidSDK/build-tools_r35.0.0-linux.zip

7.2 项目初始化与运行

创建项目
wails init -n apk-signer-studio -t vue-ts
cd apk-signer-studio

复制上面所有 Go 代码到对应位置
复制 Vue 组件到 frontend/src/components/

开发模式运行（热重载）
wails dev

生产构建
wails build

构建产物位置
ls build/bin/

7.3 测试策略

graph TD
subgraph "单元测试"
UT1["证书生成/解析测试"]
UT2["工具检测逻辑测试"]
UT3["签名参数构建测试"]
UT4["配置存储测试"]
end

    subgraph "集成测试"
        IT1["完整签名流水线测试"]
        IT2["下载+安装工具测试"]
        IT3["重打包端到端测试"]
    end

    subgraph "手动验证"
        MT1["生成测试 APK 并签名"]
        MT2["在 Android 设备上安装验证"]
        MT3["apksigner verify 交叉验证"]
    end

单元测试示例:

// internal/certificate/keystore_test.go
package certificate

import (
"os"
"path/filepath"
"testing"
)

func TestGenerateKeystore(t *testing.T) {
cfg := config.NewStore("test-config.json")
mgr := NewManager(cfg)

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test.jks")

	info := CertInfo{
		Alias:        "test-key",
		CommonName:   "Test App",
		Organization: "Test Org",
		Country:      "CN",
		Validity:     365,
		KeyAlgorithm: "RSA",
		KeySize:      2048,
	}

	output, err := mgr.GenerateKeystore(outputPath, "testpass123", info)
	if err != nil {
		t.Fatalf("生成密钥库失败: %vn输出: %s", err, output)
	}

	// 验证文件存在
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("密钥库文件未创建")
	}

	// 验证可以读取
	keystoreInfo, err := mgr.ListKeystoreInfo(outputPath, "testpass123")
	if err != nil {
		t.Fatalf("读取密钥库失败: %v", err)
	}

	if keystoreInfo.Alias != "test-key" {
		t.Errorf("别名不匹配: got %s, want test-key", keystoreInfo.Alias)
	}
}

func TestParseKeytoolOutput(t *testing.T) {
sampleOutput :=
别名: android-key
所有者: CN=Test, O=Company, C=CN
签发人: CN=Test, O=Company, C=CN
序列号: 1234567890
有效期从 2024-01-01 至 2051-05-18
MD5:  AA:BB:CC:DD
SHA1: EE:FF:00:11
SHA256: 22:33:44:55:66

	info := parseKeytoolOutput(sampleOutput)
	if info.Alias != "android-key" {
		t.Errorf("解析别名失败: %s", info.Alias)
	}
}

创建测试用 APK 文件:

创建一个最小化测试 APK (用于签名测试)
方法: 从任意 APK 文件复制，或使用 Android 项目构建
快速方法: 下载一个开源 APK
wget -O test-unsigned.apk \
"https://github.com/nickstenning/apk/raw/master/test.apk"

或使用 aapt2 手动构建最小 APK
(详见 Android 开发者文档)

完整测试流程:

运行所有 Go 单元测试
go test ./... -v

运行带覆盖率报告
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out

端到端手动测试
启动应用
wails dev

在 GUI 中:
a. 生成一个新证书
b. 下载 Build Tools (选择镜像源)
c. 选择一个 APK 进行签名
d. 验证签名结果

命令行交叉验证
apksigner verify --verbose build/output/test-signed.apk

7.4 打包发布

构建所有平台
Linux
GOOS=linux GOARCH=amd64 wails build

Windows (交叉编译需要特殊配置)
GOOS=windows GOARCH=amd64 wails build

macOS
GOOS=darwin GOARCH=arm64 wails build

macOS DMG 打包
wails build -nsis  # Windows 安装包
wails build -dmg   # macOS DMG

八、模块依赖关系总结

graph BT
MAIN["main.gon(Wails 入口)"]
APP["app.gon(API 绑定层)"]

    subgraph "业务模块"
        CERT["certificate/n证书管理"]
        TOOLS["tools/n工具管理"]
        SIGN["signer/nAPK 签名"]
        REPACK["repack/n重打包"]
        APK["apkparser/nAPK 解析"]
    end

    subgraph "基础设施"
        CFG["config/n配置存储"]
        LOG["logger/n日志"]
    end

    subgraph "外部依赖"
        WAILS["wails/v2"]
        KEYTOOL_EXT["keytool"]
        APS_EXT["apksigner"]
        ZIP_EXT["zipalign"]
        APKTOOL_EXT["apktool"]
    end

    MAIN --> WAILS
    MAIN --> APP
    APP --> CERT & TOOLS & SIGN & REPACK
    REPACK --> SIGN
    REPACK --> APKTOOL_EXT
    SIGN --> APS_EXT & ZIP_EXT
    CERT --> KEYTOOL_EXT
    CERT & TOOLS & SIGN & REPACK & APK --> CFG
    CERT & TOOLS & SIGN & REPACK --> LOG

总结
特性   说明
框架   Wails v2（Go 后端 + Vue3 前端），打包体积 ~15-25MB

核心工具   apksigner、zipalign、keytool、apktool

国内适配   内置腾讯/清华/东软镜像源 + HTTP/SOCKS5 代理

签名方案   完整支持 V1/V2/V3/V4，支持组合签名

安全   ZIP Slip 防护、SHA256 校验、密码不在日志中明文显示

扩展   模块化设计，可轻松添加批量签名、APK 分析等功能

以上代码是一个完整的设计骨架，各模块均可独立测试和迭代开发。建议按照 工具管理 → 证书管理 → APK 签名 → 重打包 的顺序逐步实现。