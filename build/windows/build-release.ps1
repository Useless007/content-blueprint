[CmdletBinding()]
param(
    [ValidateSet("amd64")]
    [string]$Architecture = "amd64",
    [string]$MakeNSISPath = "",
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

if ($env:OS -ne "Windows_NT") {
    throw "The Content Blueprint NSIS release can only be built on Windows."
}

$projectRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot "..\.."))
$installerRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot "installer"))
$payloadRoot = [System.IO.Path]::GetFullPath((Join-Path $installerRoot ".payload"))
$expectedPrefix = $installerRoot.TrimEnd([System.IO.Path]::DirectorySeparatorChar) + [System.IO.Path]::DirectorySeparatorChar
if (-not $payloadRoot.StartsWith($expectedPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Resolved installer payload directory is outside build/windows/installer."
}

& (Join-Path $projectRoot "build\verify-version.ps1")

function Invoke-Checked {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Command,
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $Command $($Arguments -join ' ')"
    }
}

foreach ($commandName in @("go", "wails")) {
    if (-not (Get-Command $commandName -ErrorAction SilentlyContinue)) {
        throw "Required command is not available on PATH: $commandName"
    }
}

$makeNSIS = $null
if (-not [string]::IsNullOrWhiteSpace($MakeNSISPath)) {
    $makeNSIS = (Resolve-Path -LiteralPath $MakeNSISPath -ErrorAction Stop).Path
}
else {
    $makeNSISCommand = Get-Command "makensis" -ErrorAction SilentlyContinue
    if ($makeNSISCommand) {
        $makeNSIS = $makeNSISCommand.Source
    }
    else {
        foreach ($candidate in @(
            "$env:ProgramFiles\NSIS\makensis.exe",
            "$env:ProgramFiles\NSIS\Bin\makensis.exe",
            "${env:ProgramFiles(x86)}\NSIS\makensis.exe",
            "${env:ProgramFiles(x86)}\NSIS\Bin\makensis.exe"
        )) {
            if (Test-Path -LiteralPath $candidate -PathType Leaf) {
                $makeNSIS = $candidate
                break
            }
        }
    }
}
if ([string]::IsNullOrWhiteSpace([string]$makeNSIS) -or
    -not (Test-Path -LiteralPath $makeNSIS -PathType Leaf)) {
    throw "makensis.exe is required. Install NSIS or pass -MakeNSISPath with its full path. See https://wails.io/docs/guides/windows-installer/."
}

$extensionFiles = @(
    "manifest.json",
    "icon-16.png",
    "icon-32.png",
    "icon-48.png",
    "icon.png",
    "service-worker.js",
    "content-script.js",
    "sidepanel.html",
    "sidepanel.css",
    "sidepanel.js",
    "src\core.js",
    "src\growth.js"
)
foreach ($relativePath in $extensionFiles) {
    $sourcePath = Join-Path $projectRoot (Join-Path "facebook-extension" $relativePath)
    if (-not (Test-Path -LiteralPath $sourcePath -PathType Leaf)) {
        throw "Required extension runtime file is missing: $sourcePath"
    }
}

$extensionManifestPath = Join-Path $projectRoot "facebook-extension\manifest.json"
$extensionManifest = Get-Content -LiteralPath $extensionManifestPath -Raw -Encoding UTF8 | ConvertFrom-Json
if ($extensionManifest.manifest_version -ne 3) {
    throw "facebook-extension/manifest.json must remain Manifest V3."
}
if ([string]::IsNullOrWhiteSpace([string]$extensionManifest.key)) {
    throw "facebook-extension/manifest.json must retain its stable key."
}
$keyBytes = [System.Convert]::FromBase64String([string]$extensionManifest.key)
$sha256 = [System.Security.Cryptography.SHA256]::Create()
try {
    $keyHash = $sha256.ComputeHash($keyBytes)
}
finally {
    $sha256.Dispose()
}
$extensionIdBuilder = New-Object System.Text.StringBuilder
foreach ($byte in $keyHash[0..15]) {
    [void]$extensionIdBuilder.Append([char]([int][char]'a' + ($byte -shr 4)))
    [void]$extensionIdBuilder.Append([char]([int][char]'a' + ($byte -band 0x0f)))
}
$extensionId = $extensionIdBuilder.ToString()
$lockedExtensionId = "ppncejmpiekmkepaeccdnpnpgdcfafje"
if ($extensionId -ne $lockedExtensionId) {
    throw "The extension key resolves to '$extensionId', not the locked ID '$lockedExtensionId'."
}

