package ai

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	SkillScopeLocal  = "local"
	SkillScopeGlobal = "global"
	SkillName        = "wsx"
)

const defaultBundledSkill = `# wsx

Use wsx when you need to inspect or operate on a multi-repo workspace built
from links to existing local repositories.

## What wsx manages

- A workspace root containing .wsx.json, .wsx.env, and linked repo directories.
- Portable committed config in .wsx.json.
- Local machine path variables in .wsx.env.
- Symlinks or Windows junctions created at the workspace root.

## Required invariants

- Keep ${VAR} placeholders in .wsx.json when available. Do not rewrite stored paths to machine-specific absolute paths.
- Resolve ${VAR} placeholders only at point of use.
- Treat .wsx.env as local-only state. It should be gitignored and never committed from generated workspaces.
- Treat link_type as runtime state. Detect it from disk instead of storing it in .wsx.json.
- On Windows, expect link creation to try symlinks first and fall back to junctions on permission errors.
- wsx exec forwards argv directly. Shell operators work only if the caller explicitly invokes a shell.

## Recommended workflow

1. Run wsx doctor --json first when you enter an unfamiliar workspace.
2. Use wsx list --json to inspect linked repos and resolved paths.
3. Use wsx status --json, wsx fetch --json, or wsx exec --json -- ... for structured multi-repo automation.
4. Use wsx tree, wsx grep, wsx dump, and wsx prompt for AI-oriented workspace inspection.

## Command guidance

- wsx init: creates .wsx.json, .wsx.env, and ensures .wsx.env is in .gitignore.
- wsx add: accepts absolute or parameterized paths, rejects circular refs, and creates the runtime link.
- wsx remove: removes the workspace link and config entry only. It must not touch the target repo.
- wsx list: reports live link health and runtime link type.
- wsx doctor: distinguishes interactive TTY use from non-interactive agent or CI use.
- wsx dump: requires a narrowing filter unless --all-files is explicitly set.
`

type SkillInstallResult struct {
	Scope     string
	Directory string
	SkillFile string
}

var skillHomeDir = os.UserHomeDir

var readBundledSkill = func(repoRoot string) ([]byte, error) {
	path := filepath.Join(repoRoot, "SKILL.md")
	data, err := os.ReadFile(path)
	if err == nil {
		return data, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return []byte(defaultBundledSkill), nil
	}
	return nil, err
}

func InstallBundledSkill(repoRoot, scope string) (SkillInstallResult, error) {
	location, err := resolveSkillInstallLocation(repoRoot, scope)
	if err != nil {
		return SkillInstallResult{}, err
	}

	if _, err := os.Stat(location.Directory); err == nil {
		return SkillInstallResult{}, fmt.Errorf("skill already installed at %s", location.Directory)
	} else if !errors.Is(err, os.ErrNotExist) {
		return SkillInstallResult{}, err
	}

	content, err := readBundledSkill(repoRoot)
	if err != nil {
		return SkillInstallResult{}, err
	}

	if err := os.MkdirAll(location.Directory, 0o755); err != nil {
		return SkillInstallResult{}, err
	}

	if err := os.WriteFile(location.SkillFile, content, 0o644); err != nil {
		return SkillInstallResult{}, err
	}

	return location, nil
}

func UninstallBundledSkill(repoRoot, scope string) (SkillInstallResult, error) {
	location, err := resolveSkillInstallLocation(repoRoot, scope)
	if err != nil {
		return SkillInstallResult{}, err
	}

	if _, err := os.Stat(location.SkillFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SkillInstallResult{}, fmt.Errorf("skill is not installed at %s", location.Directory)
		}
		return SkillInstallResult{}, err
	}

	if err := os.RemoveAll(location.Directory); err != nil {
		return SkillInstallResult{}, err
	}

	return location, nil
}

func resolveSkillInstallLocation(repoRoot, scope string) (SkillInstallResult, error) {
	normalizedScope := strings.ToLower(strings.TrimSpace(scope))
	switch normalizedScope {
	case SkillScopeLocal:
		root := filepath.Clean(repoRoot)
		directory := filepath.Join(root, ".agents", "skills", SkillName)
		return SkillInstallResult{
			Scope:     normalizedScope,
			Directory: directory,
			SkillFile: filepath.Join(directory, "SKILL.md"),
		}, nil
	case SkillScopeGlobal:
		homeDir, err := skillHomeDir()
		if err != nil {
			return SkillInstallResult{}, err
		}
		directory := filepath.Join(homeDir, ".agents", "skills", SkillName)
		return SkillInstallResult{
			Scope:     normalizedScope,
			Directory: directory,
			SkillFile: filepath.Join(directory, "SKILL.md"),
		}, nil
	default:
		return SkillInstallResult{}, fmt.Errorf("unsupported scope %q: must be local or global", scope)
	}
}
