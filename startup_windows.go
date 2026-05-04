//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	startupTaskName        = "LanFlow"
	startupTaskNameLegacy  = "LanTrans" // removed on install if it exists
)

func installStartup() error {
	// Clean up legacy startup task from the old name to prevent port conflicts.
	removeLegacyStartup()

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	taskRun := fmt.Sprintf(
		`powershell.exe -NoProfile -WindowStyle Hidden -Command "Start-Process -FilePath '%s' -ArgumentList 'serve' -WindowStyle Hidden"`,
		escapePowerShellSingleQuoted(exe),
	)
	cmd := exec.Command("schtasks", "/Create", "/TN", startupTaskName, "/SC", "ONLOGON", "/TR", taskRun, "/RL", "LIMITED", "/F")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks create failed: %w: %s", err, string(out))
	}
	return nil
}

func removeLegacyStartup() {
	cmd := exec.Command("schtasks", "/Delete", "/TN", startupTaskNameLegacy, "/F")
	if out, err := cmd.CombinedOutput(); err == nil {
		fmt.Printf("Removed legacy startup task '%s'\n", startupTaskNameLegacy)
	} else {
		// silently ignored if the task didn't exist
		_ = out
	}
}

func escapePowerShellSingleQuoted(value string) string {
	return strings.ReplaceAll(value, `'`, `''`)
}

func uninstallStartup() error {
	cmd := exec.Command("schtasks", "/Delete", "/TN", startupTaskName, "/F")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks delete failed: %w: %s", err, string(out))
	}
	return nil
}
