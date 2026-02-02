//go:build windows

package main

import (
	"fmt"
	"os"
)

var shutdownSignals = []os.Signal{os.Interrupt}

func cmdStart() {
	fmt.Fprintln(os.Stderr, "daemon mode is not supported on Windows. Use 'run' for foreground execution.")
	os.Exit(1)
}

func cmdStop() {
	fmt.Fprintln(os.Stderr, "daemon mode is not supported on Windows.")
	os.Exit(1)
}

func cmdStatus() {
	fmt.Fprintln(os.Stderr, "daemon mode is not supported on Windows.")
	os.Exit(1)
}

func processExists(pid int) bool {
	return false
}
