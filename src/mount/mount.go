// SPDX-FileCopyrightText: 2026 AnmiTaliDev <anmitalidev@nuros.org>
// SPDX-FileCopyrightText: 2026 CosmoBlade <C0sm0B14d3@proton.me>
// SPDX-License-Identifier: GPL-3.0-only

package mount

import (
	"fmt"
	"os"
	"os/exec"
)

// defaultRunCommand executes a system command.
func defaultRunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// MountProcWith creates procPath inside the current root and mounts the proc
// filesystem. runCmd may be nil, in which case the real default runner is used.
// This function is designed to be testable without touching the real /proc.
func MountProcWith(procPath string, runCmd func(name string, args ...string) error) error {
	if runCmd == nil {
		runCmd = defaultRunCommand
	}
	if err := os.MkdirAll(procPath, 0555); err != nil {
		return fmt.Errorf("cannot create %s: %v", procPath, err)
	}
	if err := runCmd("mount", "-t", "proc", "proc", procPath); err != nil {
		return fmt.Errorf("cannot mount proc: %v", err)
	}
	return nil
}

// MountProc is the convenience wrapper used by consumers. It mounts to /proc
// using the real command runner.
func MountProc() error {
	return MountProcWith("/proc", defaultRunCommand)
}
