# go-apksigner-gui 代码评审与修改建议
整体架构清晰（绑定层薄、业务下沉 internal、DTO 统一契约、日志脱敏意识到位），但存在几个**真实 Bug**、一处**安全性名不符实**的问题，以及若干健壮性/工程规范问题。按优先级列出如下。
---
## 一、高优先级（Bug / 安全，建议尽快修复）
### 1. 前端 `t()` 传字符串参数导致 `{0}` 被替换成首字符（广泛存在的 Bug）
`i18n/index.ts` 中 `t(key, params)` 只接受 `Record | unknown[]`，但大量调用点直接传 `String(err)`：
```ts
// SignView / ApkInfoView / CertView / SettingsView / ToolsView 多处
notify('error', t('apk.err.parse', String(err)))   // ❌
notify('error', t('settings.err.proxyTest', msg))  // ❌
```
字符串会走 `Object.entries(params)` 分支，`Object.entries("timeout")` 返回 `[["0","t"],["1","i"]…]`，最终显示为「代理测试失败： t」——**所有错误提示都只剩第一个字符**。
建议在 `t()` 内兜底修复（一行改动，向后兼容）：
```ts
export function t(key: string, params?: Record<string, unknown> | unknown[] | string): string {
  ...
  if (params) {
    const args = typeof params === 'string' ? [params] : params
    // 后续逻辑不变
  }
}
```
另外 `new RegExp(\`\\{${k}\\}\`)` 未转义命名参数中的正则特殊字符，顺手补上。
### 2. 密码「加密保存」实为混淆，README 声明与实现不符
`CryptoKey`（随机 32 字节）与 `SavedPasswords` 密文**存在同一个 config.json**（且注释称 0600 权限，但 Windows 上 0600 基本无效）。拿到该文件即可解密，README 中 "only usable on this machine/account" 的安全承诺并不成立。
建议二选一：
- 接入 OS 级凭据存储（Windows DPAPI `CryptProtectData`，或 wincred），主密钥交由系统托管；
- 至少在文档中如实降级描述为「防 casual 泄露，非强安全」，避免误导。
### 3. 密码通过命令行参数传递，进程列表可见
`keytool -storepass xxx` / apksigner 密码参数会出现在任务管理器/WMI 进程命令行中。建议：
- **apksigner**：改用 `--ks-pass env:VAR`（apksigner 原生支持 `pass:` / `env:` / `file:`），在子进程 Env 中注入；
- **keytool**：省略 `-storepass`，将密码写入 stdin（keytool 会交互式提示读取），需按目标 JDK 版本验证。
### 4. `SelectOpenFile` 默认过滤器格式错误（潜伏 Bug）
`app.go` 中 `apkFilter = "APK 文件 (*.apk):*.apk"` 是 Qt 风格格式，Wails v2 的 `FileFilter.Pattern` 只接受 `*.apk` 或 `*.jks;*.keystore` 这类模式串。当前前端总是传参所以未触发，但一旦空参调用默认值，Windows 对话框将匹配不到任何文件。建议改为纯模式串 + 独立 DisplayName，并顺带 i18n 化硬编码的「选择文件」「保存到」标题（`shutdown` 日志的硬编码中文同理）。
### 5. `SavePassword` 静默吞掉加密错误
```go
cipher, err := secure.Encrypt(password, cfg.CryptoKey)
if err != nil { return }  // ❌ UI 以为已保存，实际未存
```
建议先在 `Update` 外加密（可拿到 Snapshot 的 CryptoKey），失败即返回错误：
```go
cipher, err := secure.Encrypt(password, a.cfg.Snapshot().CryptoKey)
if err != nil { return err }
return a.cfg.Update(func(cfg *config.Settings) { cfg.SavedPasswords[path] = cipher })
```
### 6. 代理密码明文落盘
`settings.go` 注释说「除代理外凭据不落盘」，即代理密码设计上落盘，但目前是明文 JSON。建议与密钥库密码同样走 `secure.Encrypt`，或在 UI 明示风险。
### 7. 前端引用了不存在的 i18n key
`SettingsView.vue` 中 `t('settings.toolsDirTitle')`、`t('settings.sdkDirTitle')` 在 en.ts / zh-CN.ts 中均未定义，对话框标题会直接显示 key 原文。建议补齐（或复用 `settings.field.*`）。
### 8. 流水线步骤用「本地化文本」作 key，切语言后状态错位
后端 `OnStep` 推送的步骤名是 `i18n.T("sign.step.remove")` 等本地化字符串，前端用 `t('sign.step.*')` 匹配。**签名过程中切换语言会导致状态映射失败**。建议后端推送稳定 key（如 `"remove"`），展示名由前端 `t()` 翻译。
### 9. `DetectAll` 重复执行且无缓存/单飞
启动时 goroutine 里跑一次、`initStore → refreshTools` 再跑一次、每次 `SaveSettings` 还会同步跑一次（阻塞绑定调用）。每次检测会拉起 4~5 个子进程探测版本，开销明显。建议在 `tools.Manager` 内做 **single-flight + 短 TTL 缓存**，`SaveSettings` 改为异步 emit。
---
## 二、中优先级（健壮性 / 一致性）
- **`parseValidity` 只识别半角冒号**：中文 keytool 输出通常用全角「：」，`IndexAny(line, ":")` 会漏。当前靠 `-J-Duser.language=en` 规避，但中文解析路径声称兼容，建议补 `：`。
- **AXML 解析**：`decodeUTF16String` 不处理 UTF-16 代理对（emoji、CJK 扩展区会乱码）；`parseStringPool` 注释掉了 headerSize 读取、硬编码 28，建议按真实 headerSize 计算偏移。
- **`Manager.List` 返回 `(&info, err)` 双值**：调用方出错时丢弃 info，契约不一致，建议二选一。
- **缺少校验**：`Convert` 不校验密码非空与 dstType 枚举；`SignAPK` 未拦截 outputApk == inputApk（可能覆盖原文件）；`Export` 不校验 alias 非空。
- **不可取消**：签名流水线、`FetchBuildToolVersions`（`context.Background()`）都无取消/超时入口，建议绑定 `a.ctx` 并像安装一样提供 UI 取消。
- **保存密码的 key 未归一化路径**：Windows 大小写不敏感，`C:\A.jks` 与 `c:\a.jks` 会被视为两个 key。建议存储前 `filepath.Clean` + 统一小写（仅作 key，展示用原路径）。
- **`Store.Get()` 返回副本指针**：命名暗示可变引用，实则副本，极易误用，建议删除或改名。
- **`tools.Mirrors` 全局可变状态**：`app.go` 直接赋值包级变量，与并发读存在竞争，建议注入 Manager 字段。
- **死代码/死字段**：store 中 `requiredMissing` 计算后从未使用（header 目前把可选的 aapt2 缺失也算进「缺失 N 个工具」告警，建议改用它）；`SignOptions.debuggableApkPerm` 前端硬编码 false，建议暴露或删除。
- **`rememberPass` 默认勾选**：密码默认落盘涉及隐私偏好，建议默认不勾选。
---
## 三、工程 / 文档规范
1. **仓库卫生**：`frontend/build-output.txt`（构建报错日志）与 `package.json.md5` 被提交；README 引用的 `README.zh-CN.md`、`docs/images/*.png`、`wails.json` 在文件树中均不存在（断链）；缺 LICENSE 文件；建议补 GitHub Actions（`go test ./internal/...` + 前端构建）。
2. **`pnpm-workspace.yaml` 同时写了 `allowBuilds` 和 `onlyBuiltDependencies`**：前者疑似非标准键，且 build-output.txt 显示曾因 ignored builds 失败——建议对照 pnpm 文档确认实际生效键名，删掉冗余项。
3. **版本号硬编码**：`appVersion = "1.0.0"` 写死在 app.go，建议 build.ps1 通过 `-ldflags "-X main.appVersion=…"` 注入（顺带打 tag）。`go.mod` 的 `go 1.26` 较激进，建议加 `toolchain` 指令说明，避免贡献者工具链不一致。
4. **重复工具函数**：CertView 里 `fmtSize/fmtTime` 与 store 的 `formatSize` 重复，建议下沉到 utils。
5. **自动校验防抖的副作用**：查看页输入密码每停顿 600ms 就拉起一次 keytool，错误密码会反复触发子进程，建议改为失焦或回车触发。
6. **补充测试**：`t()` 字符串参数回归测试；`ParseKeytoolList` 全角冒号用例；Store 并发 Update/Save 竞态用例。
---
## 优先修复清单（速览）
| # | 问题 | 影响 | 位置 |
|---|------|------|------|
| 1 | `t()` 字符串参数 Bug | 所有错误提示显示不全 | 前端各 View + i18n/index.ts |
| 2 | 加密主密钥与密文同文件 | 安全承诺失效 | config + secure + README |
| 3 | 密码上命令行 | 进程列表泄露 | certificate/signer |
| 4 | FileFilter 默认值格式错 | 空参调用文件对话框损坏 | app.go |
| 5 | SavePassword 吞错 | 密码静默丢失 | app.go |
| 6 | 缺失 i18n key | 对话框显示 key 原文 | SettingsView + 语言包 |
| 7 | 步骤名本地化作 key | 切语言流水线错位 | signer + SignView |
| 8 | DetectAll 无缓存 | 启动/保存卡顿 | tools.Manager |
前 8 项改完，这个项目的可用性和安全叙事就能对齐 README 的宣传了。
