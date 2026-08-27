Unicode true

; Content Blueprint's Windows distribution is deliberately per-user. Keep
; these defines before wails_tools.nsh so a manual makensis invocation has the
; same security boundary as `wails build -installscope user`.
!ifndef WAILS_INSTALL_SCOPE
  !define WAILS_INSTALL_SCOPE "user"
!endif
!ifndef REQUEST_EXECUTION_LEVEL
  !define REQUEST_EXECUTION_LEVEL "user"
!endif

!define INFO_PROJECTNAME "content-blueprint"
!define INFO_COMPANYNAME "Useless007"
!define INFO_PRODUCTNAME "Content Blueprint"
!define INFO_PRODUCTVERSION "0.3.0"
!define INFO_COPYRIGHT "Copyright (c) 2026 Useless007"
!define PRODUCT_EXECUTABLE "content-blueprint.exe"
!define UNINST_KEY_NAME "ContentBlueprint"

!define NATIVE_HOST_NAME "com.contentblueprint.facebook"
!define NATIVE_HOST_MANIFEST "com.contentblueprint.facebook.json"
!define NATIVE_HOST_RELATIVE_PATH "native-host\${NATIVE_HOST_MANIFEST}"
!define NATIVE_HOST_REG_CHROME "Software\Google\Chrome\NativeMessagingHosts\${NATIVE_HOST_NAME}"
!define NATIVE_HOST_REG_BRAVE "Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\${NATIVE_HOST_NAME}"

!include "wails_tools.nsh"

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion "${INFO_PRODUCTVERSION}.0"
VIAddVersionKey "CompanyName" "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion" "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion" "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright" "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName" "${INFO_PRODUCTNAME}"

ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Name "${INFO_PRODUCTNAME}"
BrandingText "${INFO_PRODUCTNAME} ${INFO_PRODUCTVERSION}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe"
InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
InstallDirRegKey HKCU "${UNINST_KEY}" "InstallLocation"
ShowInstDetails show
ShowUninstDetails show

Function .onInit
  !insertmacro wails.checkArchitecture
FunctionEnd

Section "Content Blueprint" SEC_MAIN
  !insertmacro wails.setShellContext
  !insertmacro wails.webview2runtime

  SetOverwrite on
  SetOutPath "$INSTDIR"
  !insertmacro wails.files
  File "/oname=INSTALLATION.txt" "resources\INSTALLATION.txt"

  ; The companion and its manifest live together. Chrome permits a relative
  ; native-host executable path on Windows, so the static manifest remains
  ; valid even when the user chooses a different installation directory.
  SetOutPath "$INSTDIR\native-host"
  File "/oname=content-blueprint-companion.exe" ".payload\content-blueprint-companion.exe"
  File "/oname=${NATIVE_HOST_MANIFEST}" "resources\native-host-manifest.json"

  ; Package only files used by the Manifest V3 extension at runtime. Tests,
  ; package manifests, documentation, and node_modules are intentionally absent.
  SetOutPath "$INSTDIR\facebook-extension"
  File "..\..\..\facebook-extension\manifest.json"
  File "..\..\..\facebook-extension\icon.png"
  File "..\..\..\facebook-extension\service-worker.js"
  File "..\..\..\facebook-extension\content-script.js"
  File "..\..\..\facebook-extension\sidepanel.html"
  File "..\..\..\facebook-extension\sidepanel.css"
  File "..\..\..\facebook-extension\sidepanel.js"
  SetOutPath "$INSTDIR\facebook-extension\src"
  File "..\..\..\facebook-extension\src\core.js"
  File "..\..\..\facebook-extension\src\growth.js"

  ; Native Messaging is registered only for the current user and only under
  ; the two browser-specific keys owned by this product.
  SetRegView 32
  WriteRegStr HKCU "${NATIVE_HOST_REG_CHROME}" "" "$INSTDIR\${NATIVE_HOST_RELATIVE_PATH}"
  WriteRegStr HKCU "${NATIVE_HOST_REG_BRAVE}" "" "$INSTDIR\${NATIVE_HOST_RELATIVE_PATH}"
  SetRegView 64
  WriteRegStr HKCU "${NATIVE_HOST_REG_CHROME}" "" "$INSTDIR\${NATIVE_HOST_RELATIVE_PATH}"
  WriteRegStr HKCU "${NATIVE_HOST_REG_BRAVE}" "" "$INSTDIR\${NATIVE_HOST_RELATIVE_PATH}"

  CreateDirectory "$SMPROGRAMS\${INFO_PRODUCTNAME}"
  CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
  CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}\Browser Extension Folder.lnk" "$WINDIR\explorer.exe" '$"$INSTDIR\facebook-extension$"'
  CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}\Uninstall.lnk" "$INSTDIR\uninstall.exe"
  CreateShortcut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

  !insertmacro wails.associateFiles
  !insertmacro wails.associateCustomProtocols
  !insertmacro wails.writeUninstaller

  WriteRegStr HKCU "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoRepair" 1
