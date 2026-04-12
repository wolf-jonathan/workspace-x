package ai

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIgnoreMatcherAppliesRepoNestedAndBuiltInRules(t *testing.T) {
	repoRoot := t.TempDir()
	writeIgnoreFile(t, filepath.Join(repoRoot, ".gitignore"), "*.log\ngenerated/\n")
	writeIgnoreFile(t, filepath.Join(repoRoot, "generated", ".gitignore"), "!keep.txt\n")

	matcher, err := LoadIgnoreMatcher(repoRoot)
	if err != nil {
		t.Fatalf("LoadIgnoreMatcher() error = %v", err)
	}

	cases := []struct {
		path string
		want bool
	}{
		{path: "logs/app.log", want: true},
		{path: "generated/output.txt", want: true},
		{path: "generated/keep.txt", want: false},
		{path: ".git/config", want: true},
		{path: "src/.DS_Store", want: true},
		{path: "src/main.go", want: false},
	}

	for _, tc := range cases {
		if got := matcher.MatchesPath(tc.path); got != tc.want {
			t.Fatalf("MatchesPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestLoadIgnoreMatcherAppliesGlobalIgnoreFromHomeConfig(t *testing.T) {
	repoRoot := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	writeIgnoreFile(t, filepath.Join(home, ".config", "git", "ignore"), "*.secret\n")

	matcher, err := LoadIgnoreMatcher(repoRoot)
	if err != nil {
		t.Fatalf("LoadIgnoreMatcher() error = %v", err)
	}

	if !matcher.MatchesPath("config/app.secret") {
		t.Fatal("MatchesPath(config/app.secret) = false, want true from global ignore")
	}
	if matcher.MatchesPath("config/app.txt") {
		t.Fatal("MatchesPath(config/app.txt) = true, want false")
	}
}

func TestLoadIgnoreMatcherScopesNestedIgnoreToItsBranchOnly(t *testing.T) {
	repoRoot := t.TempDir()
	writeIgnoreFile(t, filepath.Join(repoRoot, "services", "api", ".gitignore"), "vendor/\n")

	matcher, err := LoadIgnoreMatcher(repoRoot)
	if err != nil {
		t.Fatalf("LoadIgnoreMatcher() error = %v", err)
	}

	if !matcher.MatchesPath("services/api/vendor/module.txt") {
		t.Fatal("MatchesPath(services/api/vendor/module.txt) = false, want true")
	}
	if matcher.MatchesPath("services/web/vendor/module.txt") {
		t.Fatal("MatchesPath(services/web/vendor/module.txt) = true, want false")
	}
}

func writeIgnoreFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
