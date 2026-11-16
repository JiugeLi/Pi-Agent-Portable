package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// The default is tied to the executable, so Explorer and terminal launches
// always share a config and session catalogue. Relative overrides use launch cwd.
func resolveProject(cwd, executable string, args []string, override string) (string, []string, bool, error) {
	project := override
	remaining := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			remaining = append(remaining, args[index:]...)
			break
		}
		if arg == "--project" {
			if index+1 == len(args) || strings.HasPrefix(args[index+1], "--") || strings.TrimSpace(args[index+1]) == "" {
				return "", nil, false, fmt.Errorf("--project requires a folder path")
			}
			index++
			project = args[index]
		} else if strings.HasPrefix(arg, "--project=") {
			project = strings.TrimPrefix(arg, "--project=")
			if strings.TrimSpace(project) == "" {
				return "", nil, false, fmt.Errorf("--project requires a folder path")
			}
		} else {
			remaining = append(remaining, arg)
		}
	}
	explicit := project != ""
	if !explicit {
		project = filepath.Join(filepath.Dir(executable), "pi-project")
	} else if !filepath.IsAbs(project) {
		project = filepath.Join(cwd, project)
	}
	return filepath.Clean(project), remaining, explicit, nil
}

// Match the pinned upstream SessionManager's default directory encoding. Keeping
// the native default makes every imported chat appear in /resume's initial list.
func projectSessionDir(project string) string {
	encoded := strings.TrimPrefix(strings.TrimPrefix(project, "/"), "\\")
	encoded = strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(encoded)
	return filepath.Join(project, ".pi-portable", "agent", "sessions", "--"+encoded+"--")
}

func copyMissing(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return err
	}
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	_, writeErr := io.Copy(output, input)
	closeErr := output.Close()
	if writeErr != nil {
		os.Remove(destination) // Only the new, incomplete copy belongs to us.
		return writeErr
	}
	if closeErr != nil {
		os.Remove(destination)
		return closeErr
	}
	if info, err := input.Stat(); err == nil {
		// Preserve the original chat ordering in Pi's resume picker.
		os.Chtimes(destination, info.ModTime(), info.ModTime())
	}
	return nil
}

func importLegacyProjects(project string, legacyDirs []string) error {
	agent := filepath.Join(project, ".pi-portable", "agent")
	for _, directory := range legacyDirs {
		oldAgent := filepath.Join(directory, ".pi-portable", "agent")
		if filepath.Clean(oldAgent) == filepath.Clean(agent) {
			continue
		}
		info, err := os.Lstat(oldAgent)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		id := sha256.Sum256([]byte(filepath.Clean(oldAgent)))
		marker := filepath.Join(project, ".pi-portable", "migrations", "project-"+hex.EncodeToString(id[:12])+".complete")
		if _, err := os.Stat(marker); err == nil {
			continue
		}
		for _, name := range []string{"auth.json", "settings.json", "models-store.json"} {
			source := filepath.Join(oldAgent, name)
			if info, err := os.Lstat(source); err == nil && info.Mode().IsRegular() {
				if err := copyMissing(source, filepath.Join(agent, name)); err != nil {
					return err
				}
			}
		}
		sessions := filepath.Join(oldAgent, "sessions")
		if info, err := os.Lstat(sessions); err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			err := filepath.WalkDir(sessions, func(source string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || !entry.Type().IsRegular() || filepath.Ext(source) != ".jsonl" {
					return nil
				}
				// Include source identity, so filename collisions cannot erase a chat.
				hash := sha256.Sum256([]byte(source))
				name := strings.TrimSuffix(entry.Name(), ".jsonl") + "-import-" + hex.EncodeToString(hash[:8]) + ".jsonl"
				return copyMissing(source, filepath.Join(projectSessionDir(project), name))
			})
			if err != nil {
				return err
			}
		}
		if err := os.MkdirAll(filepath.Dir(marker), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(marker, []byte("Imported without changing source files.\n"), 0600); err != nil {
			return err
		}
	}
	ignore := filepath.Join(project, ".gitignore")
	if _, err := os.Stat(ignore); os.IsNotExist(err) {
		file, err := os.OpenFile(ignore, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		_, writeErr := file.WriteString(".pi-portable/\n/config.json\n")
		closeErr := file.Close()
		if writeErr != nil {
			return writeErr
		}
		return closeErr
	}
	return nil
}
