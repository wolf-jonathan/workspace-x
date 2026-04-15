package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wolf-jonathan/workspace-x/internal/ai"
	"github.com/wolf-jonathan/workspace-x/internal/workspace"
)

func TestDoctorReportsHealthyWorkspace(t *testing.T) {
	root := t.TempDir()
	chdirForDoctorTest(t, root)
	writeDoctorWorkspaceConfig(t, root, workspace.Config{
		Version: "1",
		Name:    "payments-debug",
		Refs: []workspace.Ref{
			{Name: "auth-service", Path: `${WORK_REPOS}/auth-service`},
		},
	})

	reposRoot := filepath.Join(t.TempDir(), "repos")
	target := filepath.Join(reposRoot, "auth-service")
	if err := os.MkdirAll(filepath.Join(target, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.git) error = %v", err)
	}
	writeDoctorEnvFile(t, root, "WORK_REPOS="+reposRoot+"\n")
	if _, err := workspace.CreateLink(target, filepath.Join(root, "auth-service")); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}
	writeDoctorInstructionFiles(t, root, buildDoctorInstructionContent(t, "payments-debug", []ai.InstructionRepo{{Name: "auth-service", Root: target}}))

	restore := swapDoctorTerminalDetector(func() bool { return false })
	defer restore()

	stdout := new(bytes.Buffer)
	command := NewRootCommand()
	command.SetArgs([]string{"doctor"})
	command.SetOut(stdout)
	command.SetErr(new(bytes.Buffer))

	if err := ExecuteCommand(command); err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}

	output := stdout.String()
	for _, snippet := range []string{
		"OK  config_valid",
		"OK  env_file",
		"OK  var_WORK_REPOS",
		"OK  auth-service_link",
		"OK  no_duplicate_names",
		"OK  no_case_collisions",
		"OK  no_workspace_nesting",
		"OK  no_nested_refs",
		"OK  auth-service_git",
		"OK  workspace_instruction_files",
	} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("doctor output = %q, want substring %q", output, snippet)
		}
	}
}

func TestDoctorWarnsWhenWorkspaceInstructionFilesAreMissing(t *testing.T) {
	root := t.TempDir()
	chdirForDoctorTest(t, root)
	writeDoctorWorkspaceConfig(t, root, workspace.Config{
		Version: "1",
		Name:    "payments-debug",
		Refs: []workspace.Ref{
			{Name: "auth-service", Path: `${WORK_REPOS}/auth-service`},
		},
	})

	reposRoot := filepath.Join(t.TempDir(), "repos")
	target := filepath.Join(reposRoot, "auth-service")
	if err := os.MkdirAll(filepath.Join(target, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.git) error = %v", err)
	}
	writeDoctorEnvFile(t, root, "WORK_REPOS="+reposRoot+"\n")
	if _, err := workspace.CreateLink(target, filepath.Join(root, "auth-service")); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}

	restore := swapDoctorTerminalDetector(func() bool { return false })
	defer restore()

	stdout := new(bytes.Buffer)
	command := NewRootCommand()
	command.SetArgs([]string{"doctor", "--json"})
	command.SetOut(stdout)
	command.SetErr(new(bytes.Buffer))

	if err := ExecuteCommand(command); err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}

	var result doctorReport
	if decodeErr := json.Unmarshal(stdout.Bytes(), &result); decodeErr != nil {
		t.Fatalf("json.Unmarshal() error = %v", decodeErr)
	}

	if !result.Healthy {
		t.Fatal("result.Healthy = false, want true")
	}

	check, ok := findDoctorCheck(result.Checks, "workspace_instruction_files")
	if !ok {
		t.Fatalf("result.Checks = %+v, want workspace_instruction_files warning", result.Checks)
	}
	if check.Status != doctorStatusWarn {
		t.Fatalf("workspace_instruction_files status = %q, want %q", check.Status, doctorStatusWarn)
	}
	if !strings.Contains(check.Message, "missing") {
		t.Fatalf("workspace_instruction_files message = %q, want missing warning", check.Message)
	}
	if !strings.Contains(check.Message, "AGENTS.md") || !strings.Contains(check.Message, "CLAUDE.md") {
		t.Fatalf("workspace_instruction_files message = %q, want both file names", check.Message)
	}
}

