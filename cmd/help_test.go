package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wolf-jonathan/workspace-x/cmd"
)

func TestCommandHelpShowsExamples(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		snippets []string
	}{
		{
			name: "agent-init",
			args: []string{"agent-init", "--help"},
			snippets: []string{
				"Examples:",
				"wsx agent-init --purpose \"Debug payment incidents\"",
			},
		},
		{
			name: "doctor",
			args: []string{"doctor", "--help"},
			snippets: []string{
				"Examples:",
				"wsx doctor",
				"wsx doctor --fix",
			},
		},
		{
			name: "add",
			args: []string{"add", "--help"},
			snippets: []string{
				"Examples:",
				"wsx add C:\\src\\repos\\auth-service",
				"wsx add ${WORK_REPOS}\\payments-api --as payments",
			},
		},
		{
			name: "exec",
			args: []string{"exec", "--help"},
			snippets: []string{
				"Examples:",
				"wsx exec -- git status",
				"wsx exec --parallel -- npm test",
			},
		},
		{
			name: "tree",
			args: []string{"tree", "--help"},
			snippets: []string{
				"Examples:",
				"wsx tree",
				"wsx tree --all --depth 1",
			},
		},
		{
			name: "grep",
			args: []string{"grep", "--help"},
			snippets: []string{
				"Examples:",
				"wsx grep handleAuth --include \"*.go,*.ts\"",
				"wsx grep refreshToken --json --context 1",
			},
		},
		{
			name: "favorite add",
			args: []string{"favorite", "add", "--help"},
			snippets: []string{
				"Examples:",
				"wsx favorite add C:\\src\\repos --name WORK_REPOS",
			},
		},
		{
			name: "favorite list",
			args: []string{"favorite", "list", "--help"},
			snippets: []string{
				"Examples:",
				"wsx favorite list",
				"wsx favorite list --json",
			},
		},
		{
			name: "favorite import",
			args: []string{"favorite", "import", "--help"},
			snippets: []string{
				"Examples:",
				"wsx favorite import WORK_REPOS",
				"wsx favorite import WORK_REPOS CORP_REPOS",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			command := cmd.NewRootCommand()
			stdout := new(bytes.Buffer)
			stderr := new(bytes.Buffer)
			command.SetArgs(tc.args)
			command.SetOut(stdout)
			command.SetErr(stderr)

			if err := cmd.ExecuteCommand(command); err != nil {
				t.Fatalf("ExecuteCommand() error = %v", err)
			}

			output := stdout.String()
			for _, snippet := range tc.snippets {
				if !strings.Contains(output, snippet) {
					t.Fatalf("help output = %q, want substring %q", output, snippet)
				}
			}

			if stderr.Len() != 0 {
				t.Fatalf("help stderr = %q, want empty", stderr.String())
			}
		})
	}
}
