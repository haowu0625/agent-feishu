//go:build !windows

package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
)

func runNativeGUI(args []string, stdout io.Writer) error {
	if len(args) > 0 {
		fmt.Fprintln(stdout, "Agent Feishu native window is currently available on Windows.")
	}
	fmt.Fprintln(stdout, "Agent Feishu setup will run in this terminal.")
	if runtime.GOOS == "darwin" {
		fmt.Fprintln(stdout, "On macOS, scan the QR code printed below with the Feishu mobile app.")
	}
	return runSetup(nil, os.Stdin, stdout)
}
