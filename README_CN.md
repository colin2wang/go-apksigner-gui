# APK Signer Studio（go-apksigner-gui）

用桌面 GUI 包装 Google `apksigner` 工具链，让使用者无需记忆命令行即可完成
「下载工具 → 生成/查看证书 → 对齐 → 签名 → 验证」的完整流程。界面为中文，
主窗口单页多模块切换，所有长耗时操作均有实时进度与日志输出。

- 后端：Go 1.22+ + Wails v2（Go 后端 + 系统 WebView，单一 exe）
- 前端：Vue 3 + TypeScript + Vite + TailwindCSS + TDesign，包管理器 **pnpm**
- 目标平台：Windows（逻辑已做跨平台处理，Linux / macOS 亦可运行）

## 一、功能一览

| 模块 | 能力 |
| --- | --- |
| 工具链 | 自动探测 `apksigner` / `zipalign` / `aapt2` / `keytool`；在线获取 build-tools 版本列表，带进度、断点续传、SHA256 校验下载到本地工具目录；支持国内镜像源与 HTTP/SOCKS5 代理 |
| 证书 | keytool 生成 JKS/PKCS12 密钥库；查看别名、有效期与 MD5/SHA1/SHA256 指纹；导出证书；PKCS12 ↔ JKS 互转 |
| 签名 | 选择 APK 与密钥库，设置 V1/V2/V3/V4 组合、输出路径、SDK 版本限制；可选先 zipalign 后签名；支持移除旧签名；签名后自动验证并回显证书摘要 |
| APK 信息 | 读取包名、版本名/版本号、minSdk/targetSdk、文件大小、是否已签名（优先 aapt2，缺失时纯 Go 解析兜底） |
| 设置 | 工具目录、手工指定工具路径、镜像源/代理持久化、默认签名方案、一键清理临时文件 |

## 二、环境依赖

| 工具 | 用途 | 说明 |
| --- | --- | --- |
| Go 1.22+ | 编译后端 | 已验证 1.26.3 |
| Node.js 18+ / pnpm | 构建前端 | 已验证 Node 24.21 / pnpm 11+ |
| Wails CLI | 生成绑定 / 打包 | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| JDK 21 | 提供 `keytool` | 签名/证书生成依赖；apksigner/zipalign/aapt2 通过本工具下载 |
| NSIS（可选） | 生成 Windows 安装包 | 未安装时 `build.ps1 -Package` 会降级为仅产出 exe |

> Windows 下若尚未安装 WebView2 运行时，首次运行 exe 时 Wails 会引导安装。

## 三、一键构建

`.\build.ps1` 串联：安装 Go 工具链（按需）→ pnpm 安装前端依赖 → 构建前端 →
生成 Wails 绑定 → `wails build`（默认已用 `-s` 跳过内置前端构建，避免重复）。

```powershell
# 普通构建（产出 build/go-apksigner-gui.exe）
.\build.ps1

# 跳过前端/依赖，仅重新编译 Go（调试时常用）
.\build.ps1 -SkipInstall -SkipFrontend

# 生成 NSIS 安装包（需本机已装 makensis）
.\build.ps1 -Package

# 强制清理 build 目录后重新构建
.\build.ps1 -Clean
```

也可分步手动执行：

```powershell
cd frontend
pnpm install --allow-build esbuild
pnpm exec vite build        # 产出 frontend/dist
cd ..
wails generate module
wails build -s
```

> pnpm 11+ 默认禁用依赖构建脚本，esbuild 需要 postinstall 生成平台二进制，
> 因此安装时使用 `pnpm install --allow-build esbuild`（见 `frontend/pnpm-workspace.yaml`）。

## 四、开发调试

```powershell
# 同时启动前端热更新与桌面窗口（需要已生成 wailsjs 绑定）
wails dev
```

或直接仅调试前端（`frontend/dist` 不存在时后端无法独立编译）：

```powershell
cd frontend
pnpm exec vite   # 默认 http://localhost:5173
```

## 五、首次使用流程

1. 打开应用，进入「工具链」页：等待工具探测完成。若 `apksigner`/`zipalign`/`aapt2`
   缺失，选择国内镜像源（如腾讯云），点击「刷新版本列表」后安装一个 build-tools 版本
   （下载带进度条，支持中断后断点续传）。
2. 进入「证书」页 → 生成：填写别名、组织信息与有效期，设置密钥库密码，选择保存路径，
   点击「生成密钥库」。可在「查看」标签中验证别名与指纹。
3. 进入「APK 信息」页：选择待签名 APK，确认包名/版本/是否已签名。
4. 进入「签名」页：选择 APK、密钥库与别名，填写密码，勾选 V1/V2/V3（默认），
   建议保留「签名前先 zipalign」。点击「开始签名」。完成后自动验证并显示证书摘要。
5. 底部日志抽屉实时输出各步骤进度，可清空、暂停自动滚动。

## 六、目录结构

```
go-apksigner-gui/
├── main.go / app.go        # Wails 入口与绑定层（暴露全部前端 API）
├── wails.json / build.ps1  # 项目配置与一键构建脚本
├── internal/
│   ├── executor/  config/  logger/  fileutil/   # 基础设施层
│   ├── tools/      # 工具链探测 / 镜像代理 / 版本索引 / 断点续传下载
│   ├── certificate/ # keytool 生成 / 中英双语输出解析 / 导出 / 转换
│   ├── signer/      # 参数构建 / 移除旧签名 / zipalign / sign / verify 流水线
│   ├── apkinfo/     # aapt2 badging 解析 + AXML 兜底
│   └── dto/         # 前后端统一 JSON 契约
└── frontend/
    └── src/
        ├── api/        # Wails 绑定封装与事件订阅
        ├── stores/     # 轻量全局状态
        └── views/      # ToolsView / CertView / SignView / ApkInfoView / SettingsView
```

`frontend/wailsjs/` 由 `wails generate module` 自动生成，请勿手动修改。

## 七、测试

核心纯逻辑（不依赖外部进程）均有单元测试，稳定可重复执行：

```powershell
go test ./internal/...
```

覆盖：参数构建、keytool 中英文输出解析、apksigner verify 判定、XML 版本解析、
版本号排序、ZIP Slip 防护、META-INF 过滤、配置读写往返。

真实工具链（keytool / apksigner / zipalign）相关逻辑建议手动冒烟：
生成测试密钥库 → 对齐 → 签名 → 验证，README 各模块说明与其等价命令行一一对应，
便于交叉验证。

## 八、常见问题

- **下载 build-tools 很慢 / 失败**：在「设置」页切换镜像源（腾讯云/清华），或配置 HTTP/SOCKS5 代理。
- **aapt2 缺失导致 APK 信息解析不全**：工具会自动降级为内置 AXML 解析，仅能回填基础字段，建议安装 build-tools。
- **配置文件在哪**：`os.UserConfigDir()/go-apksigner-gui/config.json`，权限 0600，不保存任何密码。
- **日志里出现 `******`**：密码在落盘与事件推送前已被脱敏。
- **`go:embed` 编译报错**：前端未构建时 `frontend/dist/index.html` 不存在，请先执行 `pnpm exec vite build` 或 `.\build.ps1`。
