//go:build !windows

package main

import "os/exec"

func interactiveCommand(cmd *exec.Cmd) *exec.Cmd { return cmd }

func initializeConsoleIcon() {}

func relaunchIconConsole() (bool, error) { return false, nil }
