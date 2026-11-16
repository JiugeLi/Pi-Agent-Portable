package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProjectIsStableAcrossLaunchDirectories(t *testing.T) {
	base := t.TempDir()
	exe := filepath.Join(base, "app", "pi")
	for _, launch := range []string{filepath.Join(base, "one"), filepath.Join(base, "two")} {
		project, args, explicit, err := resolveProject(launch, exe, []string{"--resume"}, "")
		if err != nil || project != filepath.Join(base, "app", "pi-project") || explicit || !reflect.DeepEqual(args, []string{"--resume"}) {
			t.Fatalf("unstable project: %q %v %v %v", project, args, explicit, err)
		}
	}
}

func TestProjectOverrideAndCLIForwarding(t *testing.T) {
	base := t.TempDir()
	project, args, explicit, err := resolveProject(base, filepath.Join(base, "app", "pi"),
		[]string{"--project", "工作 project", "--mode", "rpc", "--", "--project=literal-prompt"}, "ignored")
	if err != nil || !explicit || project != filepath.Join(base, "工作 project") ||
		!reflect.DeepEqual(args, []string{"--mode", "rpc", "--", "--project=literal-prompt"}) {
		t.Fatalf("invalid override or CLI forwarding: %q %v %v", project, args, err)
	}
	for _, arguments := range [][]string{{"--project"}, {"--project="}, {"--project", "--resume"}} {
		if _, _, _, err := resolveProject(base, "pi", arguments, ""); err == nil {
			t.Fatal("accepted missing project path")
		}
	}
	project, _, explicit, err = resolveProject(base, "pi", nil, "env-folder")
	if err != nil || !explicit || project != filepath.Join(base, "env-folder") {
		t.Fatal("environment override failed")
	}
}

func TestLegacyMigrationPreservesSourcesAndDoesNotReimport(t *testing.T) {
	base := t.TempDir()
	project, legacy := filepath.Join(base, "pi-project"), filepath.Join(base, "old")
	oldAgent := filepath.Join(legacy, ".pi-portable", "agent")
	for _, group := range []string{"one", "two"} {
		folder := filepath.Join(oldAgent, "sessions", group)
		if err := os.MkdirAll(folder, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, "same-name.jsonl"), []byte("session-"+group), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(oldAgent, "auth.json"), []byte("test-saved-auth"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := importLegacyProjects(project, []string{legacy, legacy}); err != nil {
		t.Fatal(err)
	}
	sessions, err := filepath.Glob(filepath.Join(projectSessionDir(project), "*.jsonl"))
	if err != nil || len(sessions) != 2 {
		t.Fatal("migration lost sessions with colliding filenames")
	}
	auth := filepath.Join(project, ".pi-portable", "agent", "auth.json")
	if err := os.WriteFile(auth, []byte("updated-auth"), 0600); err != nil {
		t.Fatal(err)
	}
	// Deleting an imported session must not make it reappear on the next launch.
	if err := os.Remove(sessions[0]); err != nil {
		t.Fatal(err)
	}
	if err := importLegacyProjects(project, []string{legacy}); err != nil {
		t.Fatal(err)
	}
	sessions, _ = filepath.Glob(filepath.Join(projectSessionDir(project), "*.jsonl"))
	if len(sessions) != 1 {
		t.Fatal("migration reimported a deleted chat")
	}
	data, err := os.ReadFile(auth)
	if err != nil || string(data) != "updated-auth" {
		t.Fatal("overwrote saved credentials")
	}
	for _, group := range []string{"one", "two"} {
		data, err := os.ReadFile(filepath.Join(oldAgent, "sessions", group, "same-name.jsonl"))
		if err != nil || string(data) != "session-"+group {
			t.Fatal("legacy session was changed")
		}
	}
}

func TestProjectConfigImportsOnceAndNeverOverwrites(t *testing.T) {
	base := t.TempDir()
	runtimeDir, project, legacy := filepath.Join(base, "runtime"), filepath.Join(base, "pi-project"), filepath.Join(base, "old")
	for _, dir := range []string{filepath.Join(runtimeDir, "defaults"), project, legacy} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"config.json", "README.zh-CN.md", "README.en.md"} {
		if err := os.WriteFile(filepath.Join(runtimeDir, "defaults", name), []byte("default"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	oldConfig := filepath.Join(legacy, "config.json")
	if err := os.WriteFile(oldConfig, []byte("existing-user-config"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PI_PORTABLE_CONFIG", "")
	config := filepath.Join(project, "config.json")
	if err := ensureDefaultFiles(runtimeDir, project, config, legacy); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(config)
	if err != nil || string(data) != "existing-user-config" {
		t.Fatal("did not preserve old config")
	}
	if err := os.WriteFile(config, []byte("new-project-config"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ensureDefaultFiles(runtimeDir, project, config, legacy); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(config)
	if err != nil || string(data) != "new-project-config" {
		t.Fatal("overwrote project config")
	}
	data, err = os.ReadFile(oldConfig)
	if err != nil || string(data) != "existing-user-config" {
		t.Fatal("changed legacy config")
	}
}
