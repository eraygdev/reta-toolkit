//go:build windows

package ui

import (
	"os"

	"golang.org/x/sys/windows"
)

func init() {
	_ = windows.SetConsoleOutputCP(windows.CP_UTF8)
	_ = windows.SetConsoleCP(windows.CP_UTF8)

	stdout := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(stdout, &mode); err == nil {
		_ = windows.SetConsoleMode(stdout, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	}

	stderr := windows.Handle(os.Stderr.Fd())
	if err := windows.GetConsoleMode(stderr, &mode); err == nil {
		_ = windows.SetConsoleMode(stderr, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	}
}
