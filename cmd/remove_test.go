package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wolf-jonathan/workspace-x/cmd"
	"github.com/wolf-jonathan/workspace-x/internal/workspace"
)

func TestRemoveDeletesConfigEntryAndWorkspaceLink(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)

	mustInitWorkspace(t, root, "payments-debug")

	target := filepath.Join(t.TempDir(), "auth-service")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll(target) error = %v", err)
	}

	add := cmd.NewRootCommand()
	add.SetArgs([]string{"add", target})
	add.SetOut(new(bytes.Buffer))
	add.SetErr(new(bytes.Buffer))

	if err := cmd.ExecuteCommand(add); err != nil {
		t.Fatalf("add ExecuteCommand() error = %v", err)
	}

	instructionFile := filepath.Join(root, "CLAUDE.md")
	if err := os.WriteFile(instructionFile, []byte("workspace instructions\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(CLAUDE.md) error = %v", err)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	remove := cmd.NewRootCommand()
	remove.SetArgs([]string{"remove", "auth-service"})
	remove.SetOut(stdout)
	remove.SetErr(stderr)

	if err := cmd.ExecuteCommand(remove); err != nil {
		t.Fatalf("remove ExecuteCommand() error = %v", err)
	}

	if !strings.Contains(stderr.String(), "Warning: workspace instruction files may be stale; run wsx agent-init") {
		t.Fatalf("stderr = %q, want stale instruction warning", stderr.String())
	}

	content, err := os.ReadFile(instructionFile)
	if err != nil {
		t.Fatalf("ReadFile(CLAUDE.md) error = %v", err)
	}
	if string(content) != "workspace instructions\n" {
		t.Fatalf("CLAUDE.md = %q, want unchanged content", string(content))
	}

	loaded, err := workspace.LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(loaded.Config.Refs) != 0 {
		t.Fatalf("refs length after remove = %d, want 0", len(loaded.Config.Refs))
	}

	linkPath := filepath.Join(root, "auth-service")
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Fatalf("workspace link stat error = %v, want not exists", err)
	}

	if !strings.Contains(stdout.String(), "Removed \"auth-service\"") {
		t.Fatalf("stdout = %q, want remove confirmation", stdout.String())
	}
}

func TestRemoveLeavesTargetDirectoryUntouched(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)

	mustInitWorkspace(t, root, "payments-debug")

	target := filepath.Join(t.TempDir(), "payments-api")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll(target) error = %v", err)
	}

	marker := filepath.Join(target, "README.md")
	if err := os.WriteFile(marker, []byte("repo stays"), 0o644); err != nil {
		t.Fatalf("WriteFile(marker) error = %v", err)
	}

	add := cmd.NewRootCommand()
	add.SetArgs([]string{"add", target})
	add.SetOut(new(bytes.Buffer))
	add.SetErr(new(bytes.Buffer))

	if err := cmd.ExecuteCommand(add); err != nil {
		t.Fatalf("add ExecuteCommand() error = %v", err)
	}

	instructionFile := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(instructionFile, []byte("workspace instructions\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(AGENTS.md) error = %v", err)
	}

	stderr := new(bytes.Buffer)
	remove := cmd.NewRootCommand()
	remove.SetArgs([]string{"remove", "payments-api"})
	remove.SetOut(new(bytes.Buffer))
	remove.SetErr(stderr)

	if err := cmd.ExecuteCommand(remove); err != nil {
		t.Fatalf("remove ExecuteCommand() error = %v", err)
	}

	if !strings.Contains(stderr.String(), "Warning: workspace instruction files may be stale; run wsx agent-init") {
		t.Fatalf("stderr = %q, want stale instruction warning", stderr.String())
	}

	instructionContent, err := os.ReadFile(instructionFile)
	if err != nil {
		t.Fatalf("ReadFile(AGENTS.md) error = %v", err)
	}
	if string(instructionContent) != "workspace instructions\n" {
		t.Fatalf("AGENTS.md = %q, want unchanged content", string(instructionContent))
	}

	markerContent, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("ReadFile(marker) error = %v", err)
	}

	if string(markerContent) != "repo stays" {
		t.Fatalf("marker content = %q, want repo stays", string(markerContent))
	}
}

func TestRemoveRejectsUnknownRefWithoutMutatingConfig(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)

	mustInitWorkspace(t, root, "payments-debug")

	remove := cmd.NewRootCommand()
	remove.SetArgs([]string{"remove", "missing"})
	remove.SetOut(new(bytes.Buffer))
	remove.SetErr(new(bytes.Buffer))

	err := cmd.ExecuteCommand(remove)
	if err == nil {
		t.Fatal("ExecuteCommand() error = nil, want missing ref error")
	}

	if !strings.Contains(strings.ToLower(err.Error()), "not found") {
		t.Fatalf("error = %q, want not found message", err.Error())
	}

	loaded, loadErr := workspace.LoadConfig(root)
	if loadErr != nil {
		t.Fatalf("LoadConfig() error = %v", loadErr)
	}

	if len(loaded.Config.Refs) != 0 {
		t.Fatalf("refs length after failure = %d, want 0", len(loaded.Config.Refs))
	}
}
