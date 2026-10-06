package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	gruntime "runtime"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"go-apksigner-gui/internal/apkinfo"
	"go-apksigner-gui/internal/certificate"
	"go-apksigner-gui/internal/config"
	"go-apksigner-gui/internal/config/appcfg"
	"go-apksigner-gui/internal/dto"
	"go-apksigner-gui/internal/i18n"
	"go-apksigner-gui/internal/logger"
	"go-apksigner-gui/internal/secure"
	"go-apksigner-gui/internal/signer"
	"go-apksigner-gui/internal/tools"
)

// App 是 Wails 绑定层：暴露给前端的全部 API 都挂载在该结构体上。
// 这里只做参数整理、事件推送与结果封装，业务逻辑全部位于 internal 各模块。
type App struct {
	ctx       context.Context
	cfg       *config.Store
	appCfg    *appcfg.Config
	toolMgr   *tools.Manager
	certMgr   *certificate.Manager
	signMgr   *signer.Manager
	apkParser *apkinfo.Parser
}

// appVersion 应用版本号。
const appVersion = "1.0.0"

// NewApp 构造应用，初始化配置与各业务模块。
func NewApp() *App {
	cfg := config.New("")
	settings := cfg.Snapshot()
	appCfg := appcfg.Load(appcfg.DefaultPaths()...)

	proxy := dto.ProxyConfig{
		Enabled:  settings.Proxy.Enabled,
		Type:     settings.Proxy.Type,
		Host:     settings.Proxy.Host,
		Port:     settings.Proxy.Port,
		Username: settings.Proxy.Username,
		Password: settings.Proxy.Password,
	}
	// 将外部化配置中的镜像源注入探测器查找表（避免空列表覆盖导致越界）
	if len(appCfg.Mirrors) > 0 {
		tools.Mirrors = appCfg.Mirrors
	}
	toolMgr := tools.NewManager(settings.ToolsDir, tools.FindMirror(settings.Mirror.BaseURL), proxy, appCfg)
	for name, path := range settings.Tools {
		toolMgr.SetOverride(name, path)
	}
	toolMgr.SetSdkPath(settings.AndroidSDK)
	signMgr := signer.NewManager(toolMgr)

	return &App{
		cfg:       cfg,
		appCfg:    appCfg,
		toolMgr:   toolMgr,
		certMgr:   certificate.NewManager(toolMgr),
		signMgr:   signMgr,
		apkParser: apkinfo.NewParser(toolMgr, signMgr),
	}
}

// startup 生命周期钩子：初始化日志并后台探测工具链。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	_ = logger.Init(config.Dir())
	i18n.SetLocale(a.cfg.Snapshot().Language)
	logger.Info(i18n.T("app.log.startup"), "version", appVersion)

	go func() {
		statuses := a.toolMgr.DetectAll()
		logger.Info(i18n.T("app.log.toolsDetected"))
		runtime.EventsEmit(a.ctx, "tools:changed", statuses)
	}()
}

// shutdown 生命周期钩子：持久化配置。
func (a *App) shutdown(ctx context.Context) {
	logger.Info(i18n.T("app.log.shutdown"))
	if err := a.cfg.Save(); err != nil {
		logger.Error("保存配置失败", "error", err.Error())
	}
}

// emit 推送事件到前端。
func (a *App) emit(name string, data any) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, data)
}

// ---------------------------------------------------------------- 通用信息

// GetAppMeta 返回应用信息与关键路径。
func (a *App) GetAppMeta() dto.AppMeta {
	goVersion := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok && info.GoVersion != "" {
		goVersion = info.GoVersion
	}
	return dto.AppMeta{
		Name:                     "APK Signer Studio",
		Version:                  appVersion,
		GoVersion:                goVersion,
		Platform:                 fmt.Sprintf("%s/%s", gruntime.GOOS, gruntime.GOARCH),
		ConfigPath:               a.cfg.Path(),
		ToolsDir:                 a.toolMgr.ToolsDir(),
		DefaultBuildToolsVersion: a.appCfg.BuildTools.DefaultVersion,
	}
}

