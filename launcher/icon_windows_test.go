package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func TestPiIconResourcesAndWindowAssignment(t *testing.T) {
	small, large := loadProgramIcon(16), loadProgramIcon(32)
	if small == 0 || large == 0 {
		t.Fatal("Pi icon is missing from the executable resources")
	}
	class, _ := syscall.UTF16PtrFromString("STATIC")
	// A private, invisible test window: never modify the user's terminal.
	window, _, err := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), 0, 0x80000000,
		0, 0, 32, 32, 0, 0, 0, 0)
	if window == 0 {
		t.Fatal("Cannot create icon test window:", err)
	}
	defer user32.NewProc("DestroyWindow").Call(window)
	if !applyWindowIcons(window, small, large) {
		t.Fatal("Cannot apply Pi window icons")
	}
	for _, entry := range [][2]uintptr{{0, small}, {1, large}} {
		actual, _, _ := user32.NewProc("SendMessageW").Call(window, 0x7f, entry[0], 0)
		if actual != entry[1] {
			t.Fatal("Window did not retain its Pi icon")
		}
	}
}

func TestDedicatedIconConsoleOnlyForDirectTerminalHandoff(t *testing.T) {
	if !needsIconConsole(true, false, false, 0) {
		t.Fatal("direct terminal handoff needs its own Pi console")
	}
	for _, state := range []struct {
		owned, visible, guarded bool
		args                    int
	}{
		{false, false, false, 0}, {true, true, false, 0}, {true, false, true, 0}, {true, false, false, 1},
	} {
		if needsIconConsole(state.owned, state.visible, state.guarded, state.args) {
			t.Fatal("must preserve shared terminals, existing consoles, CLI output and the loop guard")
		}
	}
}

func TestInboxConsoleHostStartsClientWithUnicodeCwd(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Pi 中文 project")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	report := filepath.Join(dir, "host-check.json")
	deadline, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	host := filepath.Join(os.Getenv("SystemRoot"), "System32", "conhost.exe")
	// Exercise the real host/parser without opening a visible terminal.
	cmd := exec.CommandContext(deadline, host, "-ForceNoHandoff", "--headless", "--", os.Args[0], "-test.run=^TestInboxConsoleHostClientHelper$")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PI_ICON_HOST_REPORT="+report, "PI_PORTABLE_ICON_CONSOLE=1")
	// Headless conhost requires VT pipes rather than NUL standard handles.
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Run(); err != nil {
		t.Fatal("Inbox console host could not start its client:", err)
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal("Console host did not run its client:", err)
	}
	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result["cwd"] != dir || result["guard"] != "1" {
		t.Fatal("Console host changed cwd/environment")
	}
}

func TestInboxConsoleHostClientHelper(t *testing.T) {
	report := os.Getenv("PI_ICON_HOST_REPORT")
	if report == "" {
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]string{"cwd": cwd, "guard": os.Getenv("PI_PORTABLE_ICON_CONSOLE")}
	if os.Getenv("PI_ICON_CLASSIC_CHECK") == "1" {
		window, _, _ := kernel32.NewProc("GetConsoleWindow").Call()
		style, _, _ := user32.NewProc("GetWindowLongW").Call(window, ^uintptr(15))
		visible, _, _ := user32.NewProc("IsWindowVisible").Call(window)
		if style&0x00c00000 == 0x00c00000 && visible == 0 && ownsConsole() && applyWindowIcons(window, loadProgramIcon(16), loadProgramIcon(32)) {
			result["privateClassicWindow"] = "true"
		}
	}
	data, _ := json.Marshal(result)
	if err := os.WriteFile(report, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestPrivateIconConsoleUsesARealWindowWithoutTerminalHandoff(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Pi 中文 window")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	report := filepath.Join(dir, "private-console.json")
	host := filepath.Join(os.Getenv("SystemRoot"), "System32", "conhost.exe")
	env := append(os.Environ(), "PI_ICON_HOST_REPORT="+report, "PI_PORTABLE_ICON_CONSOLE=1", "PI_ICON_CLASSIC_CHECK=1")
	process, err := startIconConsole(host, os.Args[0], dir, []string{"-test.run=^TestInboxConsoleHostClientHelper$"}, env, true)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(process)
	defer func() {
		if status, _ := syscall.WaitForSingleObject(process, 5000); status == syscall.WAIT_TIMEOUT {
			syscall.TerminateProcess(process, 1)
		}
	}()
	deadline := time.Now().Add(10 * time.Second)
	var data []byte
	for time.Now().Before(deadline) {
		data, err = os.ReadFile(report)
		if err == nil && len(data) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatal("Private console did not run its client:", err)
	}
	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result["cwd"] != dir || result["guard"] != "1" || result["privateClassicWindow"] != "true" {
		t.Fatal("Own-window startup did not preserve cwd, environment or Pi icons:", result)
	}
}
