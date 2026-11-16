package main

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

// Only Explorer's new, private console belongs to this launcher. Existing
// PowerShell/cmd/SSH terminals and redirected CLI/RPC streams stay untouched.
func ownsConsole() bool {
	var mode uint32
	for _, file := range []*os.File{os.Stdin, os.Stdout, os.Stderr} {
		ok, _, _ := kernel32.NewProc("GetConsoleMode").Call(file.Fd(), uintptr(unsafe.Pointer(&mode)))
		if ok == 0 {
			return false
		}
	}
	var processes [2]uint32
	count, _, _ := kernel32.NewProc("GetConsoleProcessList").Call(uintptr(unsafe.Pointer(&processes[0])), 2)
	return count == 1 && processes[0] == uint32(os.Getpid())
}

func interactiveCommand(cmd *exec.Cmd) *exec.Cmd {
	owned := ownsConsole()
	if owned {
		setupConsoleIcon()
	}
	if len(os.Args) != 1 || !owned {
		return cmd
	}
	powershell := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	if _, err := os.Stat(powershell); err != nil {
		return cmd
	}
	setupConsoleFont()
	return powerShellCommand(cmd, powershell)
}

func powerShellCommand(cmd *exec.Cmd, powershell string) *exec.Cmd {
	// Use data, never PowerShell source interpolation, for paths and arguments.
	launch, err := json.Marshal(struct {
		Node string   `json:"node"`
		Args []string `json:"args"`
	}{cmd.Path, cmd.Args[1:]})
	if err != nil {
		return cmd
	}
	// Do not run a profile, alter persistent defaults, change cwd, or relax policy.
	script := `$ErrorActionPreference = 'Stop'
$launch = ConvertFrom-Json $env:PI_PORTABLE_POWERSHELL_LAUNCH
Remove-Item Env:\PI_PORTABLE_POWERSHELL_LAUNCH
try {
  [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false)
  [Console]::InputEncoding = New-Object System.Text.UTF8Encoding($false)
  $ui = $Host.UI.RawUI
  $ui.WindowTitle = 'Pi Agent Portable - PowerShell'
  $ui.BackgroundColor = 'Black'
  $ui.ForegroundColor = 'Gray'
  $width = [Math]::Min(120, $ui.MaxPhysicalWindowSize.Width)
  $height = [Math]::Min(36, $ui.MaxPhysicalWindowSize.Height)
  $buffer = $ui.BufferSize
  $buffer.Width = [Math]::Max($buffer.Width, $width)
  $buffer.Height = [Math]::Max($buffer.Height, $height)
  $ui.BufferSize = $buffer
  $window = $ui.WindowSize
  $window.Width = $width
  $window.Height = $height
  $ui.WindowSize = $window
  Clear-Host
} catch { }
$piArgs = @($launch.args)
& $launch.node @piArgs
exit $LASTEXITCODE
`
	wrapped := exec.Command(powershell, "-NoLogo", "-NoProfile", "-EncodedCommand", encodePowerShell(script))
	wrapped.Env = append(os.Environ(), "PI_PORTABLE_POWERSHELL_LAUNCH="+string(launch))
	return wrapped
}

func encodePowerShell(script string) string {
	units := utf16.Encode([]rune(script))
	data := make([]byte, len(units)*2)
	for index, unit := range units {
		binary.LittleEndian.PutUint16(data[index*2:], unit)
	}
	return base64.StdEncoding.EncodeToString(data)
}

func setupConsoleFont() {
	// Consolas replaces the raster font often used by older Windows consoles.
	type fontInfo struct {
		Size   uint32
		Index  uint32
		X, Y   int16
		Family uint32
		Weight uint32
		Face   [32]uint16
	}
	font := fontInfo{Y: 18, Family: 54, Weight: 400}
	font.Size = uint32(unsafe.Sizeof(font))
	copy(font.Face[:], utf16.Encode([]rune("Consolas")))
	kernel32.NewProc("SetCurrentConsoleFontEx").Call(os.Stdout.Fd(), 0, uintptr(unsafe.Pointer(&font)))
}
