[CmdletBinding()]
param(
    [string]$ExpectedVersion = ""
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$projectRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$versionPath = Join-Path $projectRoot "VERSION"
if (-not (Test-Path -LiteralPath $versionPath -PathType Leaf)) {
    throw "VERSION is missing."
}

$sourceVersion = (Get-Content -LiteralPath $versionPath -Raw -Encoding UTF8).Trim()
if ($sourceVersion -notmatch '^\d+\.\d+\.\d+$') {
    throw "VERSION must contain a stable SemVer value such as 0.3.0."
}
if (-not [string]::IsNullOrWhiteSpace($ExpectedVersion) -and $ExpectedVersion -ne $sourceVersion) {
    throw "Expected version '$ExpectedVersion' does not match VERSION '$sourceVersion'."
}

function Read-JsonFile {
    param([Parameter(Mandatory = $true)][string]$RelativePath)

    $path = Join-Path $projectRoot $RelativePath
    return Get-Content -LiteralPath $path -Raw -Encoding UTF8 | ConvertFrom-Json
}

function Assert-Version {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [AllowEmptyString()][string]$Actual
    )

    if ($Actual -ne $sourceVersion) {
        throw "$Name declares '$Actual'; expected '$sourceVersion'."
    }
}

function Read-PackageLockVersions {
    param([Parameter(Mandatory = $true)][string]$RelativePath)

    # Windows PowerShell 5.1 rejects package-lock.json's required empty-string
    # root-package key. The first two version fields are the lock and root
    # package versions in npm lockfile v3, so inspect those fields directly.
    $path = Join-Path $projectRoot $RelativePath
    $source = Get-Content -LiteralPath $path -Raw -Encoding UTF8
    $matches = [regex]::Matches($source, '"version"\s*:\s*"(?<version>[^"]+)"')
    if ($matches.Count -lt 2) {
        throw "$RelativePath does not contain lockfile and root-package versions."
    }
    return @($matches[0].Groups['version'].Value, $matches[1].Groups['version'].Value)
}

$extensionManifest = Read-JsonFile "facebook-extension\manifest.json"
$extensionPackage = Read-JsonFile "facebook-extension\package.json"
$frontendPackage = Read-JsonFile "frontend\package.json"
$wailsProject = Read-JsonFile "wails.json"
$extensionLockVersions = Read-PackageLockVersions "facebook-extension\package-lock.json"
$frontendLockVersions = Read-PackageLockVersions "frontend\package-lock.json"

Assert-Version "Extension manifest" ([string]$extensionManifest.version)
Assert-Version "Extension package" ([string]$extensionPackage.version)
Assert-Version "Extension package lock" $extensionLockVersions[0]
Assert-Version "Extension lock root package" $extensionLockVersions[1]
Assert-Version "Frontend package" ([string]$frontendPackage.version)
Assert-Version "Frontend package lock" $frontendLockVersions[0]
Assert-Version "Frontend lock root package" $frontendLockVersions[1]
Assert-Version "Wails product metadata" ([string]$wailsProject.info.productVersion)

$versionSource = Get-Content -LiteralPath (Join-Path $projectRoot "internal\versioninfo\version.go") -Raw -Encoding UTF8
$versionMatch = [regex]::Match($versionSource, 'CurrentVersion\s*=\s*"(?<version>[^"]+)"')
if (-not $versionMatch.Success) {
    throw "internal/versioninfo/version.go does not declare CurrentVersion."
}
Assert-Version "Go versioninfo" $versionMatch.Groups['version'].Value

$installerSource = Get-Content -LiteralPath (Join-Path $projectRoot "build\windows\installer\project.nsi") -Raw -Encoding UTF8
$installerMatch = [regex]::Match($installerSource, '!define\s+INFO_PRODUCTVERSION\s+"(?<version>[^"]+)"')
if (-not $installerMatch.Success) {
    throw "project.nsi does not declare INFO_PRODUCTVERSION."
}
Assert-Version "NSIS installer" $installerMatch.Groups['version'].Value

$installationText = Get-Content -LiteralPath (Join-Path $projectRoot "build\windows\installer\resources\INSTALLATION.txt") -Raw -Encoding UTF8
if (-not $installationText.StartsWith("Content Blueprint $sourceVersion", [System.StringComparison]::Ordinal)) {
    throw "INSTALLATION.txt does not start with Content Blueprint $sourceVersion."
}

Write-Host "Version declarations agree: $sourceVersion"