func TestDoctorJSONReportsUnresolvedVariableInNonInteractiveMode(t *testing.T) {
	root := t.TempDir()
	chdirForDoctorTest(t, root)
	writeDoctorWorkspaceConfig(t, root, workspace.Config{
		Version: "1",
		Name:    "payments-debug",
		Refs: []workspace.Ref{
			{Name: "auth-service", Path: `${WORK_REPOS}/auth-service`},
		},
	})

	restore := swapDoctorTerminalDetector(func() bool { return false })
	defer restore()

	stdout := new(bytes.Buffer)
	command := NewRootCommand()
	command.SetArgs([]string{"doctor", "--json"})
	command.SetOut(stdout)
	command.SetErr(new(bytes.Buffer))

	err := ExecuteCommand(command)
	if err == nil {
		t.Fatal("ExecuteCommand() error = nil, want unresolved variable failure")
	}

	var result doctorReport
	if decodeErr := json.Unmarshal(stdout.Bytes(), &result); decodeErr != nil {
		t.Fatalf("json.Unmarshal() error = %v", decodeErr)
	}

	if result.Healthy {
		t.Fatal("result.Healthy = true, want false")
	}

	foundEnvWarning := false
	foundVarError := false
	for _, check := range result.Checks {
		if check.Name == "env_file" && check.Status == doctorStatusWarn {
			foundEnvWarning = true
		}
		if check.Name == "var_WORK_REPOS" && check.Status == doctorStatusError {
			foundVarError = true
		}
	}

	if !foundEnvWarning {
		t.Fatalf("result.Checks = %+v, want env_file warning", result.Checks)
	}
	if !foundVarError {
		t.Fatalf("result.Checks = %+v, want var_WORK_REPOS error", result.Checks)
	}
	assertDoctorCheckAbsent(t, result.Checks, "workspace_instruction_files")
}

func TestDoctorFixRequiresInteractiveTerminal(t *testing.T) {
	root := t.TempDir()
	chdirForDoctorTest(t, root)
	writeDoctorWorkspaceConfig(t, root, workspace.Config{
		Version: "1",
		Name:    "payments-debug",
	})

	restore := swapDoctorTerminalDetector(func() bool { return false })
	defer restore()

	command := NewRootCommand()
	command.SetArgs([]string{"doctor", "--fix"})
	command.SetOut(new(bytes.Buffer))
	command.SetErr(new(bytes.Buffer))

	err := ExecuteCommand(command)
	if err == nil {
		t.Fatal("ExecuteCommand() error = nil, want --fix terminal error")
	}
	if !strings.Contains(err.Error(), "--fix requires an interactive terminal") {
		t.Fatalf("ExecuteCommand() error = %q, want --fix terminal error", err.Error())
	}
}

func TestDoctorDoesNotPromptOrWriteEnvWithoutFix(t *testing.T) {
	root := t.TempDir()
	chdirForDoctorTest(t, root)
	writeDoctorWorkspaceConfig(t, root, workspace.Config{
		Version: "1",
		Name:    "payments-debug",
		Refs: []workspace.Ref{
			{Name: "auth-service", Path: `${WORK_REPOS}/auth-service`},
		},
	})

	restore := swapDoctorTerminalDetector(func() bool { return true })
	defer restore()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	command := NewRootCommand()
	command.SetArgs([]string{"doctor"})
	command.SetIn(strings.NewReader("C:\\repos\n"))
	command.SetOut(stdout)
	command.SetErr(stderr)

	err := ExecuteCommand(command)
	if err == nil {
		t.Fatal("ExecuteCommand() error = nil, want unresolved variable failure")
	}

	if strings.Contains(stdout.String(), "Enter the path for WORK_REPOS") {
		t.Fatalf("doctor stdout = %q, want no prompt without --fix", stdout.String())
	}
	if strings.Contains(stderr.String(), "Enter the path for WORK_REPOS") {
		t.Fatalf("doctor stderr = %q, want no prompt without --fix", stderr.String())
	}
	if _, statErr := os.Stat(filepath.Join(root, workspace.EnvFileName)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("Stat(.wsx.env) error = %v, want not exists", statErr)
	}
}