// GetSettings 返回全局设置。
func (a *App) GetSettings() dto.AppSettings {
	s := a.cfg.Snapshot()
	mirror := a.toolMgr.Mirror()
	proxy := a.toolMgr.Proxy()
	return dto.AppSettings{
		ToolsDir:     a.toolMgr.ToolsDir(),
		ConfigPath:   a.cfg.Path(),
		LastKeystore: s.LastKeystore,
		LastAlias:    s.LastAlias,
		LastApk:      s.LastApk,
		DefaultV1:    s.DefaultV1,
		DefaultV2:    s.DefaultV2,
		DefaultV3:    s.DefaultV3,
		DefaultV4:    s.DefaultV4,
		RecentFiles:  s.RecentFiles,
		RecentAlias:  s.RecentAlias,
		Mirror:       mirror,
		Proxy:        proxy,
		AndroidSdk:   s.AndroidSDK,
		Language:     s.Language,
		Tools:        s.Tools,
	}
}

// SaveSettings 保存设置（镜像源、代理、默认签名方案、手工指定的工具路径）。
func (a *App) SaveSettings(settings dto.AppSettings) error {
	err := a.cfg.Update(func(cfg *config.Settings) {
		cfg.LastKeystore = settings.LastKeystore
		cfg.LastAlias = settings.LastAlias
		cfg.LastApk = settings.LastApk
		cfg.DefaultV1 = settings.DefaultV1
		cfg.DefaultV2 = settings.DefaultV2
		cfg.DefaultV3 = settings.DefaultV3
		cfg.DefaultV4 = settings.DefaultV4
		cfg.Mirror = config.MirrorSetting{Name: settings.Mirror.Name, BaseURL: settings.Mirror.BaseURL}
		cfg.Proxy = config.ProxySetting{
			Enabled:  settings.Proxy.Enabled,
			Type:     settings.Proxy.Type,
			Host:     settings.Proxy.Host,
			Port:     settings.Proxy.Port,
			Username: settings.Proxy.Username,
			Password: settings.Proxy.Password,
		}
		cfg.AndroidSDK = settings.AndroidSdk
		if settings.Tools != nil {
			cfg.Tools = settings.Tools
		}
		cfg.Language = settings.Language
	})
	if err != nil {
		return fmt.Errorf("保存设置失败: %w", err)
	}
	i18n.SetLocale(settings.Language)
	a.toolMgr.SetMirror(tools.FindMirror(settings.Mirror.BaseURL))
	if perr := a.toolMgr.SetProxy(settings.Proxy); perr != nil {
		return perr
	}
	for name, path := range settings.Tools {
		a.toolMgr.SetOverride(name, path)
	}
	a.toolMgr.SetSdkPath(settings.AndroidSdk)
	logger.Info(i18n.T("app.log.settingsSaved"), "mirror", settings.Mirror.Name, "proxy", settings.Proxy.Enabled, "language", settings.Language)
	a.emit("tools:changed", a.toolMgr.DetectAll())
	return nil
}

// SetLanguage 即时切换后端文案语言并持久化（供前端切语言时调用，无需回传整个设置）。
func (a *App) SetLanguage(language string) error {
	i18n.SetLocale(language)
	return a.cfg.Update(func(cfg *config.Settings) {
		cfg.Language = i18n.Locale()
	})
}

// TestProxy 测试代理连通性：通过配置的代理访问 Google 主页，不影响已保存配置。
// 业务逻辑委托给 tools.TestProxyConnection，符合绑定层只做参数整理与结果封装的约定。
func (a *App) TestProxy(proxy dto.ProxyConfig) dto.TaskResult {
	result := tools.TestProxyConnection(proxy, a.appCfg.ProxyTest.Target, time.Duration(a.appCfg.ProxyTest.TimeoutSeconds)*time.Second)
	logger.Info("代理连通性测试", "enabled", proxy.Enabled, "type", proxy.Type,
		"host", proxy.Host, "port", proxy.Port, "success", result.Success)
	return result
}

// RecentLogs 返回最近的日志条目（用于日志抽屉初始化）。
func (a *App) RecentLogs(n int) []logger.Entry {
	return logger.Recent(n)
}

// ---------------------------------------------------------------- 文件选择

// apkFilter APK 文件对话框过滤器。
const apkFilter = "APK 文件 (*.apk):*.apk"

// SelectOpenFile 打开文件（默认 APK）。
func (a *App) SelectOpenFile(filter string) (string, error) {
	if filter == "" {
		filter = apkFilter
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "选择文件",
		Filters: []runtime.FileFilter{{DisplayName: "筛选的文件", Pattern: filter}},
	})
}

