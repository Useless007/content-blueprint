[CmdletBinding()]
param(
    [ValidateSet("Codex", "Claude", "Both")]
    [string]$Target = "Both",
    [string]$CompanionExecutable = ""
)

$ErrorActionPreference = "Stop"
$serverName = "content-blueprint-facebook"
if ([string]::IsNullOrWhiteSpace($CompanionExecutable)) {
    $CompanionExecutable = Join-Path $PSScriptRoot "..\..\build\bin\content-blueprint-companion.exe"
}
$resolvedExecutable = (Resolve-Path -LiteralPath $CompanionExecutable).Path
if (-not [System.IO.File]::Exists($resolvedExecutable)) {
    throw "Companion executable was not found: $resolvedExecutable"
}

if ($Target -in @("Codex", "Both") -and -not (Get-Command codex -ErrorAction SilentlyContinue)) {
    throw "Codex CLI is not installed or not available on PATH."
}
if ($Target -in @("Claude", "Both") -and -not (Get-Command claude -ErrorAction SilentlyContinue)) {
    throw "Claude Code is not installed or not available on PATH."
}

if ($Target -in @("Codex", "Both")) {
    & codex mcp add $serverName -- $resolvedExecutable
    if ($LASTEXITCODE -ne 0) {
        throw "Codex MCP registration failed."
    }
    Write-Host "Registered $serverName with Codex."
}

if ($Target -in @("Claude", "Both")) {
    & claude mcp add --transport stdio --scope user $serverName -- $resolvedExecutable
    if ($LASTEXITCODE -ne 0) {
        throw "Claude MCP registration failed."
    }
    Write-Host "Registered $serverName with Claude Code at user scope."
}

Write-Host "Use /mcp inside Claude Code or Codex to verify the connection."
