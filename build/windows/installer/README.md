# Windows per-user installer

The NSIS package installs Content Blueprint without elevation under:

```text
%LOCALAPPDATA%\Programs\Content Blueprint
```

It contains the Wails application, `native-host\content-blueprint-companion.exe`,
and only the runtime files needed by the unpacked Manifest V3 extension,
including its icon and the Growth Pack validator/renderer.
`node_modules`, tests, package manifests, and development documentation are not
packaged.

## Build

From the project root:

```powershell
.\build\windows\build-release.ps1
```

The release script validates both manifests, runs Go and extension tests, builds
the companion payload, then asks Wails to compile the amd64 application and NSIS
installer with user install scope. Use `-SkipTests` only after the same revision
has already passed the test suites. `makensis.exe` must be on `PATH`, in the
standard NSIS installation directory, or supplied explicitly:

```powershell
.\build\windows\build-release.ps1 -MakeNSISPath C:\tools\nsis\Bin\makensis.exe
```

Outputs are written to `build\bin`:

- `content-blueprint-amd64-installer.exe`
- `content-blueprint.exe`
- `content-blueprint-companion.exe`
- `SHA256SUMS.txt`

Building the package does not execute it and does not change the registry.
The current public artifact is not code-signed, so Windows SmartScreen may show
an unknown-publisher warning. SHA-256 verification is documented for this release;
Authenticode signing through NSIS `!finalize`/`!uninstfinalize` is recommended for
a future release once a project signing certificate is available.

## Install and uninstall boundary

Installation writes only these Native Messaging keys in `HKCU` (in both the
32-bit and 64-bit registry views so either browser architecture can discover the
host):

```text
Software\Google\Chrome\NativeMessagingHosts\com.contentblueprint.facebook
Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\com.contentblueprint.facebook
```

Both point to the installed manifest, whose sole allowed origin is:

```text
chrome-extension://ppncejmpiekmkepaeccdnpnpgdcfafje/
```

The uninstaller removes a browser registry key only when its current value still
points to that exact installation. It deletes exact packaged files and then
removes only empty package directories; it never recursively removes the install
directory. Saved Briefs, Content Packs, settings, browser storage, and Claude or
Codex MCP configuration are preserved. MCP registration remains an explicit,
manual user action.

When `7z` or NanaZip is available, the release script also inspects the completed
NSIS archive and fails if any required runtime file is missing or a development
path such as `node_modules`, `tests`, or `package-lock.json` was included.
