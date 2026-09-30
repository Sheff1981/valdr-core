Unicode true

!include "MUI2.nsh"
!include "x64.nsh"

!ifndef APP_VERSION
  !error "APP_VERSION define is required"
!endif
!ifndef APP_FILE_VERSION
  !error "APP_FILE_VERSION define is required"
!endif
!ifndef OUTFILE
  !error "OUTFILE define is required"
!endif
!ifndef DESKTOP_EXE
  !error "DESKTOP_EXE define is required"
!endif
!ifndef NODE_EXE
  !error "NODE_EXE define is required"
!endif
!ifndef MINER_EXE
  !error "MINER_EXE define is required"
!endif

Name "VALDR Desktop"
OutFile "${OUTFILE}"
InstallDir "$LOCALAPPDATA\Programs\VALDR Desktop"
RequestExecutionLevel admin
SetCompressor /SOLID lzma

VIProductVersion "${APP_FILE_VERSION}"
VIFileVersion "${APP_FILE_VERSION}"
VIAddVersionKey "CompanyName" "VALDR"
VIAddVersionKey "FileDescription" "VALDR Desktop Installer"
VIAddVersionKey "ProductName" "VALDR Desktop"
VIAddVersionKey "ProductVersion" "${APP_VERSION}"
VIAddVersionKey "FileVersion" "${APP_VERSION}"

!define MUI_ABORTWARNING

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "VALDR Desktop" SecMain
  SetShellVarContext current
  SetOutPath "$INSTDIR"

  ; Upgrade safety: do not overwrite running binaries or force-kill valdrd.
  nsExec::ExecToStack 'tasklist /FI "IMAGENAME eq VALDR.exe" /NH'
  Pop $0
  Pop $1
  ${If} $1 != ""
    ${If} $1 != "INFO: No tasks are running which match the specified criteria."
      MessageBox MB_ICONEXCLAMATION|MB_OK "VALDR Desktop is still running. Close VALDR Desktop completely, then run this installer again. Wallet and blockchain data will be preserved."
      Abort
    ${EndIf}
  ${EndIf}

  nsExec::ExecToStack 'tasklist /FI "IMAGENAME eq valdrd.exe" /NH'
  Pop $0
  Pop $1
  ${If} $1 != ""
    ${If} $1 != "INFO: No tasks are running which match the specified criteria."
      MessageBox MB_ICONEXCLAMATION|MB_OK "The VALDR node is still shutting down. Wait a few seconds, then run this installer again. Do not force-close valdrd.exe because it may be writing blockchain data."
      Abort
    ${EndIf}
  ${EndIf}

  nsExec::ExecToStack 'tasklist /FI "IMAGENAME eq valdr-miner.exe" /NH'
  Pop $0
  Pop $1
  ${If} $1 != ""
    ${If} $1 != "INFO: No tasks are running which match the specified criteria."
      MessageBox MB_ICONEXCLAMATION|MB_OK "VALDR Miner is still running. Stop mining and close VALDR Desktop, then run this installer again."
      Abort
    ${EndIf}
  ${EndIf}

  File /oname=VALDR.exe "${DESKTOP_EXE}"
  File /oname=valdrd.exe "${NODE_EXE}"
  File /oname=valdr-miner.exe "${MINER_EXE}"

  ; VALDR Testnet2 P2P must work after installation without PowerShell.
  ; Allow only the bundled node executable on the native P2P TCP port.
  nsExec::ExecToLog 'netsh advfirewall firewall delete rule name="VALDR Testnet2 P2P" program="$INSTDIR\valdrd.exe"'
  nsExec::ExecToLog 'netsh advfirewall firewall add rule name="VALDR Testnet2 P2P" dir=in action=allow protocol=TCP localport=17333 program="$INSTDIR\valdrd.exe" profile=any enable=yes'

  WriteUninstaller "$INSTDIR\Uninstall.exe"

  CreateDirectory "$SMPROGRAMS\VALDR"
  CreateShortcut "$SMPROGRAMS\VALDR\VALDR Desktop.lnk" "$INSTDIR\VALDR.exe"
  CreateShortcut "$DESKTOP\VALDR Desktop.lnk" "$INSTDIR\VALDR.exe"

  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\VALDRDesktop" "DisplayName" "VALDR Desktop"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\VALDRDesktop" "DisplayVersion" "${APP_VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\VALDRDesktop" "Publisher" "VALDR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\VALDRDesktop" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\VALDRDesktop" "DisplayIcon" "$INSTDIR\VALDR.exe"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\VALDRDesktop" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\VALDRDesktop" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\VALDRDesktop" "NoRepair" 1
SectionEnd

Section "Uninstall"
  SetShellVarContext current

  Delete "$DESKTOP\VALDR Desktop.lnk"
  Delete "$SMPROGRAMS\VALDR\VALDR Desktop.lnk"
  RMDir "$SMPROGRAMS\VALDR"

  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\VALDRDesktop"

  ; Remove the inbound rule owned by this installation.
  nsExec::ExecToLog 'netsh advfirewall firewall delete rule name="VALDR Testnet2 P2P" program="$INSTDIR\valdrd.exe"'

  ; Remove only files owned by VALDR. Never recursively delete a user-selected
  ; install directory because it may contain unrelated user files.
  Delete "$INSTDIR\VALDR.exe"
  Delete "$INSTDIR\valdrd.exe"
  Delete "$INSTDIR\valdr-miner.exe"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"

  ; User data lives outside the program directory and is intentionally preserved.
SectionEnd
