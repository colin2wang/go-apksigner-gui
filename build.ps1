#Requires -Version 5.1
<#
.SYNOPSIS
    go-apksigner-gui one-click build script: first compile frontend (Vite + pnpm), then compile Go backend (Wails v2).
.PARAMETER SkipInstall
    Skip pnpm install (use when dependencies already exist)
.PARAMETER SkipFrontend
    Skip frontend build (reuse existing frontend/dist)
.PARAMETER Package
    Additionally generate Windows installer using NSIS (NSIS must be installed locally)
.PARAMETER Clean
    Clean build directory before build
.EXAMPLE
    .\build.ps1
    .\build.ps1 -Package
    .\build.ps1 -SkipInstall -SkipFrontend
#>

[CmdletBinding()]
param(
    [switch]$SkipInstall,
    [switch]$SkipFrontend,
    [switch]$Package,
    [switch]$Clean
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path

# Print step info with cyan color
function Write-Step($msg) { Write-Host "`n[STEP] $msg" -ForegroundColor Cyan }
# Print success message with green color
function Write-Ok($msg)   { Write-Host "[ OK ] $msg" -ForegroundColor Green }
# Print warning message with yellow color
function Write-Warn($msg) { Write-Host "[WARN] $msg" -ForegroundColor Yellow }
# Print failure message with red color
function Write-Fail($msg) { Write-Host "[FAIL] $msg" -ForegroundColor Red }

# Check if required command exists; exit with hint if missing
function Assert-Command($cmd, $hint) {
    $found = Get-Command $cmd -ErrorAction SilentlyContinue
    if (-not $found) {
        Write-Fail "Command not found: $cmd. $hint"
        exit 1
    }
    return $found
}

try {
    Push-Location $root

    Write-Step 'Check build environment'
    Assert-Command 'go' 'Please install Go 1.22+: https://go.dev/dl/' | Out-Null
    Assert-Command 'pnpm' 'Please install pnpm: npm i -g pnpm' | Out-Null
    Write-Ok ('go   : ' + (go version))
    Write-Ok ('pnpm : ' + (pnpm -v))

    # Wails CLI: responsible for frontend-backend binding generation and app packaging
    if (-not (Get-Command wails -ErrorAction SilentlyContinue)) {
        Write-Warn 'wails CLI not detected, installing github.com/wailsapp/wails/v2/cmd/wails@latest ...'
        go install github.com/wailsapp/wails/v2/cmd/wails@latest
        if ($LASTEXITCODE -ne 0) { Write-Fail 'Failed to install wails CLI'; exit 1 }
        $goBin = & go env GOPATH
        $env:Path = "$goBin\bin;$env:Path"
    } else {
        wails version
    }

    # Clean build folder if -Clean switch is set
    if ($Clean -and (Test-Path (Join-Path $root 'build'))) {
        Write-Step 'Clean build directory'
        Remove-Item -Recurse -Force (Join-Path $root 'build')
        Write-Ok 'build directory cleaned'
    }

    # Install frontend dependencies unless skipped
    if (-not $SkipInstall) {
        Write-Step 'Install frontend dependencies (pnpm install)'
        Push-Location (Join-Path $root 'frontend')
        try {
            # pnpm disables dependency build scripts by default; esbuild needs postinstall
            pnpm install --allow-build esbuild
            if ($LASTEXITCODE -ne 0) {
                Write-Warn 'Install with --allow-build failed, fallback to plain install'
                pnpm install
                if ($LASTEXITCODE -ne 0) { Write-Fail 'pnpm install failed'; exit 1 }
            }
        } finally { Pop-Location }
        Write-Ok 'Frontend dependencies ready'
    }

    # Wails (v2.16) has no standalone `wails generate bindings` command; the
    # Go<->frontend bindings are regenerated automatically by `wails build`.
    # Therefore we let `wails build` (below) generate the bindings AND build the
    # frontend itself (i.e. WITHOUT -s). This guarantees the embedded
    # frontend/dist always contains the latest backend methods such as
    # SetLanguage, so in-app language switching works at runtime. Pass
    # -SkipFrontend to reuse an already-built dist instead.

    Write-Step 'Compile desktop application (wails build)'
    # Build the frontend inside wails: it regenerates bindings first, then builds
    # the frontend, so the embedded dist reflects the latest backend methods.
    # -SkipFrontend reuses an already-built dist (skips the internal frontend build).
    $buildArgs = @('build', '-ldflags', '-s -w')
    if ($SkipFrontend) { $buildArgs += '-s' }
    if ($Package) {
        if (Get-Command makensis -ErrorAction SilentlyContinue) {
            Write-Ok 'NSIS detected, installer will be generated'
            $buildArgs += '-nsis'
        } else {
            Write-Warn 'NSIS(makensis) not detected, only executable will be built. Install NSIS and add -Package to generate installer.'
        }
    }
    wails @buildArgs
    if ($LASTEXITCODE -ne 0) { Write-Fail 'Go backend compilation failed'; exit 1 }

    # 复制外部配置文件 app.yaml 到 exe 同目录（运行时优先读取，缺失则回退内置默认值）
    Write-Step 'Copy external configuration (app.yaml)'
    $appYaml = Join-Path $root 'app.yaml'
    $binDir = Join-Path $root 'build\bin'
    if (Test-Path $appYaml) {
        if (-not (Test-Path $binDir)) { New-Item -ItemType Directory -Force -Path $binDir | Out-Null }
        Copy-Item -Force $appYaml (Join-Path $binDir 'app.yaml')
        Write-Ok ('app.yaml -> ' + (Join-Path $binDir 'app.yaml'))
    } else {
        Write-Warn 'app.yaml 未在仓库根目录找到，已跳过复制（应用将使用内置默认配置）'
    }

    Write-Step 'Build completed'
    $out = Join-Path $root 'build\bin'
    if (Test-Path $out) {
        Get-ChildItem -Path $out -Recurse -File |
            Select-Object FullName, @{n = 'SizeMB'; e = { [math]::Round($_.Length / 1MB, 2) } } |
            Format-Table -AutoSize
    }
    Write-Host "`nRun dev debug mode: wails dev" -ForegroundColor DarkGray
} finally {
    Pop-Location
}
