//go:build windows

package desktop

import (
	"os/exec"
	"testing"
)

func TestConfigureManagedChildProcessHidesWindowsConsole(t *testing.T) {
	cmd := exec.Command("cmd.exe", "/c", "exit", "0")
	configureManagedChildProcess(cmd)
	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr is nil")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("managed child process does not hide its window")
	}
	if cmd.SysProcAttr.CreationFlags&windowsCreateNoWindow == 0 {
		t.Fatalf("CreationFlags=%#x missing CREATE_NO_WINDOW", cmd.SysProcAttr.CreationFlags)
	}
}
