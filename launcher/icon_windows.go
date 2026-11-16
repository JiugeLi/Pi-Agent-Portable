package main

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var user32 = syscall.NewLazyDLL("user32.dll")

func needsIconConsole(owned, visible, alreadyRelaunched bool, argumentCount int) bool {
	return owned && !visible && !alreadyRelaunched && argumentCount == 0
}

func relaunchIconConsole() (bool, error) {
	window, _, _ := kernel32.NewProc("GetConsoleWindow").Call()
	visible, _, _ := user32.NewProc("IsWindowVisible").Call(window)
	if !needsIconConsole(ownsConsole(), visible != 0, os.Getenv("PI_PORTABLE_ICON_CONSOLE") == "1", len(os.Args)-1) {
		return false, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return false, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	host := filepath.Join(os.Getenv("SystemRoot"), "System32", "conhost.exe")
	// An explicit inbox console host avoids Windows Terminal's taskbar identity.
	// This is per launch; no terminal profiles or default-app settings change.
	process, err := startIconConsole(host, executable, cwd, nil, append(os.Environ(), "PI_PORTABLE_ICON_CONSOLE=1"), false)
	if err != nil {
		return false, err
	}
	syscall.CloseHandle(process)
	return true, nil
}

func startIconConsole(host, executable, cwd string, args, env []string, hidden bool) (syscall.Handle, error) {
	// Do not use exec.Cmd's default NUL handles: conhost interprets valid std
	// handles as ConPTY streams. NULL handles request its own console window.
	argv := append([]string{host, "-ForceNoHandoff", "--", executable}, args...)
	_, process, err := syscall.StartProcess(host, argv, &syscall.ProcAttr{
		Dir: cwd, Env: env, Files: []uintptr{0, 0, 0},
		Sys: &syscall.SysProcAttr{CreationFlags: 0x8 | 0x200, NoInheritHandles: true, HideWindow: hidden},
	})
	return syscall.Handle(process), err
}

func initializeConsoleIcon() {
	if ownsConsole() {
		setupConsoleIcon()
	}
}

func loadProgramIcon(size int) uintptr {
	module, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	// rsrc's first icon group is resource 1. LR_SHARED keeps its handle valid
	// while the launcher waits for PowerShell and Pi to finish.
	icon, _, _ := user32.NewProc("LoadImageW").Call(module, 1, 1, uintptr(size), uintptr(size), 0x8000)
	return icon
}

func applyWindowIcons(window, small, large uintptr) bool {
	if window == 0 || small == 0 || large == 0 {
		return false
	}
	var result uintptr
	// Use a deadline so an unresponsive console cannot block Pi startup.
	for _, entry := range [][2]uintptr{{0, small}, {1, large}} {
		ok, _, _ := user32.NewProc("SendMessageTimeoutW").Call(window, 0x80, entry[0], entry[1], 2, 500, uintptr(unsafe.Pointer(&result)))
		if ok == 0 {
			return false
		}
	}
	return true
}

func setupConsoleIcon() {
	window, _, _ := kernel32.NewProc("GetConsoleWindow").Call()
	visible, _, _ := user32.NewProc("IsWindowVisible").Call(window)
	if visible == 0 {
		// ConPTY/Windows Terminal owns its visible host window. Do not change an
		// unrelated terminal window, its profile, or any pinned taskbar shortcut.
		return
	}
	applyWindowIcons(window, loadProgramIcon(16), loadProgramIcon(32))
}