func TestDoctorInteractiveModeResolvesVariablesAndWritesEnvFile(t *testing.T) {
	root := t.TempDir()
	chdirForDoctorTest(t, root)
	writeDoctorWorkspaceConfig(t, root, workspace.Config{
		Version: "1",
		Name:    "payments-debug",
		Refs: []workspace.Ref{
			{Name: "auth-service", Path: `${WORK_REPOS}/auth-service`},
		},
	})

	reposRoot := filepath.Join(t.TempDir(), "repos")
	target := filepath.Join(reposRoot, "auth-service")
	if err := os.MkdirAll(filepath.Join(target, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.git) error = %v", err)
	}
	if _, err := workspace.CreateLink(target, filepath.Join(root, "auth-service")); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}
	writeDoctorInstructionFiles(t, root, buildDoctorInstructionContent(t, "payments-debug", []ai.InstructionRepo{{Name: "auth-service", Root: target}}))

	restore := swapDoctorTerminalDetector(func() bool { return true })
	defer restore()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	command := NewRootCommand()
	command.SetArgs([]string{"doctor", "--fix"})
	command.SetIn(strings.NewReader(reposRoot + "\n"))
	command.SetOut(stdout)
	command.SetErr(stderr)

	if err := ExecuteCommand(command); err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}

	output := stdout.String() + stderr.String()
	for _, snippet := range []string{
		"Enter the path for WORK_REPOS",
		"Saved WORK_REPOS to .wsx.env",
		"OK  var_WORK_REPOS",
	} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("doctor output = %q, want substring %q", output, snippet)
		}
	}

	content, err := os.ReadFile(filepath.Join(root, workspace.EnvFileName))
	if err != nil {
		t.Fatalf("ReadFile(.wsx.env) error = %v", err)
	}
	if string(content) != "WORK_REPOS="+reposRoot+"\n" {
		t.Fatalf(".wsx.env = %q, want WORK_REPOS entry", string(content))
	}
}

func TestDoctorWarnsWhenWorkspaceInstructionFilesAreStaleAfterAdd(t *testing.T) {
	root := t.TempDir()
	chdirForDoctorTest(t, root)
	writeDoctorWorkspaceConfig(t, root, workspace.Config{
		Version: "1",
		Name:    "payments-debug",
		Refs: []workspace.Ref{
			{Name: "auth-service", Path: `${WORK_REPOS}/auth-service`},
		},
	})

	reposRoot := filepath.Join(t.TempDir(), "repos")
	authTarget := filepath.Join(reposRoot, "auth-service")
	frontendTarget := filepath.Join(reposRoot, "frontend")
	for _, dir := range []string{authTarget, frontendTarget} {
		if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatalf("MkdirAll(.git) error = %v", err)
		}
	}
	writeDoctorEnvFile(t, root, "WORK_REPOS="+reposRoot+"\n")
	if _, err := workspace.CreateLink(authTarget, filepath.Join(root, "auth-service")); err != nil {
		t.Fatalf("CreateLink(auth-service) error = %v", err)
	}
	content := buildDoctorInstructionContent(t, "payments-debug", []ai.InstructionRepo{{Name: "auth-service", Root: authTarget}})
	writeDoctorInstructionFiles(t, root, content)

	cfg := workspace.Config{
		Version: "1",
		Name:    "payments-debug",
		Refs: []workspace.Ref{
			{Name: "auth-service", Path: `${WORK_REPOS}/auth-service`},
			{Name: "frontend", Path: `${WORK_REPOS}/frontend`},
		},
	}
	writeDoctorWorkspaceConfig(t, root, cfg)
	if _, err := workspace.CreateLink(frontendTarget, filepath.Join(root, "frontend")); err != nil {
		t.Fatalf("CreateLink(frontend) error = %v", err)
	}

	restore := swapDoctorTerminalDetector(func() bool { return false })
	defer restore()

	stdout := new(bytes.Buffer)
	command := NewRootCommand()
	command.SetArgs([]string{"doctor", "--json"})
	command.SetOut(stdout)
	command.SetErr(new(bytes.Buffer))

	if err := ExecuteCommand(command); err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}

	var result doctorReport
	if decodeErr := json.Unmarshal(stdout.Bytes(), &result); decodeErr != nil {
		t.Fatalf("json.Unmarshal() error = %v", decodeErr)
	}

	check, ok := findDoctorCheck(result.Checks, "workspace_instruction_files")
	if !ok {
		t.Fatalf("result.Checks = %+v, want workspace_instruction_files warning", result.Checks)
	}
	if check.Status != doctorStatusWarn {
		t.Fatalf("workspace_instruction_files status = %q, want %q", check.Status, doctorStatusWarn)
	}
	if !strings.Contains(check.Message, "stale") {
		t.Fatalf("workspace_instruction_files message = %q, want stale warning", check.Message)
	}
}

