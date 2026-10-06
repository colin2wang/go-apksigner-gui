# APK Signer Studio (go-apksigner-gui)

[中文版](README.zh-CN.md)

A desktop GUI wrapper around Google's `apksigner` toolchain, letting users complete the full workflow of "install toolchain → generate/view certificates → align → sign → verify" without memorizing command lines. The UI is bilingual (Simplified Chinese / English, Chinese by default); the main window is a single page with multiple modules, and every long-running operation streams real-time progress and log output.

- Backend: Go + Wails v2 (Go backend + system WebView, single exe)
- Frontend: Vue 3 + TypeScript + Vite + TailwindCSS + TDesign, package manager **pnpm**
- Target platform: Windows (logic is cross-platform; Linux / macOS can also run it)

## 1. Screenshots

| Toolchain | Certificates |
| --- | --- |
| ![Toolchain page: auto-detects apksigner / zipalign / aapt2 / keytool / sdkmanager, showing versions and paths](docs/images/screenshot-toolchain.png) | ![Certificates page: keystore directory; click a keystore and enter its password to view details](docs/images/screenshot-certificates.png) |

| Align & Sign | Settings |
| --- | --- |
| ![Align & Sign page: choose the APK to sign, output path, keystore and alias](docs/images/screenshot-align-sign.png) | ![Settings page: tools directory, manual tool paths, interface language, etc.](docs/images/screenshot-settings.png) |

## 2. Feature Overview

| Module | Capabilities |
| --- | --- |
| Toolchain | Auto-detects `apksigner` / `zipalign` / `aapt2` / `keytool` / `sdkmanager`; fetches the build-tools version list from the repository index (falls back to the candidate list in `app.yaml` when unreachable); installs a build-tools version via `sdkmanager`, showing progress and supporting cancellation; supports China mirror sources and HTTP/HTTPS/SOCKS5 proxies with a one-click connectivity test |
| Certificates | Five tabs: **Directory / Import / Generate / View / Convert**. Generate JKS/PKCS12 keystores with `keytool` (RSA 2048·4096 or EC); view aliases, validity periods, owner/issuer, key algorithm and MD5/SHA1/SHA256 fingerprints; export a certificate to PEM; convert PKCS12 ↔ JKS; delete an alias (with confirmation); import an external keystore into the built-in `keystores` directory; optionally remember the keystore password (AES-256-GCM encrypted) |
| Signing | Pick the APK, keystore and alias, set the V1/V2/V3/V4 combination, output path and min/max SDK restrictions; optionally zipalign before signing; optionally remove old signatures first; shows the pipeline (remove → align → sign → verify) as it runs; automatically verifies after signing and shows the certificate digest and warnings |
| APK Info | Read package name, app label, version name/code, minSdk/targetSdk, file size and signing status; also verify an APK standalone. Prefers `aapt2`, falls back to the built-in pure-Go AXML parser when aapt2 is missing |
| Settings | Tools directory, manual per-tool paths, Android SDK root path, interface language (zh-CN / en), mirror source and proxy (with connectivity test), default signing scheme, one-click temp/cache cleanup, re-detect toolchain |

Tool detection order for each tool: user-specified path → local tools directory → configured Android SDK root → `JAVA_HOME` / common install locations (keytool only) → system `PATH`. On Windows, `apksigner` is preferentially invoked as `java -jar apksigner.jar` to avoid `cmd /c` re-parsing passwords that contain characters such as `"` `%` `&` `|` `(` `)`.

## 3. Environment Requirements

| Tool | Purpose | Notes |
| --- | --- | --- |
| Go | Compile backend | `go.mod` declares `go 1.26`; Wails v2.16 |
| Node.js 18+ / pnpm | Build frontend | Vite 6 / Vue 3.5 / TypeScript 5.7 |
| Wails CLI | Generate bindings / package | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| JDK (8+) | Provides `keytool` and `java` | Required for certificate generation and for `sdkmanager`-based build-tools installation |
| NSIS (optional) | Build Windows installer | Without it, `build.ps1 -Package` degrades to producing only the exe |

> On Windows, if the WebView2 runtime is not yet installed, Wails will guide you through installing it on first launch of the exe.

## 4. One-Click Build

`.\build.ps1` chains: check the toolchain (Go / pnpm / Wails CLI) → `pnpm install` → `wails build` (which regenerates the Go↔frontend bindings **and** builds the frontend, so the embedded `frontend/dist` always reflects the latest backend methods) → copy `app.yaml` next to the executable.

```powershell
# Normal build (produces build\bin\apk-signer-studio.exe)
.\build.ps1

# Skip dependency install (reuse node_modules)
.\build.ps1 -SkipInstall

# Skip the built-in frontend build (adds -s, reuses existing frontend/dist)
.\build.ps1 -SkipFrontend

# Generate NSIS installer (requires makensis installed locally)
.\build.ps1 -Package

# Force-clean the build directory and rebuild
.\build.ps1 -Clean
```

You can also run the steps manually:

```powershell
cd frontend
pnpm install --allow-build esbuild
pnpm exec vite build        # produces frontend/dist
cd ..
wails build -s              # -s: skip the built-in frontend build
```

> pnpm 11+ disables dependency build scripts by default, and esbuild needs postinstall to generate platform binaries, so install with `pnpm install --allow-build esbuild` (see `frontend/pnpm-workspace.yaml`).

