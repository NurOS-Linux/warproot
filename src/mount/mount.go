// SPDX-FileCopyrightText: 2026 AnmiTaliDev <anmitalidev@nuros.org>
// SPDX-FileCopyrightText: 2026 CosmoBlade <ilovesmetana777@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package mount

import (
	"fmt"
	"os"
	"os/exec"
)

// MountProc creates /proc inside the current root and mounts the proc filesystem.
// Must be called after chroot.
func MountProc() error {
	const procPath = "/proc"
	if err := os.MkdirAll(procPath, 0555); err != nil {
		return fmt.Errorf("cannot create /proc: %v", err)
	}
	if err := runCommand("mount", "-t", "proc", "proc", procPath); err != nil {
		return fmt.Errorf("cannot mount proc: %v", err)
	}
	return nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