func TestDoctorWarnsWhenWorkspaceInstructionFilesAreStaleAfterRemove(t *testing.T) {
	root := t.TempDir()
	chdirForDoctorTest(t, root)
	writeDoctorWorkspaceConfig(t, root, workspace.Config{
		Version: "1",
		Name:    "payments-debug",
		Refs: []workspace.Ref{
			{Name: "auth-service", Path: `${WORK_REPOS}/auth-service`},
			{Name: "frontend", Path: `${WORK_REPOS}/frontend`},
		},
	})

	reposRoot := filepath.Join(t.TempDir(), "repos")
	authTarget := filepath.Join(reposRoot, "auth-service")
	frontendTarget := filepath.Join(reposRoot, "frontend")
	for _, dir := range []string{authTarget, frontendTarget} {
		if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatalf("MkdirAll(.git) error = %v", err)
		}
	}
	writeDoctorEnvFile(t, root, "WORK_REPOS="+reposRoot+"\n")
	for name, target := range map[string]string{
		"auth-service": authTarget,
		"frontend":     frontendTarget,
	} {
		if _, err := workspace.CreateLink(target, filepath.Join(root, name)); err != nil {
			t.Fatalf("CreateLink(%s) error = %v", name, err)
		}
	}
	content := buildDoctorInstructionContent(t, "payments-debug", []ai.InstructionRepo{
		{Name: "auth-service", Root: authTarget},
		{Name: "frontend", Root: frontendTarget},
	})
	writeDoctorInstructionFiles(t, root, content)

	if err := workspace.RemoveLink(filepath.Join(root, "frontend")); err != nil {
		t.Fatalf("RemoveLink(frontend) error = %v", err)
	}
	writeDoctorWorkspaceConfig(t, root, workspace.Config{
		Version: "1",
		Name:    "payments-debug",
		Refs: []workspace.Ref{
			{Name: "auth-service", Path: `${WORK_REPOS}/auth-service`},
		},
	})

	restore := swapDoctorTerminalDetector(func() bool { return false })
	defer restore()

	stdout := new(bytes.Buffer)
	command := NewRootCommand()
	command.SetArgs([]string{"doctor", "--json"})
	command.SetOut(stdout)
	command.SetErr(new(bytes.Buffer))

	if err := ExecuteCommand(command); err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}

	var result doctorReport
	if decodeErr := json.Unmarshal(stdout.Bytes(), &result); decodeErr != nil {
		t.Fatalf("json.Unmarshal() error = %v", decodeErr)
	}

	check, ok := findDoctorCheck(result.Checks, "workspace_instruction_files")
	if !ok {
		t.Fatalf("result.Checks = %+v, want workspace_instruction_files warning", result.Checks)
	}
	if check.Status != doctorStatusWarn {
		t.Fatalf("workspace_instruction_files status = %q, want %q", check.Status, doctorStatusWarn)
	}
	if !strings.Contains(check.Message, "stale") {
		t.Fatalf("workspace_instruction_files message = %q, want stale warning", check.Message)
	}
}

