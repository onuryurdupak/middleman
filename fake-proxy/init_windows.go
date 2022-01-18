//go:build windows

package main

import (
	"golang.org/x/sys/windows"
	"os"
)

func init() {
	stdout := windows.Handle(os.Stdout.Fd())
	var originalMode uint32

	_ = windows.GetConsoleMode(stdout, &originalMode)
	_ = windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
