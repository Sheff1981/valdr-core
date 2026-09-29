//go:build !windows

package desktop

import "os/exec"

func configureManagedChildProcess(_ *exec.Cmd) {}