`app.yaml` in the repository root is copied to the exe directory at build time and takes precedence over built-in defaults. It can be edited (without recompiling) to adjust:

| Section | Contents |
| --- | --- |
| `sdkmanager` | Download URL (and optional SHA256) of the Android command-line tools per OS |
| `buildTools` | Default version and the offline candidate version list |
| `mirrors` | Repository index mirror sources (Google, Tencent Cloud, Tsinghua, NEUSOFT, USTC) |
| `proxyTest` | Target URL and timeout used by the Settings → "Test proxy" button |
| `download` | Download timeout and retry count |

## 5. Development & Debugging

```powershell
# Start frontend hot reload and the desktop window together
wails dev
```

Or debug only the frontend (the backend cannot compile standalone when `frontend/dist` doesn't exist):

```powershell
cd frontend
pnpm exec vite   # defaults to http://localhost:5173
```

`frontend/wailsjs/` is generated by `wails build` / `wails dev`; do not modify it manually.

## 6. First-Use Workflow

1. Open the app and go to the **Toolchain** page: wait for detection to finish. If `apksigner`/`zipalign`/`aapt2` are missing, go to **Settings** and pick a China mirror source (e.g. Tencent Cloud), then return and click "Refresh version list" and install a build-tools version. The first installation automatically downloads the Android command-line tools and then runs `sdkmanager build-tools;<version>`; a progress area shows the stage and a cancel button is available.
2. Go to **Certificates → Generate**: fill in the alias, common name, organization info and validity period, set the keystore password (≥ 6 chars), choose a save path (defaults to the `keystores` directory next to the exe), and click "Generate keystore". You can verify the alias and fingerprints in the **View** tab.
3. Go to **APK Info**: select the APK to sign and confirm the package name/version/signature status.
4. Go to **Signing**: select the APK, keystore and alias, enter the password, check V1/V2/V3 (default), and keep "zipalign before signing" enabled (recommended). Click "Start signing". Afterwards it automatically verifies and shows the certificate digest.
5. The log drawer at the bottom shows real-time output for each step; you can clear it or pause auto-scrolling.

## 7. Directory Structure

```
go-apksigner-gui/
├── main.go / app.go        # Wails entry and binding layer (exposes all frontend APIs)
├── app.yaml                # external config: mirrors / versions / sdkmanager URL / proxy test
├── wails.json / build.ps1  # project configuration and one-click build script
├── internal/
│   ├── executor/  fileutil/  logger/   # process execution, ZIP/file helpers, logs
│   ├── i18n/      secure/              # backend bilingual messages, AES-256-GCM crypto
│   ├── config/    config/appcfg/       # settings persistence + app.yaml loading
│   ├── tools/      # detection / mirror+proxy / version index / resumable download / sdkmanager install
│   ├── certificate/# keytool generate / output parsing / export / convert / delete alias
│   ├── signer/     # args building / remove old signature / zipalign / sign / verify pipeline
│   ├── apkinfo/    # aapt2 badging parsing + AXML fallback
│   └── dto/        # unified frontend-backend JSON contracts
└── frontend/
    └── src/
        ├── api/        # Wails binding wrappers and event subscriptions
        ├── i18n/       # zh-CN / en message catalogs
        ├── stores/     # lightweight global state
        └── views/      # ToolsView / CertView / SignView / ApkInfoView / SettingsView
```

## 8. Testing

Core pure logic (no external processes required) has unit tests and is stable and repeatable:

```powershell
go test ./internal/...
```

Coverage across 9 test files: tool index/version parsing, sign argument building, keytool Chinese/English output parsing, certificate info extraction, aapt2/AXML APK parsing, ZIP extraction (ZIP Slip protection, META-INF filtering), process execution timeouts, log masking, config read/write round-trips.

For logic involving the real toolchain (`keytool` / `apksigner` / `zipalign` / `sdkmanager`), a manual smoke test is recommended: generate a test keystore → align → sign → verify.

## 9. FAQ

- **Downloading / installing build-tools is slow or fails**: Switch mirror sources (Tencent Cloud / Tsinghua / NEUSOFT / USTC) on the Settings page, or configure an HTTP/HTTPS/SOCKS5 proxy and use "Test proxy" to confirm connectivity. The version list also falls back to the offline candidate list in `app.yaml` when the index is unreachable.
- **`sdkmanager` install reports failure**: `sdkmanager` itself requires Java. Make sure `java` is on `PATH` or `JAVA_HOME` is set (the JDK providing `keytool` already satisfies this). The command-line tools package is downloaded from the URL configured in `app.yaml`.
- **Missing aapt2 causes incomplete APK info parsing**: The tool automatically degrades to the built-in AXML parser, which only fills in basic fields; installing build-tools is recommended.
- **Where is the config file**: `os.UserConfigDir()/go-apksigner-gui/config.json`, permissions 0600.
- **Are keystore passwords stored on disk?**: Only if you explicitly tick "remember password". They are encrypted with AES-256-GCM using a random 32-byte master key that lives in the same 0600 config file, so they are only usable on this machine/account. You can remove a saved password at any time.
- **`******` appears in logs**: Passwords are masked before being written to disk or pushed via events.
- **`go:embed` compile error**: When the frontend hasn't been built, `frontend/dist/index.html` doesn't exist; run `pnpm exec vite build` or `.\build.ps1` first.
