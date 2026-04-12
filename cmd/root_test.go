package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jwolf/wsx/cmd"
)

func TestRootHelpShowsSupportedCommands(t *testing.T) {
	command := cmd.NewRootCommand()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"--help"})

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := stdout.String()
	for _, snippet := range []string{
		"Currently supported commands:",
		"init",
		"Only implemented commands are shown below.",
	} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("help output = %q, want substring %q", output, snippet)
		}
	}
}