func TestDoctorWarnsWhenWorkspaceInstructionFilesAreStaleAfterRepoInstructionChange(t *testing.T) {
	root := t.TempDir()
	chdirForDoctorTest(t, root)
	writeDoctorWorkspaceConfig(t, root, workspace.Config{
		Version: "1",
		Name:    "payments-debug",
		Refs: []workspace.Ref{
			{Name: "auth-service", Path: `${WORK_REPOS}/auth-service`},
		},
	})

	reposRoot := filepath.Join(t.TempDir(), "repos")
	target := filepath.Join(reposRoot, "auth-service")
	if err := os.MkdirAll(filepath.Join(target, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.git) error = %v", err)
	}
	writeDoctorEnvFile(t, root, "WORK_REPOS="+reposRoot+"\n")
	if _, err := workspace.CreateLink(target, filepath.Join(root, "auth-service")); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}
	content := buildDoctorInstructionContent(t, "payments-debug", []ai.InstructionRepo{{Name: "auth-service", Root: target}})
	writeDoctorInstructionFiles(t, root, content)

	if err := os.MkdirAll(filepath.Join(target, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll(docs) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "docs", "AGENTS.md"), []byte("# Docs Agents\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(docs/AGENTS.md) error = %v", err)
	}

	restore := swapDoctorTerminalDetector(func() bool { return false })
	defer restore()

	stdout := new(bytes.Buffer)
	command := NewRootCommand()
	command.SetArgs([]string{"doctor", "--json"})
	command.SetOut(stdout)
	command.SetErr(new(bytes.Buffer))

	if err := ExecuteCommand(command); err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}

	var result doctorReport
	if decodeErr := json.Unmarshal(stdout.Bytes(), &result); decodeErr != nil {
		t.Fatalf("json.Unmarshal() error = %v", decodeErr)
	}

	check, ok := findDoctorCheck(result.Checks, "workspace_instruction_files")
	if !ok {
		t.Fatalf("result.Checks = %+v, want workspace_instruction_files warning", result.Checks)
	}
	if check.Status != doctorStatusWarn {
		t.Fatalf("workspace_instruction_files status = %q, want %q", check.Status, doctorStatusWarn)
	}
	if !strings.Contains(check.Message, "stale") {
		t.Fatalf("workspace_instruction_files message = %q, want stale warning", check.Message)
	}
}

func TestDoctorFixJSONKeepsStdoutMachineReadable(t *testing.T) {
	root := t.TempDir()
	chdirForDoctorTest(t, root)
	writeDoctorWorkspaceConfig(t, root, workspace.Config{
		Version: "1",
		Name:    "payments-debug",
		Refs: []workspace.Ref{
			{Name: "auth-service", Path: `${WORK_REPOS}/auth-service`},
		},
	})

	reposRoot := filepath.Join(t.TempDir(), "repos")
	target := filepath.Join(reposRoot, "auth-service")
	if err := os.MkdirAll(filepath.Join(target, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.git) error = %v", err)
	}
	if _, err := workspace.CreateLink(target, filepath.Join(root, "auth-service")); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}
	writeDoctorInstructionFiles(t, root, buildDoctorInstructionContent(t, "payments-debug", []ai.InstructionRepo{{Name: "auth-service", Root: target}}))

	restore := swapDoctorTerminalDetector(func() bool { return true })
	defer restore()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	command := NewRootCommand()
	command.SetArgs([]string{"doctor", "--fix", "--json"})
	command.SetIn(strings.NewReader(reposRoot + "\n"))
	command.SetOut(stdout)
	command.SetErr(stderr)

	if err := ExecuteCommand(command); err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}

	var result doctorReport
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; stdout = %q", err, stdout.String())
	}

	if !strings.Contains(stderr.String(), "Enter the path for WORK_REPOS") {
		t.Fatalf("doctor stderr = %q, want prompt output", stderr.String())
	}
	if strings.Contains(stdout.String(), "Enter the path for WORK_REPOS") {
		t.Fatalf("doctor stdout = %q, want JSON only", stdout.String())
	}
}

