package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPowerShellPreservesPathsArgumentsEnvironmentAndExit(t *testing.T) {
	powershell := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is required for the PowerShell bridge integration check")
	}
	dir := filepath.Join(t.TempDir(), "工作 ' $literal")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	argument := "a 'quoted' path; $env:UNEXPECTED & 中文"
	script := `process.stdout.write(JSON.stringify({args:process.argv.slice(1),cwd:process.cwd(),marker:process.env.PI_BRIDGE_TEST,bridgePresent:!!process.env.PI_PORTABLE_POWERSHELL_LAUNCH}));process.exit(7)`
	cmd := powerShellCommand(exec.Command(node, "-e", script, "--", argument, ""), powershell)
	cmd.Dir = dir
	cmd.Env = append(cmd.Env, "PI_BRIDGE_TEST=inherited")
	output, err := cmd.Output()
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 7 {
		t.Fatalf("expected exit 7, got %v", err)
	}
	var result struct {
		Args          []string `json:"args"`
		Cwd           string   `json:"cwd"`
		Marker        string   `json:"marker"`
		BridgePresent bool     `json:"bridgePresent"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatal("PowerShell polluted the child output:", err)
	}
	// Windows PowerShell 5.1 drops empty native arguments. Interactive launch
	// only forwards our nonempty bootstrap path; arbitrary CLI args bypass it.
	if len(result.Args) < 1 || result.Args[0] != argument || result.Cwd != dir || result.Marker != "inherited" || result.BridgePresent {
		t.Fatalf("PowerShell changed launch data: %+v", result)
	}
}