// SelectSaveFile 选择保存路径。
func (a *App) SelectSaveFile(defaultName, filter string) (string, error) {
	if filter == "" {
		filter = apkFilter
	}
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                "保存到",
		DefaultFilename:      defaultName,
		DefaultDirectory:     "",
		CanCreateDirectories: true,
		Filters:              []runtime.FileFilter{{DisplayName: "筛选的文件", Pattern: filter}},
	})
}

// SelectDirectory 选择目录。
func (a *App) SelectDirectory(title string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
}

// ---------------------------------------------------------------- 工具链

// GetTools 返回工具检测结果。
func (a *App) GetTools() []dto.ToolStatus {
	return a.toolMgr.DetectAll()
}

// GetMirrors 返回内置镜像源。
func (a *App) GetMirrors() []dto.MirrorSource {
	return tools.Mirrors
}

// FetchBuildToolVersions 获取可安装的 build-tools 版本列表。
func (a *App) FetchBuildToolVersions() ([]dto.BuildToolVersion, error) {
	versions, err := a.toolMgr.FetchVersions(context.Background())
	if err != nil {
		return []dto.BuildToolVersion{}, err
	}
	return versions, nil
}

// InstallBuildTools 通过 sdkmanager 安装指定版本的 build-tools，进度通过 download:progress 事件推送。
func (a *App) InstallBuildTools(version string) dto.TaskResult {
	result, err := a.toolMgr.InstallViaSdkManager(context.Background(), version, func(p dto.Progress) {
		a.emit("download:progress", p)
	})
	if err != nil {
		logger.Error(i18n.T("tools.log.installFail"), "version", version, "error", err.Error())
		return dto.TaskResult{Success: false, Message: err.Error()}
	}
	if result.Success {
		logger.Info(i18n.T("tools.log.installDone"), "version", version)
	} else {
		logger.Error(i18n.T("tools.log.installIncomplete"), "version", version, "message", result.Message)
	}
	return result
}

// CancelInstall 取消正在进行的下载。
func (a *App) CancelInstall(version string) {
	a.toolMgr.CancelDownload(version)
	logger.Warn(i18n.T("tools.log.cancelDownload"), "version", version)
}

// CleanTemp 清理下载缓存与临时文件。
func (a *App) CleanTemp() dto.TaskResult {
	if err := a.toolMgr.CleanTemp(); err != nil {
		return dto.TaskResult{Success: false, Message: err.Error()}
	}
	return dto.TaskResult{Success: true, Message: i18n.T("app.msg.cleanTempDone")}
}

// ---------------------------------------------------------------- 证书管理

// GenerateKeystore 生成密钥库。
func (a *App) GenerateKeystore(req dto.KeystoreRequest) dto.TaskResult {
	out, err := a.certMgr.Generate(context.Background(), req)
	if err != nil {
		logger.Error("生成密钥库失败", "error", logger.Mask(err.Error(), req.StorePass, req.KeyPass))
		return dto.TaskResult{Success: false, Message: err.Error(), Detail: out}
	}
	if req.StorePass != "" {
		a.rememberKeystore(req.OutputPath, req.Cert.Alias)
	}
	logger.Info(i18n.T("cert.msg.genDone", req.OutputPath), "path", req.OutputPath)
	return dto.TaskResult{Success: true, Message: i18n.T("cert.msg.genDone", req.OutputPath), Detail: out}
}

// ListKeystore 查看密钥库详情。
func (a *App) ListKeystore(path, storePass string) (dto.KeystoreInfo, error) {
	info, err := a.certMgr.List(context.Background(), path, storePass)
	if err != nil {
		return dto.KeystoreInfo{Path: path}, err
	}
	a.rememberKeystore(path, "")
	return *info, nil
}

// ListAliases 列出密钥库中的别名。
func (a *App) ListAliases(path, storePass string) ([]string, error) {
	aliases, err := a.certMgr.Aliases(context.Background(), path, storePass)
	if err != nil {
		return []string{}, err
	}
	a.rememberKeystore(path, "")
	return aliases, nil
}

