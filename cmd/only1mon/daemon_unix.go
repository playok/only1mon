//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/playok/only1mon/internal/config"
)

var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

func cmdStart() {
	cfg := config.Load()

	// Check if already running
	if pid, err := readPidFile(cfg.PidFile); err == nil {
		if processExists(pid) {
			fmt.Printf("only1mon is already running (PID %d)\n", pid)
			os.Exit(1)
		}
		// Stale PID file
		os.Remove(cfg.PidFile)
	}

	// Build args: replace "start" with "run" for the child
	childArgs := []string{"run"}
	childArgs = append(childArgs, buildForwardFlags(cfg)...)

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to find executable: %v\n", err)
		os.Exit(1)
	}

	// Open log file
	logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file %s: %v\n", cfg.LogFile, err)
		os.Exit(1)
	}

	child := &exec.Cmd{
		Path:   exe,
		Args:   append([]string{filepath.Base(exe)}, childArgs...),
		Stdout: logFile,
		Stderr: logFile,
		SysProcAttr: &syscall.SysProcAttr{
			Setsid: true, // detach from terminal
		},
	}

	if err := child.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start daemon: %v\n", err)
		os.Exit(1)
	}

	pid := child.Process.Pid
	if err := writePidFile(cfg.PidFile, pid); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write PID file: %v\n", err)
	}

	// Release the child — parent exits
	child.Process.Release()
	logFile.Close()

	fmt.Printf("only1mon started (PID %d)\n", pid)
	fmt.Printf("  Listen : http://%s\n", cfg.Listen)
	fmt.Printf("  Base   : %s\n", cfg.BasePath)
	fmt.Printf("  Config : %s\n", cfg.ConfigPath)
	fmt.Printf("  PID    : %s\n", cfg.PidFile)
	fmt.Printf("  Log    : %s\n", cfg.LogFile)
}

func cmdStop() {
	cfg := config.Load()

	pid, err := readPidFile(cfg.PidFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "only1mon is not running (no PID file: %s)\n", cfg.PidFile)
		os.Exit(1)
	}

	if !processExists(pid) {
		fmt.Printf("only1mon is not running (stale PID %d)\n", pid)
		os.Remove(cfg.PidFile)
		os.Exit(1)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to find process %d: %v\n", pid, err)
		os.Exit(1)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		fmt.Fprintf(os.Stderr, "failed to stop PID %d: %v\n", pid, err)
		os.Exit(1)
	}

	// Wait for process to exit (up to 10 seconds)
	for i := 0; i < 100; i++ {
		time.Sleep(100 * time.Millisecond)
		if !processExists(pid) {
			os.Remove(cfg.PidFile)
			fmt.Printf("only1mon stopped (PID %d)\n", pid)
			return
		}
	}

	fmt.Printf("only1mon stop signal sent (PID %d), waiting for exit...\n", pid)
	os.Remove(cfg.PidFile)
}

func cmdStatus() {
	cfg := config.Load()

	pid, err := readPidFile(cfg.PidFile)
	if err != nil {
		fmt.Println("only1mon is stopped")
		os.Exit(1)
	}

	if processExists(pid) {
		fmt.Printf("only1mon is running (PID %d)\n", pid)
		fmt.Printf("  Listen : http://%s\n", cfg.Listen)
		fmt.Printf("  Base   : %s\n", cfg.BasePath)
		fmt.Printf("  Config : %s\n", cfg.ConfigPath)
		fmt.Printf("  PID    : %s\n", cfg.PidFile)
		fmt.Printf("  Log    : %s\n", cfg.LogFile)
	} else {
		fmt.Printf("only1mon is stopped (stale PID file, was PID %d)\n", pid)
		os.Remove(cfg.PidFile)
		os.Exit(1)
	}
}

func processExists(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks existence without actually sending a signal
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
