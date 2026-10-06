package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"go-apksigner-gui/internal/config/appcfg"
	"go-apksigner-gui/internal/dto"
	"go-apksigner-gui/internal/executor"
	"go-apksigner-gui/internal/fileutil"
)

// Manager 工具链门面：探测、版本索引、下载安装、临时目录清理。
type Manager struct {
	mu        sync.RWMutex
	toolsDir  string
	mirror    dto.MirrorSource
	proxy     dto.ProxyConfig
	override  map[string]string
	client    *http.Client
	cancels   map[string]context.CancelFunc
	cacheTool map[string]Tool
	cfg       *appcfg.Config
	sdkPath   string
}

// NewManager 创建工具链管理器。cfg 为外部化配置（可为 nil，内部回退默认值）。
func NewManager(toolsDir string, mirror dto.MirrorSource, proxy dto.ProxyConfig, cfg *appcfg.Config) *Manager {
	if toolsDir == "" {
		toolsDir = filepath.Join(os.TempDir(), "go-apksigner-gui", "tools")
	}
	if cfg == nil {
		cfg = appcfg.DefaultConfig()
	}
	m := &Manager{
		toolsDir: toolsDir,
		mirror:   mirror,
		proxy:    proxy,
		override: map[string]string{},
		cancels:  map[string]context.CancelFunc{},
		cfg:      cfg,
	}
	if mirror.BaseURL == "" {
		m.mirror = DefaultMirror()
	}
	m.client, _ = NewHTTPClient(m.proxy, 30*time.Minute)
	_ = os.MkdirAll(m.toolsDir, 0o755)
	return m
}

// SetSdkPath 设置用户配置的 Android SDK 根目录（手动安装场景）。
func (m *Manager) SetSdkPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sdkPath = path
}

// SetMirror 切换镜像源。
func (m *Manager) SetMirror(mirror dto.MirrorSource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mirror = mirror
}

// Mirror 返回当前镜像源。
func (m *Manager) Mirror() dto.MirrorSource {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mirror
}

// SetProxy 更新代理设置并重建 HTTP 客户端。
func (m *Manager) SetProxy(proxy dto.ProxyConfig) error {
	client, err := NewHTTPClient(proxy, 30*time.Minute)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.proxy = proxy
	m.client = client
	return nil
}

// Proxy 返回当前代理设置。
func (m *Manager) Proxy() dto.ProxyConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.proxy
}

// SetOverride 记录用户手工指定的工具路径，空字符串表示清除。
func (m *Manager) SetOverride(name, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.override == nil {
		m.override = map[string]string{}
	}
	if path == "" {
		delete(m.override, name)
	} else {
		m.override[name] = path
	}
	m.cacheTool = nil
}

// ToolsDir 返回工具目录。
func (m *Manager) ToolsDir() string { return m.toolsDir }

// DetectAll 返回全部工具的状态。
func (m *Manager) DetectAll() []dto.ToolStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	detected := Detect(DetectOptions{ToolsDir: m.toolsDir, Override: m.override, SdkPath: m.sdkPath})
	m.cacheTool = detected

	statuses := make([]dto.ToolStatus, 0, len(specs))
	for _, s := range specs {
		t := detected[s.name]
		statuses = append(statuses, dto.ToolStatus{
			Name:      s.name,
			Required:  s.required,
			Installed: t.Available(),
			Path:      t.Path,
			Version:   t.Version,
			Source:    sourceOf(t),
		})
	}
	return statuses
}

// Tool 返回指定工具的调用方式（缓存优先）。
func (m *Manager) Tool(name string) (Tool, error) {
	m.mu.RLock()
	cached, ok := m.cacheTool[name]
	m.mu.RUnlock()
	if ok {
		return cached, nil
	}
	result := m.DetectAll()
	for _, st := range result {
		if st.Name == name {
			m.mu.RLock()
			t := m.cacheTool[name]
			m.mu.RUnlock()
			if t.Available() {
				return t, nil
			}
			return Tool{}, fmt.Errorf("工具 %s 不可用，请先在工具链页面安装 build-tools", name)
		}
	}
	return Tool{}, fmt.Errorf("未知工具: %s", name)
}

