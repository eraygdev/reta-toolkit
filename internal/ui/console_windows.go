//go:build windows

package ui

import (
	"os"

	"golang.org/x/sys/windows"
)

const cpUTF8 = 65001

func init() {
	_ = windows.SetConsoleOutputCP(cpUTF8)
	_ = windows.SetConsoleCP(cpUTF8)

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
