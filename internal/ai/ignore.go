package ai

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

const gitIgnoreFileName = ".gitignore"

var builtInIgnoreLines = []string{
	".git/",
	".DS_Store",
	"Thumbs.db",
	"*.pyc",
	"__pycache__/",
	"*.class",
}

type IgnoreMatcher struct {
	repoRoot string
	chain    *ignore.GitIgnore
	builtIns *ignore.GitIgnore
}

func LoadIgnoreMatcher(repoRoot string) (*IgnoreMatcher, error) {
	absoluteRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}

	lines, err := loadIgnoreChain(absoluteRoot)
	if err != nil {
		return nil, err
	}

	return &IgnoreMatcher{
		repoRoot: absoluteRoot,
		chain:    ignore.CompileIgnoreLines(lines...),
		builtIns: ignore.CompileIgnoreLines(builtInIgnoreLines...),
	}, nil
}

func (m *IgnoreMatcher) MatchesPath(path string) bool {
	if m == nil {
		return false
	}

	relativePath, ok := m.normalizePath(path)
	if !ok || relativePath == "" {
		return false
	}

	return m.chain.MatchesPath(relativePath) || m.builtIns.MatchesPath(relativePath)
}

func loadIgnoreChain(repoRoot string) ([]string, error) {
	lines := make([]string, 0, 32)

	globalLines, err := loadGlobalIgnoreLines()
	if err != nil {
		return nil, err
	}
	lines = append(lines, globalLines...)

	ignoreFiles, err := collectIgnoreFiles(repoRoot)
	if err != nil {
		return nil, err
	}

	for _, path := range ignoreFiles {
		scopeDir := filepath.Dir(path)
		baseRel, err := filepath.Rel(repoRoot, scopeDir)
		if err != nil {
			return nil, err
		}

		fileLines, err := readIgnoreFileLines(path)
		if err != nil {
			return nil, err
		}

		for _, line := range fileLines {
			lines = append(lines, scopeIgnoreLine(line, baseRel))
		}
	}

	return lines, nil
}

func loadGlobalIgnoreLines() ([]string, error) {
	globalPath := findGlobalIgnoreFile()
	if globalPath == "" {
		return nil, nil
	}

	return readIgnoreFileLines(globalPath)
}

func findGlobalIgnoreFile() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		return ""
	}

	for _, candidate := range []string{
		filepath.Join(home, ".config", "git", "ignore"),
		filepath.Join(home, ".gitignore_global"),
	} {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate
		}
	}

	return ""
}

func collectIgnoreFiles(repoRoot string) ([]string, error) {
	paths := make([]string, 0, 8)

	err := filepath.WalkDir(repoRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}

		if entry.IsDir() {
			return nil
		}

		if entry.Name() == gitIgnoreFileName {
			paths = append(paths, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(paths, func(i, j int) bool {
		leftDepth := strings.Count(filepath.Clean(paths[i]), string(filepath.Separator))
		rightDepth := strings.Count(filepath.Clean(paths[j]), string(filepath.Separator))
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return paths[i] < paths[j]
	})

	return paths, nil
}

func readIgnoreFileLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lines := make([]string, 0, 16)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func scopeIgnoreLine(line, baseRel string) string {
	baseRel = filepath.ToSlash(filepath.Clean(baseRel))
	if baseRel == "." {
		baseRel = ""
	}

	trimmed := strings.TrimRight(line, "\r")
	if baseRel == "" || trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return trimmed
	}

	negated := false
	pattern := trimmed
	if strings.HasPrefix(pattern, "!") {
		negated = true
		pattern = pattern[1:]
	}

	if pattern == "" || strings.HasPrefix(pattern, "#") {
		return trimmed
	}

	suffixed := strings.HasSuffix(pattern, "/")
	core := strings.TrimSuffix(pattern, "/")

	var scoped string
	switch {
	case strings.HasPrefix(core, "/"):
		scoped = "/" + joinIgnorePath(baseRel, strings.TrimPrefix(core, "/"))
	case strings.Contains(core, "/"):
		scoped = joinIgnorePath(baseRel, core)
	default:
		scoped = joinIgnorePath(baseRel, "**", core)
	}

	if suffixed {
		scoped += "/"
	}
	if negated {
		scoped = "!" + scoped
	}

	return scoped
}

func joinIgnorePath(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = filepath.ToSlash(strings.TrimSpace(part))
		part = strings.Trim(part, "/")
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}

	return strings.Join(filtered, "/")
}

func (m *IgnoreMatcher) normalizePath(path string) (string, bool) {
	if path == "" {
		return "", false
	}

	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) {
		relativePath, err := filepath.Rel(m.repoRoot, cleanPath)
		if err != nil {
			return "", false
		}
		if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
			return "", false
		}
		cleanPath = relativePath
	}

	cleanPath = filepath.ToSlash(cleanPath)
	cleanPath = strings.TrimPrefix(cleanPath, "./")
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	if cleanPath == "." || cleanPath == "" {
		return "", false
	}

	return cleanPath, true
}