// MustTool 返回工具，不可用时返回错误（供流水线使用）。
func (m *Manager) MustTool(name string) (Tool, error) {
	t, err := m.Tool(name)
	if err != nil {
		return Tool{}, err
	}
	if !t.Available() {
		return Tool{}, fmt.Errorf("工具 %s 不可用，请先在工具链页面安装 build-tools 或手工指定路径", name)
	}
	return t, nil
}

// sourceOf 计算工具来源标记。
func sourceOf(t Tool) string {
	if !t.Available() {
		return "missing"
	}
	return t.Source
}

// FetchVersions 获取可安装的 build-tools 版本清单：优先仓库索引，失败时回退配置/内置清单。
func (m *Manager) FetchVersions(ctx context.Context) ([]dto.BuildToolVersion, error) {
	m.mu.RLock()
	baseURL := m.mirror.BaseURL
	client := m.client
	m.mu.RUnlock()

	if urls := IndexURL(baseURL); urls != "" {
		data, err := fetchBytes(ctx, client, urls, 30*time.Second)
		if err == nil {
			if versions, perr := ParseRepository(data, baseURL, HostOS()); perr == nil {
				return versions, nil
			}
		}
	}
	// 仓库索引不可达时，优先使用 app.yaml 中配置的候选版本，最后回退内置清单。
	if len(m.cfg.BuildTools.Versions) > 0 {
		versions := make([]dto.BuildToolVersion, 0, len(m.cfg.BuildTools.Versions))
		for _, v := range m.cfg.BuildTools.Versions {
			name := fmt.Sprintf("build-tools_r%s-%s.zip", v, HostOS())
			versions = append(versions, dto.BuildToolVersion{
				Version:  v,
				FileName: name,
				URL:      resolveURL(baseURL, name),
			})
		}
		return versions, nil
	}
	versions := StaticVersions(HostOS(), baseURL)
	return versions, nil
}

// fetchBytes 简单 GET 拉取小文本。
func fetchBytes(ctx context.Context, client *http.Client, url string, timeout time.Duration) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}

// sdkRoot 返回 SDK 根目录：优先用户配置路径，否则使用托管目录 toolsDir/android-sdk。
func (m *Manager) sdkRoot() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.sdkPath != "" {
		return m.sdkPath
	}
	return filepath.Join(m.toolsDir, "android-sdk")
}

// sdkManagerPath 返回 sdkmanager 可执行文件路径（按系统区分扩展名）。
func sdkManagerPath(root string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(root, "cmdline-tools", "latest", "bin", "sdkmanager.bat")
	}
	return filepath.Join(root, "cmdline-tools", "latest", "bin", "sdkmanager")
}

