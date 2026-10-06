# APK Signer Studio (go-apksigner-gui)

A desktop GUI wrapper around Google's `apksigner` toolchain, letting users complete the full workflow of "download tools → generate/view certificates → align → sign → verify" without memorizing command lines. The UI is in Chinese, with a single-page main window switching between multiple modules; all long-running operations show real-time progress and log output.

- Backend: Go 1.22+ + Wails v2 (Go backend + system WebView, single exe)
- Frontend: Vue 3 + TypeScript + Vite + TailwindCSS + TDesign, package manager **pnpm**
- Target platform: Windows (logic is cross-platform; Linux / macOS can also run it)

## 1. Feature Overview

| Module | Capabilities |
| --- | --- |
| Toolchain | Auto-detects `apksigner` / `zipalign` / `aapt2` / `keytool`; fetches the build-tools version list online and downloads to a local tool directory with progress, resumable downloads, and SHA256 verification; supports China mirror sources and HTTP/SOCKS5 proxies |
| Certificates | Generate JKS/PKCS12 keystores with keytool; view aliases, validity periods, and MD5/SHA1/SHA256 fingerprints; export certificates; convert between PKCS12 ↔ JKS |
| Signing | Select an APK and keystore, set the V1/V2/V3/V4 combination, output path, and SDK version restrictions; optionally zipalign before signing; supports removing old signatures; automatically verifies after signing and shows the certificate digest |
| APK Info | Reads package name, version name/code, minSdk/targetSdk, file size, and signature status (prefers aapt2, falls back to pure-Go parsing when missing) |
| Settings | Tool directory, manual tool paths, persistent mirror/proxy settings, default signing scheme, one-click temp file cleanup |

## 2. Environment Requirements

| Tool | Purpose | Notes |
| --- | --- | --- |
| Go 1.22+ | Compile backend | Verified with 1.26.3 |
| Node.js 18+ / pnpm | Build frontend | Verified with Node 24.21 / pnpm 11+ |
| Wails CLI | Generate bindings / package | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| JDK 21 | Provides `keytool` | Required for signing/certificate generation; apksigner/zipalign/aapt2 are downloaded via this tool |
| NSIS (optional) | Build Windows installer | Without it, `build.ps1 -Package` degrades to producing only the exe |

> On Windows, if the WebView2 runtime is not yet installed, Wails will guide you through installing it on first launch of the exe.

## 3. One-Click Build

`.\build.ps1` chains: install Go toolchain (as needed) → pnpm install frontend dependencies → build frontend → generate Wails bindings → `wails build` (by default `-s` skips the built-in frontend build to avoid duplication).

```powershell
# Normal build (produces build/go-apksigner-gui.exe)
.\build.ps1

# Skip frontend/deps, only recompile Go (common for debugging)
.\build.ps1 -SkipInstall -SkipFrontend

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
wails generate module
wails build -s
```

> pnpm 11+ disables dependency build scripts by default, and esbuild needs postinstall to generate platform binaries, so install with `pnpm install --allow-build esbuild` (see `frontend/pnpm-workspace.yaml`).

## 4. Development & Debugging

```powershell
# Start frontend hot reload and the desktop window together (wailsjs bindings must already be generated)
wails dev
```

Or debug only the frontend (the backend cannot compile standalone when `frontend/dist` doesn't exist):

```powershell
cd frontend
pnpm exec vite   # defaults to http://localhost:5173
```

## 5. First-Use Workflow

1. Open the app and go to the "Toolchain" page: wait for tool detection to complete. If `apksigner`/`zipalign`/`aapt2` are missing, choose a China mirror source (e.g. Tencent Cloud), click "Refresh version list", and install a build-tools version (downloads show a progress bar and support resuming after interruption).
2. Go to the "Certificates" page → Generate: fill in the alias, organization info, and validity period, set the keystore password, choose a save path, and click "Generate keystore". You can verify the alias and fingerprints in the "View" tab.
3. Go to the "APK Info" page: select the APK to sign and confirm the package name/version/signature status.
4. Go to the "Signing" page: select the APK, keystore, and alias, enter the password, check V1/V2/V3 (default), and keep "zipalign before signing" enabled (recommended). Click "Start signing". After completion it automatically verifies and shows the certificate digest.
5. The log drawer at the bottom shows real-time progress for each step; you can clear it or pause auto-scrolling.

## 6. Directory Structure

```
go-apksigner-gui/
├── main.go / app.go        # Wails entry and binding layer (exposes all frontend APIs)
├── wails.json / build.ps1  # Project configuration and one-click build script
├── internal/
│   ├── executor/  config/  logger/  fileutil/   # Infrastructure layer
│   ├── tools/      # Toolchain detection / mirror proxy / version index / resumable downloads
│   ├── certificate/ # keytool generation / bilingual output parsing / export / conversion
│   ├── signer/      # Parameter building / remove old signatures / zipalign / sign / verify pipeline
│   ├── apkinfo/     # aapt2 badging parsing + AXML fallback
│   └── dto/         # Unified frontend-backend JSON contracts
└── frontend/
    └── src/
        ├── api/        # Wails binding wrappers and event subscriptions
        ├── stores/     # Lightweight global state
        └── views/      # ToolsView / CertView / SignView / ApkInfoView / SettingsView
```

`frontend/wailsjs/` is auto-generated by `wails generate module`; do not modify it manually.

## 7. Testing

Core pure logic (no external processes required) has unit tests and is stable and repeatable:

```powershell
go test ./internal/...
```

Coverage: parameter building, keytool Chinese/English output parsing, apksigner verify determination, XML version parsing, version-code sorting, ZIP Slip protection, META-INF filtering, config read/write round-trips.

For logic involving the real toolchain (keytool / apksigner / zipalign), a manual smoke test is recommended: generate a test keystore → align → sign → verify. Each module's README description corresponds one-to-one with its equivalent command lines, making cross-verification easy.

## 8. FAQ

- **Downloading build-tools is slow / fails**: Switch mirror sources (Tencent Cloud / Tsinghua) on the "Settings" page, or configure an HTTP/SOCKS5 proxy.
- **Missing aapt2 causes incomplete APK info parsing**: The tool automatically degrades to the built-in AXML parser, which can only fill in basic fields; installing build-tools is recommended.
- **Where is the config file**: `os.UserConfigDir()/go-apksigner-gui/config.json`, permissions 0600, no passwords are stored.
- **`******` appears in logs**: Passwords are masked before being written to disk or pushed via events.
- **`go:embed` compile error**: When the frontend hasn't been built, `frontend/dist/index.html` doesn't exist; run `pnpm exec vite build` or `.\build.ps1` first.
