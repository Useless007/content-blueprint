[CmdletBinding()]
param(
    [string]$CompanionExecutable = ""
)

$ErrorActionPreference = "Stop"
$hostName = "com.contentblueprint.facebook"
$extensionId = "ppncejmpiekmkepaeccdnpnpgdcfafje"
if ([string]::IsNullOrWhiteSpace($CompanionExecutable)) {
    $CompanionExecutable = Join-Path $PSScriptRoot "..\..\build\bin\content-blueprint-companion.exe"
}
$localData = [Environment]::GetFolderPath("LocalApplicationData")
if ([string]::IsNullOrWhiteSpace($localData)) {
    throw "Windows LocalApplicationData directory is unavailable."
}

$resolvedExecutable = (Resolve-Path -LiteralPath $CompanionExecutable).Path
if (-not [System.IO.File]::Exists($resolvedExecutable)) {
    throw "Companion executable was not found: $resolvedExecutable"
}

$targetDirectory = [System.IO.Path]::GetFullPath((Join-Path $localData "ContentBlueprint\NativeMessaging"))
$allowedRoot = [System.IO.Path]::GetFullPath((Join-Path $localData "ContentBlueprint"))
if (-not $targetDirectory.StartsWith($allowedRoot + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Resolved native-host directory is outside ContentBlueprint LocalAppData."
}
[System.IO.Directory]::CreateDirectory($targetDirectory) | Out-Null

$hostManifestPath = Join-Path $targetDirectory "$hostName.json"
$hostManifest = [ordered]@{
    name = $hostName
    description = "Content Blueprint Facebook Companion"
    path = $resolvedExecutable
    type = "stdio"
    allowed_origins = @("chrome-extension://$extensionId/")
} | ConvertTo-Json -Depth 4
$utf8WithoutBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText($hostManifestPath, $hostManifest, $utf8WithoutBom)

$registryPaths = @(
    "HKCU:\Software\Google\Chrome\NativeMessagingHosts\$hostName",
    "HKCU:\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\$hostName"
)
foreach ($registryPath in $registryPaths) {
    New-Item -Path $registryPath -Force | Out-Null
    Set-Item -Path $registryPath -Value $hostManifestPath
}

Write-Host "Installed Chrome/Brave Native Messaging host: $hostName"
Write-Host "Manifest: $hostManifestPath"
Write-Host "Extension ID: $extensionId"
Write-Host "Reload the unpacked extension at chrome://extensions or brave://extensions after installation."
