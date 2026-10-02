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
  ; Use the command exit code rather than comparing localized tasklist text.
  nsExec::ExecToStack 'cmd /C tasklist /FI "IMAGENAME eq VALDR.exe" /NH ^| find /I "VALDR.exe" ^>nul'
  Pop $0
  Pop $1
  ${If} $0 == "0"
    IfSilent valdr_running_silent
    MessageBox MB_ICONEXCLAMATION|MB_OK "VALDR Desktop is still running. Close VALDR Desktop completely, then run this installer again. Wallet and blockchain data will be preserved."
valdr_running_silent:
    Abort
  ${EndIf}

  nsExec::ExecToStack 'cmd /C tasklist /FI "IMAGENAME eq valdrd.exe" /NH ^| find /I "valdrd.exe" ^>nul'
  Pop $0
  Pop $1
  ${If} $0 == "0"
    IfSilent valdrd_running_silent
    MessageBox MB_ICONEXCLAMATION|MB_OK "The VALDR node is still shutting down. Wait a few seconds, then run this installer again. Do not force-close valdrd.exe because it may be writing blockchain data."
valdrd_running_silent:
    Abort
  ${EndIf}

  nsExec::ExecToStack 'cmd /C tasklist /FI "IMAGENAME eq valdr-miner.exe" /NH ^| find /I "valdr-miner.exe" ^>nul'
  Pop $0
  Pop $1
  ${If} $0 == "0"
    IfSilent miner_running_silent
    MessageBox MB_ICONEXCLAMATION|MB_OK "VALDR Miner is still running. Stop mining and close VALDR Desktop, then run this installer again."
miner_running_silent:
    Abort
  ${EndIf}

  File /oname=VALDR.exe "${DESKTOP_EXE}"
  File /oname=valdrd.exe "${NODE_EXE}"
  File /oname=valdr-miner.exe "${MINER_EXE}"

  ; VALDR Testnet2 P2P must be allowed through Windows Defender Firewall.
  ; Bitcoin Core normally causes Windows to offer a firewall prompt on first
  ; listen. VALDR installs the equivalent inbound exception explicitly so
  ; ordinary users do not have to notice or correctly answer that prompt.
  ;
  ; The actual network listener is valdrd.exe, not VALDR.exe, therefore the
  ; firewall rule must be bound to the bundled node process itself.
  nsExec::ExecToStack 'netsh advfirewall firewall delete rule name="VALDR Testnet2 P2P" program="$INSTDIR\valdrd.exe"'
  Pop $0
  Pop $1

  nsExec::ExecToStack 'netsh advfirewall firewall add rule name="VALDR Testnet2 P2P" dir=in action=allow protocol=TCP localport=17333 program="$INSTDIR\valdrd.exe" profile=private,public enable=yes'
  Pop $0
  Pop $1
  ${If} $0 != "0"
    ; GitHub-hosted Windows CI runners do not provide the same firewall service
    ; behavior as an ordinary Windows installation. Bypass only in that known
    ; CI environment; real user installations still fail closed.
    ReadEnvStr $R1 "GITHUB_ACTIONS"
    ${If} $R1 != "true"
      IfSilent firewall_failed_silent
      MessageBox MB_ICONSTOP|MB_OK "VALDR could not create its Windows Firewall rule for TCP port 17333.$\r$\n$\r$\nWithout this rule the node may be unable to accept inbound peers.$\r$\n$\r$\nInstallation will stop so this networking problem is not hidden."
firewall_failed_silent:
      Abort
    ${EndIf}
  ${EndIf}

  ; Provider-hosted public nodes may intentionally listen on a mapped guest
  ; port that differs from the Testnet2 default. Keep the ordinary 17333 rule
  ; narrow and add a second program-bound rule only for an explicit operator
  ; VALDR_DESKTOP_P2P_PORT override.
  ReadEnvStr $R2 "VALDR_DESKTOP_P2P_PORT"
  ${If} $R2 == ""
    ReadRegStr $R2 HKCU "Environment" "VALDR_DESKTOP_P2P_PORT"
  ${EndIf}
  ${If} $R2 != ""
    ${If} $R2 != "17333"
      nsExec::ExecToStack 'netsh advfirewall firewall delete rule name="VALDR Testnet2 P2P override" program="$INSTDIR\valdrd.exe"'
      Pop $0
      Pop $1
      nsExec::ExecToStack 'netsh advfirewall firewall add rule name="VALDR Testnet2 P2P override" dir=in action=allow protocol=TCP localport=$R2 program="$INSTDIR\valdrd.exe" profile=private,public enable=yes'
      Pop $0
      Pop $1
      ${If} $0 != "0"
        ReadEnvStr $R1 "GITHUB_ACTIONS"
        ${If} $R1 != "true"
          IfSilent firewall_override_failed_silent
          MessageBox MB_ICONSTOP|MB_OK "VALDR could not create its Windows Firewall rule for the configured P2P port $R2.$\r$\n$\r$\nThe public node would not be reliably reachable, so installation will stop."
firewall_override_failed_silent:
          Abort
        ${EndIf}
      ${EndIf}
    ${EndIf}
  ${EndIf}

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

  ; Remove the inbound rules owned by this installation.
  nsExec::ExecToLog 'netsh advfirewall firewall delete rule name="VALDR Testnet2 P2P" program="$INSTDIR\valdrd.exe"'
  nsExec::ExecToLog 'netsh advfirewall firewall delete rule name="VALDR Testnet2 P2P override" program="$INSTDIR\valdrd.exe"'

  ; Remove only files owned by VALDR. Never recursively delete a user-selected
  ; install directory because it may contain unrelated user files.
  Delete "$INSTDIR\VALDR.exe"
  Delete "$INSTDIR\valdrd.exe"
  Delete "$INSTDIR\valdr-miner.exe"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"

  ; User data lives outside the program directory and is intentionally preserved.
SectionEnd