SectionEnd

Section "uninstall"
  !insertmacro wails.setShellContext
  ; Do not remove a native-host registration if another install has redirected
  ; it since this installer wrote it. Chrome checks both registry views on
  ; 64-bit Windows, so ownership is checked independently in each view.
  SetRegView 32
  ReadRegStr $0 HKCU "${NATIVE_HOST_REG_CHROME}" ""
  StrCmp $0 "$INSTDIR\${NATIVE_HOST_RELATIVE_PATH}" 0 chrome_32_registration_done
  DeleteRegKey HKCU "${NATIVE_HOST_REG_CHROME}"
chrome_32_registration_done:

  ReadRegStr $0 HKCU "${NATIVE_HOST_REG_BRAVE}" ""
  StrCmp $0 "$INSTDIR\${NATIVE_HOST_RELATIVE_PATH}" 0 brave_32_registration_done
  DeleteRegKey HKCU "${NATIVE_HOST_REG_BRAVE}"
brave_32_registration_done:

  SetRegView 64
  ReadRegStr $0 HKCU "${NATIVE_HOST_REG_CHROME}" ""
  StrCmp $0 "$INSTDIR\${NATIVE_HOST_RELATIVE_PATH}" 0 chrome_64_registration_done
  DeleteRegKey HKCU "${NATIVE_HOST_REG_CHROME}"
chrome_64_registration_done:

  ReadRegStr $0 HKCU "${NATIVE_HOST_REG_BRAVE}" ""
  StrCmp $0 "$INSTDIR\${NATIVE_HOST_RELATIVE_PATH}" 0 brave_64_registration_done
  DeleteRegKey HKCU "${NATIVE_HOST_REG_BRAVE}"
brave_64_registration_done:

  Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}\${INFO_PRODUCTNAME}.lnk"
  Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}\Browser Extension Folder.lnk"
  Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}\Uninstall.lnk"
  RMDir "$SMPROGRAMS\${INFO_PRODUCTNAME}"
  Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

  ; Delete only the exact payload files installed above. No recursive delete is
  ; used, and user-created files cause their containing directory to remain.
  Delete /REBOOTOK "$INSTDIR\facebook-extension\src\core.js"
  Delete /REBOOTOK "$INSTDIR\facebook-extension\src\growth.js"
  RMDir "$INSTDIR\facebook-extension\src"
  Delete /REBOOTOK "$INSTDIR\facebook-extension\manifest.json"
  Delete /REBOOTOK "$INSTDIR\facebook-extension\icon.png"
  Delete /REBOOTOK "$INSTDIR\facebook-extension\service-worker.js"
  Delete /REBOOTOK "$INSTDIR\facebook-extension\content-script.js"
  Delete /REBOOTOK "$INSTDIR\facebook-extension\sidepanel.html"
  Delete /REBOOTOK "$INSTDIR\facebook-extension\sidepanel.css"
  Delete /REBOOTOK "$INSTDIR\facebook-extension\sidepanel.js"
  RMDir "$INSTDIR\facebook-extension"

  Delete /REBOOTOK "$INSTDIR\native-host\${NATIVE_HOST_MANIFEST}"
  Delete /REBOOTOK "$INSTDIR\native-host\content-blueprint-companion.exe"
  RMDir "$INSTDIR\native-host"
  Delete /REBOOTOK "$INSTDIR\INSTALLATION.txt"
  Delete /REBOOTOK "$INSTDIR\${PRODUCT_EXECUTABLE}"

  !insertmacro wails.unassociateFiles
  !insertmacro wails.unassociateCustomProtocols
  !insertmacro wails.deleteUninstaller
  RMDir "$INSTDIR"

  ; Intentionally preserved:
  ; - $AppData\ContentBlueprint (settings, saved Briefs, Content Packs)
  ; - browser extension storage
  ; - Claude Code and Codex MCP configuration
SectionEnd