// ExportCertificate 导出证书。
func (a *App) ExportCertificate(keystorePath, storePass, alias, outputPath string) dto.TaskResult {
	out, err := a.certMgr.Export(context.Background(), keystorePath, storePass, alias, outputPath)
	if err != nil {
		return dto.TaskResult{Success: false, Message: err.Error(), Detail: out}
	}
	return dto.TaskResult{Success: true, Message: i18n.T("cert.msg.exportDone", outputPath), Detail: out}
}

// ConvertKeystore 密钥库格式转换。
func (a *App) ConvertKeystore(srcPath, srcPass, dstPath, dstPass, dstType string) dto.TaskResult {
	out, err := a.certMgr.Convert(context.Background(), srcPath, srcPass, dstPath, dstPass, dstType)
	if err != nil {
		return dto.TaskResult{Success: false, Message: err.Error(), Detail: out}
	}
	return dto.TaskResult{Success: true, Message: i18n.T("cert.msg.convertDone", dstType, dstPath), Detail: out}
}

// DeleteAlias 删除密钥库中的别名。
func (a *App) DeleteAlias(path, storePass, alias string) dto.TaskResult {
	out, err := a.certMgr.DeleteAlias(context.Background(), path, storePass, alias)
	if err != nil {
		return dto.TaskResult{Success: false, Message: err.Error(), Detail: out}
	}
	return dto.TaskResult{Success: true, Message: i18n.T("cert.msg.deleteDone", alias), Detail: out}
}

// rememberKeystore 记录最近使用的密钥库与别名。
func (a *App) rememberKeystore(path, alias string) {
	if path == "" {
		return
	}
	_ = a.cfg.Update(func(cfg *config.Settings) {
		cfg.LastKeystore = path
		cfg.RecentFiles = config.AddRecent(cfg.RecentFiles, path)
		if alias != "" {
			cfg.LastAlias = alias
			cfg.RecentAlias = config.AddRecent(cfg.RecentAlias, alias)
		}
	})
}

// ---------------------------------------------------------------- 密钥库目录与导入

// keystoreDirPath 返回与可执行文件同级的 keystores 目录（自动创建）。
func keystoreDirPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(filepath.Dir(exe), "keystores")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// KeystoreDir 返回 keystores 目录路径（供前端设置默认保存位置）。
func (a *App) KeystoreDir() string {
	dir, err := keystoreDirPath()
	if err != nil {
		return ""
	}
	return dir
}