// InstallViaSdkManager 通过 sdkmanager 安装指定版本的 build-tools。
// 若 SDK 根目录下尚未就绪 sdkmanager，会自动下载 command-line tools 并按 latest 结构解压。
func (m *Manager) InstallViaSdkManager(ctx context.Context, version string, on ProgressFunc) (dto.TaskResult, error) {
	if version == "" {
		return failedResult("未指定 build-tools 版本"), nil
	}
	root := m.sdkRoot()
	if err := os.MkdirAll(root, 0o755); err != nil {
		return failedResult(err.Error()), nil
	}

	ctx, cancel := context.WithCancel(ctx)
	m.registerCancel("install:"+version, cancel)
	defer func() {
		cancel()
		m.clearCancel("install:" + version)
	}()

	emitProgress(on, "build-tools;"+version, 0, 0, "downloading", "准备安装环境")
	sdkMgr, err := m.ensureSdkManager(ctx, root, on)
	if err != nil {
		emit(on, dto.Progress{Stage: "error", Message: err.Error()})
		return failedResult(err.Error()), nil
	}

	if !javaAvailable() {
		msg := "未检测到 Java/JDK，sdkmanager 无法运行。请先安装 JDK 并配置 JAVA_HOME 或将其加入 PATH"
		emit(on, dto.Progress{Stage: "error", Message: msg})
		return failedResult(msg), nil
	}

	emitProgress(on, "build-tools;"+version, 0, 0, "downloading", "正在通过 sdkmanager 安装 "+version)
	cmd := exec.CommandContext(ctx, sdkMgr, "--sdk_root", root, "build-tools;"+version)
	executor.HideWindow(cmd)
	cmd.Env = m.sdkManagerEnv(root)
	cmd.Stdin = strings.NewReader("y\ny\ny\n")
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	for _, line := range strings.Split(outBuf.String()+"\n"+errBuf.String(), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			emit(on, dto.Progress{FileName: "build-tools;" + version, Stage: "downloading", Message: line})
		}
	}
	if runErr != nil {
		msg := fmt.Sprintf("sdkmanager 安装失败: %v", runErr)
		emit(on, dto.Progress{Stage: "error", Message: msg})
		return failedResult(msg), nil
	}

	m.mu.Lock()
	m.cacheTool = nil
	m.mu.Unlock()
	statuses := m.DetectAll()
	emit(on, dto.Progress{FileName: "build-tools;" + version, Stage: "done", Percent: 100, Message: "安装完成"})
	detail := fmt.Sprintf("SDK 根目录: %s\n", root)
	for _, s := range statuses {
		detail += fmt.Sprintf("%s: installed=%v path=%s version=%s\n", s.Name, s.Installed, s.Path, s.Version)
	}
	return dto.TaskResult{Success: true, Message: fmt.Sprintf("build-tools %s 安装完成", version), Detail: detail}, nil
}

// sdkManagerEnv 构造 sdkmanager 运行环境变量，注入 SDK 根与代理（来自设置）。
func (m *Manager) sdkManagerEnv(root string) []string {
	env := os.Environ()
	env = append(env, "ANDROID_SDK_ROOT="+root, "ANDROID_HOME="+root)
	proxy := m.Proxy()
	if proxy.Enabled && proxy.Host != "" {
		scheme := proxy.Type
		if scheme == "" {
			scheme = "http"
		}
		host := fmt.Sprintf("%s://%s:%d", scheme, proxy.Host, proxy.Port)
		if proxy.Username != "" {
			host = fmt.Sprintf("%s://%s:%s@%s:%d", scheme, proxy.Username, proxy.Password, proxy.Host, proxy.Port)
		}
		env = append(env, "HTTP_PROXY="+host, "HTTPS_PROXY="+host)
	}
	return env
}

