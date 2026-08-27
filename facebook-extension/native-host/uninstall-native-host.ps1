[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$hostName = "com.contentblueprint.facebook"
$localData = [Environment]::GetFolderPath("LocalApplicationData")
if ([string]::IsNullOrWhiteSpace($localData)) {
    throw "Windows LocalApplicationData directory is unavailable."
}

$targetDirectory = [System.IO.Path]::GetFullPath((Join-Path $localData "ContentBlueprint\NativeMessaging"))
$allowedRoot = [System.IO.Path]::GetFullPath((Join-Path $localData "ContentBlueprint"))
if (-not $targetDirectory.StartsWith($allowedRoot + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Resolved native-host directory is outside ContentBlueprint LocalAppData."
}
$hostManifestPath = Join-Path $targetDirectory "$hostName.json"

$registryPaths = @(
    "HKCU:\Software\Google\Chrome\NativeMessagingHosts\$hostName",
    "HKCU:\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\$hostName"
)
foreach ($registryPath in $registryPaths) {
    if (Test-Path -LiteralPath $registryPath) {
        $registryKey = Get-Item -LiteralPath $registryPath
        $registeredManifest = [string]$registryKey.GetValue("")
        if ([string]::Equals($registeredManifest, $hostManifestPath, [System.StringComparison]::OrdinalIgnoreCase)) {
            Remove-Item -LiteralPath $registryPath
        }
        else {
            Write-Warning "Kept Native Messaging registration owned by another manifest: $registryPath"
        }
    }
}

if (Test-Path -LiteralPath $hostManifestPath -PathType Leaf) {
    $ownedManifest = $false
    try {
        $manifest = Get-Content -LiteralPath $hostManifestPath -Raw -Encoding UTF8 | ConvertFrom-Json
        $origins = @($manifest.allowed_origins)
        $ownedManifest = $manifest.name -eq $hostName -and
            $manifest.type -eq "stdio" -and
            $origins.Count -eq 1 -and
            $origins[0] -eq "chrome-extension://ppncejmpiekmkepaeccdnpnpgdcfafje/"
    }
    catch {
        $ownedManifest = $false
    }
    if ($ownedManifest) {
        Remove-Item -LiteralPath $hostManifestPath
    }
    else {
        Write-Warning "Kept Native Messaging manifest because its ownership could not be verified: $hostManifestPath"
    }
}
if ((Test-Path -LiteralPath $targetDirectory -PathType Container) -and
    -not (Get-ChildItem -LiteralPath $targetDirectory -Force | Select-Object -First 1)) {
    Remove-Item -LiteralPath $targetDirectory
}

Write-Host "Removed Chrome/Brave Native Messaging host registration: $hostName"
Write-Host "The Content Blueprint executable and saved briefs/content packs were not deleted."