// ListKeystoreDir 列出 keystores 目录下的密钥库文件（仅文件信息，不含内容）。
func (a *App) ListKeystoreDir() ([]dto.KeystoreFile, error) {
	dir, err := keystoreDirPath()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	exts := map[string]bool{".jks": true, ".keystore": true, ".p12": true, ".pfx": true}
	out := make([]dto.KeystoreFile, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !exts[strings.ToLower(filepath.Ext(e.Name()))] {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, dto.KeystoreFile{
			Path:    filepath.Join(dir, e.Name()),
			Name:    e.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime().UnixMilli(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModTime > out[j].ModTime })
	return out, nil
}

// ImportKeystore 将外部密钥库复制进 keystores 目录（保留原文件），命名为 imported_<时间>.<原扩展名>。
func (a *App) ImportKeystore(srcPath, storePass string) dto.TaskResult {
	if srcPath == "" {
		return dto.TaskResult{Success: false, Message: i18n.T("cert.err.chooseImport")}
	}
	if _, err := a.certMgr.List(context.Background(), srcPath, storePass); err != nil {
		return dto.TaskResult{Success: false, Message: i18n.T("cert.err.importVerify") + ": " + err.Error()}
	}
	dir, err := keystoreDirPath()
	if err != nil {
		return dto.TaskResult{Success: false, Message: err.Error()}
	}
	ext := strings.ToLower(filepath.Ext(srcPath))
	if ext == "" {
		ext = ".jks"
	}
	dst := filepath.Join(dir, "imported_"+time.Now().Format("20060102_150405")+ext)
	if _, err := os.Stat(dst); err == nil {
		return dto.TaskResult{Success: false, Message: i18n.T("cert.err.importExists", dst)}
	}
	if err := copyFile(srcPath, dst); err != nil {
		return dto.TaskResult{Success: false, Message: i18n.T("cert.err.importCopy") + ": " + err.Error()}
	}
	a.rememberKeystore(dst, "")
	return dto.TaskResult{Success: true, Message: i18n.T("cert.msg.importDone", dst)}
}

// GetSavedPassword 返回指定密钥库保存的密码（已解密），未保存时返回空串。
func (a *App) GetSavedPassword(path string) string {
	s := a.cfg.Snapshot()
	cipher, ok := s.SavedPasswords[path]
	if !ok || cipher == "" {
		return ""
	}
	plain, err := secure.Decrypt(cipher, s.CryptoKey)
	if err != nil {
		return ""
	}
	return plain
}

// SavePassword 加密保存指定密钥库的密码（password 为空则清除）。
func (a *App) SavePassword(path, password string) error {
	if path == "" {
		return errors.New(i18n.T("app.err.emptyPath"))
	}
	return a.cfg.Update(func(cfg *config.Settings) {
		if cfg.SavedPasswords == nil {
			cfg.SavedPasswords = map[string]string{}
		}
		if password == "" {
			delete(cfg.SavedPasswords, path)
			return
		}
		cipher, err := secure.Encrypt(password, cfg.CryptoKey)
		if err != nil {
			return
		}
		cfg.SavedPasswords[path] = cipher
	})
}

// ForgetPassword 删除指定密钥库保存的密码。
func (a *App) ForgetPassword(path string) error {
	return a.cfg.Update(func(cfg *config.Settings) {
		delete(cfg.SavedPasswords, path)
	})
}

// copyFile 将源文件完整复制到目标路径。
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// ---------------------------------------------------------------- 签名与验证

// SignAPK 执行签名流水线，步骤通过 task:step 事件、日志通过 task:log 事件推送。
func (a *App) SignAPK(opts dto.SignOptions) dto.SignResult {
	cb := &signer.Callbacks{
		Secrets: []string{opts.StorePass, opts.KeyPass},
		OnStep:  func(s dto.Step) { a.emit("task:step", s) },
		OnLog: func(stream, text string) {
			a.emit("task:log", map[string]string{"stream": stream, "text": text})
		},
	}
	result, err := a.signMgr.Sign(context.Background(), opts, cb)
	if err != nil {
		logger.Error(i18n.T("sign.log.signFail"), "error", logger.Mask(err.Error(), opts.StorePass, opts.KeyPass))
		return dto.SignResult{Success: false, Message: err.Error()}
	}
	if result.Success {
		logger.Info(i18n.T("sign.log.signDone"), "output", result.OutputPath, "schemes", result.SignVersion)
		_ = a.cfg.Update(func(cfg *config.Settings) {
			cfg.LastApk = opts.InputAPK
			cfg.LastKeystore = opts.Keystore
			if opts.Alias != "" {
				cfg.LastAlias = opts.Alias
				cfg.RecentAlias = config.AddRecent(cfg.RecentAlias, opts.Alias)
			}
			cfg.RecentFiles = config.AddRecent(cfg.RecentFiles, opts.InputAPK)
		})
	}
	return *result
}

// VerifyAPK 验证 APK 签名。
func (a *App) VerifyAPK(apkPath string) dto.VerifyResult {
	result, err := a.signMgr.Verify(context.Background(), apkPath)
	if err != nil {
		return dto.VerifyResult{Verified: false, RawOutput: err.Error()}
	}
	return *result
}

// ZipAlign 单独执行对齐。
func (a *App) ZipAlign(inputAPK, outputAPK string) dto.TaskResult {
	out, err := a.signMgr.ZipAlign(context.Background(), inputAPK, outputAPK, nil)
	if err != nil {
		return dto.TaskResult{Success: false, Message: err.Error(), Detail: out}
	}
	return dto.TaskResult{Success: true, Message: i18n.T("sign.msg.zipalignDone", outputAPK), Detail: out}
}

// ---------------------------------------------------------------- APK 信息

// ParseAPK 解析 APK 信息。
func (a *App) ParseAPK(apkPath string) dto.APKInfo {
	info, err := a.apkParser.Parse(context.Background(), apkPath)
	if err != nil {
		logger.Warn(i18n.T("apkinfo.log.parseFail"), "path", apkPath, "error", err.Error())
		info.Message = err.Error()
	}
	if apkPath != "" {
		_ = a.cfg.Update(func(cfg *config.Settings) {
			cfg.LastApk = apkPath
			cfg.RecentFiles = config.AddRecent(cfg.RecentFiles, apkPath)
		})
	}
	return info
}