// ensureSdkManager 确保 sdkmanager 存在：已存在则直接返回，否则下载 cmdline-tools 并按 latest 结构解压。
func (m *Manager) ensureSdkManager(ctx context.Context, root string, on ProgressFunc) (string, error) {
	if p := sdkManagerPath(root); fileExists(p) {
		return p, nil
	}
	url := m.cfg.SdkManager.URLForOS(HostOS())
	if url == "" {
		return "", fmt.Errorf("未配置 sdkmanager 下载地址（请检查 app.yaml 的 sdkmanager.%sUrl）", HostOS())
	}
	downloadDir := filepath.Join(m.toolsDir, "downloads")
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		return "", err
	}
	fileName := filepath.Base(url)
	v := dto.BuildToolVersion{
		Version:  "cmdline-tools",
		FileName: fileName,
		URL:      url,
		SHA256:   m.cfg.SdkManager.SHA256ForOS(HostOS()),
	}
	emitProgress(on, fileName, 0, v.Size, "downloading", "下载 command-line tools")
	zipPath, err := Download(ctx, m.client, v, downloadDir, on)
	if err != nil {
		return "", fmt.Errorf("下载 command-line tools 失败: %w", err)
	}
	extractDir := filepath.Join(m.toolsDir, "tmp", "cmdline-tools-extract")
	_ = os.RemoveAll(extractDir)
	emitProgress(on, fileName, v.Size, v.Size, "extracting", "解压 command-line tools")
	if _, err := fileutil.Extract(zipPath, extractDir, fileutil.ExtractOptions{}); err != nil {
		return "", fmt.Errorf("解压失败: %w", err)
	}
	// Google 压缩包顶层为 cmdline-tools/{bin,lib,...}，需移动到 cmdline-tools/latest 结构
	src := filepath.Join(extractDir, "cmdline-tools")
	if _, err := os.Stat(src); err != nil {
		src = extractDir // 兼容个别平铺结构
	}
	dst := filepath.Join(root, "cmdline-tools", "latest")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	_ = os.RemoveAll(dst)
	if err := os.Rename(src, dst); err != nil {
		return "", fmt.Errorf("移动 sdkmanager 目录失败: %w", err)
	}
	_ = os.RemoveAll(extractDir)
	makeExecutable(dst)
	if p := sdkManagerPath(root); fileExists(p) {
		return p, nil
	}
	return "", fmt.Errorf("sdkmanager 解压后未找到可执行文件: %s", sdkManagerPath(root))
}

// javaAvailable 检测系统是否具备运行 sdkmanager 所需的 Java。
func javaAvailable() bool {
	if _, err := exec.LookPath("java"); err == nil {
		return true
	}
	if jh := os.Getenv("JAVA_HOME"); jh != "" {
		exe := "java"
		if runtime.GOOS == "windows" {
			exe = "java.exe"
		}
		if fileExists(filepath.Join(jh, "bin", exe)) {
			return true
		}
	}
	return false
}

// CancelDownload 取消正在进行的安装（sdkmanager 进程会随 context 取消而终止）。
func (m *Manager) CancelDownload(version string) {
	for _, key := range []string{"install:" + version, "download:" + version} {
		m.mu.Lock()
		cancel, ok := m.cancels[key]
		if ok {
			delete(m.cancels, key)
		}
		m.mu.Unlock()
		if ok {
			cancel()
		}
	}
}

func (m *Manager) registerCancel(key string, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancels == nil {
		m.cancels = map[string]context.CancelFunc{}
	}
	m.cancels[key] = cancel
}

func (m *Manager) clearCancel(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cancels, key)
}

// CleanTemp 清理下载缓存与临时目录。
func (m *Manager) CleanTemp() error {
	var firstErr error
	for _, dir := range []string{filepath.Join(m.toolsDir, "downloads"), filepath.Join(m.toolsDir, "tmp")} {
		if err := os.RemoveAll(dir); err != nil && firstErr == nil && !os.IsNotExist(err) {
			firstErr = err
		}
	}
	return firstErr
}

// makeExecutable 在类 Unix 平台上补齐可执行权限。
func makeExecutable(root string) {
	if _, err := os.Stat(root); err != nil {
		return
	}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		for _, name := range []string{"apksigner", "zipalign", "aapt2", "aapt"} {
			if info.Name() == name {
				_ = os.Chmod(path, 0o755)
			}
		}
		return nil
	})
}

// failedResult 构造失败结果。
func failedResult(msg string) dto.TaskResult {
	return dto.TaskResult{Success: false, Message: msg}
}

// RunTool 以统一方式调用工具链中的可执行文件。
func RunTool(ctx context.Context, t Tool, args []string, timeout time.Duration, onLine func(stream, text string)) *executor.Result {
	bin, fullArgs := t.Command(args)
	return executor.Run(ctx, bin, fullArgs, executor.Options{Timeout: timeout, OnLine: onLine})
}
