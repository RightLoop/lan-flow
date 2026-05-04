//go:build !windows

package main

import "errors"

func installStartup() error {
	return errors.New("install-startup is only supported on Windows")
}

func uninstallStartup() error {
	return errors.New("uninstall-startup is only supported on Windows")
}

