package workspace_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/wolf-jonathan/workspace-x/internal/workspace"
)

func TestLoadEnvParsesKeyValuePairsAndIgnoresComments(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	content := "# comment\nWORK_REPOS=C:\\Users\\Yoni\\work\nPERSONAL_REPOS=C:\\Users\\Yoni\\personal\n\n"
	if err := os.WriteFile(filepath.Join(root, workspace.EnvFileName), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	env, err := workspace.LoadEnv(root)
	if err != nil {
		t.Fatalf("LoadEnv() error = %v", err)
	}

	if got := env["WORK_REPOS"]; got != `C:\Users\Yoni\work` {
		t.Fatalf("WORK_REPOS = %q, want C:\\Users\\Yoni\\work", got)
	}

	if got := env["PERSONAL_REPOS"]; got != `C:\Users\Yoni\personal` {
		t.Fatalf("PERSONAL_REPOS = %q, want C:\\Users\\Yoni\\personal", got)
	}
}

func TestLoadEnvReturnsNotFoundWhenFileMissing(t *testing.T) {
	t.Parallel()

	_, err := workspace.LoadEnv(t.TempDir())
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("LoadEnv() error = %v, want os.ErrNotExist", err)
	}
}

func TestResolvePathUsesEnvFileValuesBeforeProcessEnv(t *testing.T) {
	t.Setenv("WORK_REPOS", `D:\fallback`)

	resolved, err := workspace.ResolvePath(`${WORK_REPOS}/auth-service`, workspace.EnvVars{
		"WORK_REPOS": `C:\Users\Yoni\work`,
	})
	if err != nil {
		t.Fatalf("ResolvePath() error = %v", err)
	}

	want := filepath.Clean(`C:\Users\Yoni\work\auth-service`)
	if runtime.GOOS != "windows" {
		want = filepath.Clean(`C:\Users\Yoni\work/auth-service`)
	}

	if resolved != want {
		t.Fatalf("resolved path = %q, want %q", resolved, want)
	}
}

func TestResolvePathFallsBackToProcessEnv(t *testing.T) {
	t.Setenv("PERSONAL_REPOS", `D:\projects`)

	resolved, err := workspace.ResolvePath(`${PERSONAL_REPOS}/side-project`, nil)
	if err != nil {
		t.Fatalf("ResolvePath() error = %v", err)
	}

	want := filepath.Clean(`D:\projects\side-project`)
	if runtime.GOOS != "windows" {
		want = filepath.Clean(`D:\projects/side-project`)
	}

	if resolved != want {
		t.Fatalf("resolved path = %q, want %q", resolved, want)
	}
}

func TestResolvePathFallsBackToGlobalFavorites(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("APPDATA", configDir)
	t.Setenv("HOME", configDir)
	t.Setenv("USERPROFILE", configDir)

	err := workspace.SaveFavoriteStore(workspace.FavoriteStore{
		Favorites: []workspace.Favorite{{
			Name:  "AUTH_SERVICE",
			Path:  `D:\repos\auth-service`,
			Added: time.Now().UTC(),
		}},
	})
	if err != nil {
		t.Fatalf("SaveFavoriteStore() error = %v", err)
	}

	resolved, err := workspace.ResolvePath(`${AUTH_SERVICE}`, nil)
	if err != nil {
		t.Fatalf("ResolvePath() error = %v", err)
	}

	want := filepath.Clean(`D:\repos\auth-service`)
	if resolved != want {
		t.Fatalf("resolved path = %q, want %q", resolved, want)
	}
}

func TestResolvePathNormalizesLiteralSeparatorsWithoutRewritingEnvValues(t *testing.T) {
	resolved, err := workspace.ResolvePath(`${WORK_REPOS}\payments-api/config`, workspace.EnvVars{
		"WORK_REPOS": `C:\Users\Yoni\work`,
	})
	if err != nil {
		t.Fatalf("ResolvePath() error = %v", err)
	}

	want := filepath.Clean(`C:\Users\Yoni\work\payments-api\config`)
	if runtime.GOOS != "windows" {
		want = filepath.Clean(`C:\Users\Yoni\work/payments-api/config`)
	}

	if resolved != want {
		t.Fatalf("resolved path = %q, want %q", resolved, want)
	}
}

func TestResolvePathReturnsErrorForMissingVariable(t *testing.T) {
	t.Parallel()

	_, err := workspace.ResolvePath(`${MISSING}/repo`, nil)
	if !errors.Is(err, workspace.ErrUnresolvedVariable) {
		t.Fatalf("ResolvePath() error = %v, want ErrUnresolvedVariable", err)
	}
}
