<#
.SYNOPSIS
  Builds the Trenova Capture MSI from release binaries.

.DESCRIPTION
  Expects the workspace built for both targets:
    cargo build --release --target x86_64-pc-windows-msvc -p trenova-capture -p trenova-capture-scan -p trenova-capture-svc -p trenova-capture-update
    cargo build --release --target i686-pc-windows-msvc  -p trenova-capture-scan
  and the WiX v4 tool with its Util extension:
    dotnet tool install --global wix
    wix extension add -g WixToolset.Util.wixext

  Stages the binaries under installer\stage, signs them when -Sign is given
  (see sign.ps1), builds installer\out\TrenovaCapture-<version>-x64.msi, and
  signs that too.

.PARAMETER Version
  MAJOR.MINOR.PATCH. Defaults to the workspace version in Cargo.toml.

.PARAMETER Sign
  Sign every executable and the MSI with sign.ps1.
#>
[CmdletBinding()]
param(
    [string]$Version,
    [switch]$Sign
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$root = Split-Path -Parent $PSScriptRoot
$stage = Join-Path $PSScriptRoot 'stage'
$out = Join-Path $PSScriptRoot 'out'
$logo = Join-Path $root '..\..\client\apps\web\public\logo.ico'

if (-not $Version) {
    $manifest = Get-Content (Join-Path $root 'Cargo.toml') -Raw
    if ($manifest -notmatch '(?m)^\[workspace\.package\][^\[]*?^version\s*=\s*"([^"]+)"') {
        throw 'Could not read the workspace version from Cargo.toml; pass -Version.'
    }
    $Version = $Matches[1]
}
if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    throw "The version must be MAJOR.MINOR.PATCH, not '$Version'."
}

$binaries = @(
    @{ From = 'target\x86_64-pc-windows-msvc\release\trenova-capture.exe';        To = 'trenova-capture.exe' },
    @{ From = 'target\x86_64-pc-windows-msvc\release\trenova-capture-scan.exe';   To = 'trenova-capture-scan-x64.exe' },
    @{ From = 'target\i686-pc-windows-msvc\release\trenova-capture-scan.exe';     To = 'trenova-capture-scan-x86.exe' },
    @{ From = 'target\x86_64-pc-windows-msvc\release\trenova-capture-svc.exe';    To = 'trenova-capture-svc.exe' },
    @{ From = 'target\x86_64-pc-windows-msvc\release\trenova-capture-update.exe'; To = 'trenova-capture-update.exe' }
)

if (Test-Path $stage) { Remove-Item $stage -Recurse -Force }
New-Item -ItemType Directory -Path $stage, $out -Force | Out-Null

foreach ($binary in $binaries) {
    $from = Join-Path $root $binary.From
    if (-not (Test-Path $from)) {
        throw "Missing $from. Build the workspace for both targets first."
    }
    Copy-Item $from (Join-Path $stage $binary.To)
}

# The agent's window runs in WebView2, and the agent loads Microsoft's
# WebView2Loader.dll when it starts: without it beside trenova-capture.exe,
# the agent does not start at all. The WebView2 bindings' build copies the
# loader into their build output; the newest copy is the one this build used.
$loader = Get-ChildItem (Join-Path $root 'target\x86_64-pc-windows-msvc\release\build') -Directory -Filter 'webview2-com-sys-*' -ErrorAction SilentlyContinue |
    ForEach-Object { Join-Path $_.FullName 'out\x64\WebView2Loader.dll' } |
    Where-Object { Test-Path $_ } |
    Get-Item |
    Sort-Object LastWriteTime -Descending |
    Select-Object -First 1
if (-not $loader) {
    throw 'Missing WebView2Loader.dll in the x64 release build. Build trenova-capture for x86_64-pc-windows-msvc first.'
}
Copy-Item $loader.FullName (Join-Path $stage 'WebView2Loader.dll')

# Every executable carries the Visual C++ runtime itself (static_vcruntime in
# each build script): a Windows without the Visual C++ Redistributable has no
# VCRUNTIME140.dll or VCRUNTIME140_1.dll, and an executable that imports one
# does not start. dumpbin, from the MSVC tools the build already needs, lists
# what each one imports.
$vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio\Installer\vswhere.exe'
$dumpbin = if (Test-Path $vswhere) {
    & $vswhere -latest -products * -find 'VC\Tools\MSVC\**\bin\Hostx64\x64\dumpbin.exe' | Select-Object -First 1
}
if (-not $dumpbin) {
    throw 'dumpbin.exe was not found; install the Visual Studio C++ build tools.'
}
foreach ($exe in Get-ChildItem $stage -Filter *.exe) {
    $imports = & $dumpbin /nologo /dependents $exe.FullName
    if ($LASTEXITCODE -ne 0) { throw "dumpbin could not read $($exe.Name)." }
    $runtime = $imports | Where-Object { $_ -match '(?i)^\s*(vcruntime|msvcp)\d+\S*\.dll\s*$' }
    if ($runtime) {
        throw "$($exe.Name) imports $(($runtime | ForEach-Object { $_.Trim() }) -join ', '), which not every Windows has. Its build script must call static_vcruntime::metabuild()."
    }
    Write-Host "$($exe.Name) imports: $((($imports | Where-Object { $_ -match '(?i)\.dll\s*$' }) | ForEach-Object { $_.Trim() }) -join ', ')"
}

if ($Sign) {
    & (Join-Path $PSScriptRoot 'sign.ps1') -Path (Get-ChildItem $stage -Filter *.exe | ForEach-Object FullName)
}

$msi = Join-Path $out "TrenovaCapture-$Version-x64.msi"
wix build `
    -arch x64 `
    -d "Version=$Version" `
    -d "StageDir=$stage" `
    -d "LogoPath=$((Resolve-Path $logo).Path)" `
    -d "AssetsDir=$(Join-Path $PSScriptRoot 'assets')" `
    -ext WixToolset.Util.wixext `
    -ext WixToolset.UI.wixext `
    -culture en-US `
    -loc (Join-Path $PSScriptRoot 'Package.en-us.wxl') `
    -o $msi `
    (Join-Path $PSScriptRoot 'Package.wxs') `
    (Join-Path $PSScriptRoot 'Ui.wxs')
if ($LASTEXITCODE -ne 0) { throw "wix build failed with $LASTEXITCODE." }

if ($Sign) {
    & (Join-Path $PSScriptRoot 'sign.ps1') -Path $msi
}

Write-Host "Built $msi"
$msi