func TestDoctorReportsRepointedWorkspaceLinkAsBroken(t *testing.T) {
	root := t.TempDir()
	chdirForDoctorTest(t, root)
	writeDoctorWorkspaceConfig(t, root, workspace.Config{
		Version: "1",
		Name:    "payments-debug",
		Refs: []workspace.Ref{
			{Name: "auth-service", Path: `${WORK_REPOS}/auth-service`},
		},
	})

	reposRoot := filepath.Join(t.TempDir(), "repos")
	configuredTarget := filepath.Join(reposRoot, "auth-service")
	actualTarget := filepath.Join(reposRoot, "other-service")
	for _, dir := range []string{configuredTarget, actualTarget} {
		if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatalf("MkdirAll(.git) error = %v", err)
		}
	}
	writeDoctorEnvFile(t, root, "WORK_REPOS="+reposRoot+"\n")
	if _, err := workspace.CreateLink(actualTarget, filepath.Join(root, "auth-service")); err != nil {
		t.Fatalf("CreateLink() error = %v", err)
	}

	restore := swapDoctorTerminalDetector(func() bool { return false })
	defer restore()

	stdout := new(bytes.Buffer)
	command := NewRootCommand()
	command.SetArgs([]string{"doctor", "--json"})
	command.SetOut(stdout)
	command.SetErr(new(bytes.Buffer))

	err := ExecuteCommand(command)
	if err == nil {
		t.Fatal("ExecuteCommand() error = nil, want unhealthy workspace failure")
	}

	var result doctorReport
	if decodeErr := json.Unmarshal(stdout.Bytes(), &result); decodeErr != nil {
		t.Fatalf("json.Unmarshal() error = %v", decodeErr)
	}

	for _, check := range result.Checks {
		if check.Name != "auth-service_link" {
			continue
		}
		if check.Status != doctorStatusError {
			t.Fatalf("auth-service_link status = %q, want %q", check.Status, doctorStatusError)
		}
		if !strings.Contains(check.Message, "instead of") {
			t.Fatalf("auth-service_link message = %q, want repointed-link detail", check.Message)
		}
		return
	}

	t.Fatalf("result.Checks = %+v, want auth-service_link error", result.Checks)
}

func buildDoctorInstructionContent(t *testing.T, workspaceName string, repos []ai.InstructionRepo) string {
	t.Helper()

	content, err := ai.BuildWorkspaceInstructionContent(workspaceName, "", repos)
	if err != nil {
		t.Fatalf("BuildWorkspaceInstructionContent() error = %v", err)
	}

	return content
}

func writeDoctorInstructionFiles(t *testing.T, root, content string) {
	t.Helper()

	for _, relativePath := range []string{"CLAUDE.md", "AGENTS.md"} {
		path := filepath.Join(root, relativePath)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", relativePath, err)
		}
	}
}

func findDoctorCheck(checks []doctorCheck, name string) (doctorCheck, bool) {
	for _, check := range checks {
		if check.Name == name {
			return check, true
		}
	}

	return doctorCheck{}, false
}

func assertDoctorCheckAbsent(t *testing.T, checks []doctorCheck, name string) {
	t.Helper()

	for _, check := range checks {
		if check.Name == name {
			t.Fatalf("checks = %+v, did not expect %s check", checks, name)
		}
	}
}

func writeDoctorWorkspaceConfig(t *testing.T, root string, cfg workspace.Config) {
	t.Helper()

	if err := workspace.SaveConfig(root, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
}

func writeDoctorEnvFile(t *testing.T, root, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(root, workspace.EnvFileName), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(.wsx.env) error = %v", err)
	}
}

func chdirForDoctorTest(t *testing.T, dir string) {
	t.Helper()

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore Chdir() error = %v", err)
		}
	})
}

func swapDoctorTerminalDetector(detector func() bool) func() {
	previous := doctorIsTerminal
	doctorIsTerminal = detector

	return func() {
		doctorIsTerminal = previous
	}
}