$nativeManifestPath = Join-Path $installerRoot "resources\native-host-manifest.json"
$nativeManifest = Get-Content -LiteralPath $nativeManifestPath -Raw -Encoding UTF8 | ConvertFrom-Json
$expectedOrigin = "chrome-extension://$lockedExtensionId/"
if ($nativeManifest.name -ne "com.contentblueprint.facebook" -or
    $nativeManifest.path -ne "content-blueprint-companion.exe" -or
    $nativeManifest.type -ne "stdio" -or
    @($nativeManifest.allowed_origins).Count -ne 1 -or
    @($nativeManifest.allowed_origins)[0] -ne $expectedOrigin) {
    throw "The packaged Native Messaging manifest does not match the locked host identity and extension origin."
}

if (-not (Test-Path -LiteralPath $payloadRoot -PathType Container)) {
    New-Item -ItemType Directory -Path $payloadRoot | Out-Null
}
$payloadExecutable = Join-Path $payloadRoot "content-blueprint-companion.exe"
if (Test-Path -LiteralPath $payloadExecutable -PathType Leaf) {
    Remove-Item -LiteralPath $payloadExecutable -Force
}

Push-Location $projectRoot
$originalPath = $env:PATH
try {
    $env:PATH = ([System.IO.Path]::GetDirectoryName($makeNSIS)) + [System.IO.Path]::PathSeparator + $originalPath
    if (-not $SkipTests) {
        Invoke-Checked -Command "go" -Arguments @("test", "./...")
        Push-Location (Join-Path $projectRoot "facebook-extension")
        try {
            Invoke-Checked -Command "npm" -Arguments @("test")
        }
        finally {
            Pop-Location
        }
    }

    Invoke-Checked -Command "go" -Arguments @(
        "build",
        "-trimpath",
        "-o", $payloadExecutable,
        ".\cmd\content-blueprint-companion"
    )

    Invoke-Checked -Command "wails" -Arguments @(
        "build",
        "-clean",
        "-nsis",
        "-installscope", "user",
        "-platform", "windows/$Architecture",
        "-trimpath",
        "-webview2", "download"
    )
}
finally {
    $env:PATH = $originalPath
    Pop-Location
}

$binDirectory = Join-Path $projectRoot "build\bin"
$installerPath = Join-Path $binDirectory "content-blueprint-$Architecture-installer.exe"
$applicationPath = Join-Path $binDirectory "content-blueprint.exe"
$companionPath = Join-Path $binDirectory "content-blueprint-companion.exe"
foreach ($artifactPath in @($installerPath, $applicationPath)) {
    if (-not (Test-Path -LiteralPath $artifactPath -PathType Leaf)) {
        throw "Expected release artifact was not created: $artifactPath"
    }
}
Copy-Item -LiteralPath $payloadExecutable -Destination $companionPath -Force

$archiveLister = Get-Command "7z" -ErrorAction SilentlyContinue
if ($archiveLister) {
    $archiveListing = & $archiveLister.Source "l" "-ba" $installerPath
    if ($LASTEXITCODE -ne 0) {
        throw "7z could not inspect the generated NSIS installer."
    }
    $archiveText = $archiveListing -join "`n"
    $expectedArchivePaths = @(
        "content-blueprint.exe",
        "INSTALLATION.txt",
        "native-host\content-blueprint-companion.exe",
        "native-host\com.contentblueprint.facebook.json",
        "facebook-extension\manifest.json",
        "facebook-extension\icon-16.png",
        "facebook-extension\icon-32.png",
        "facebook-extension\icon-48.png",
        "facebook-extension\icon.png",
        "facebook-extension\service-worker.js",
        "facebook-extension\content-script.js",
        "facebook-extension\sidepanel.html",
        "facebook-extension\sidepanel.css",
        "facebook-extension\sidepanel.js",
        "facebook-extension\src\core.js",
        "facebook-extension\src\growth.js"
    )
    foreach ($archivePath in $expectedArchivePaths) {
        if (-not $archiveText.Contains($archivePath)) {
            throw "Generated installer is missing an expected payload: $archivePath"
        }
    }
    if ($archiveText -match '(?i)(^|[\\/])(node_modules|tests)([\\/]|$)|package(?:-lock)?\.json|README\.md') {
        throw "Generated installer contains a development-only extension file."
    }
}
else {
    Write-Warning "7z was not found; exact NSIS archive-content inspection was skipped."
}

$hashPath = Join-Path $binDirectory "SHA256SUMS.txt"
$hashLines = foreach ($artifactPath in @($installerPath, $applicationPath, $companionPath)) {
    $hash = Get-FileHash -LiteralPath $artifactPath -Algorithm SHA256
    "{0}  {1}" -f $hash.Hash.ToLowerInvariant(), ([System.IO.Path]::GetFileName($artifactPath))
}
[System.IO.File]::WriteAllLines($hashPath, $hashLines, (New-Object System.Text.UTF8Encoding($false)))

Write-Host "Windows release built without running the installer."
Get-Item -LiteralPath $installerPath, $applicationPath, $companionPath, $hashPath |
    Select-Object FullName, Length, LastWriteTime
