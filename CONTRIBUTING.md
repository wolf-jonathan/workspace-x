# Contributing

## Scope

`wsx` is a CLI for building AI-friendly linked workspaces from
existing local repositories. Changes should preserve the core workspace model:
`.wsx.json`, `.wsx.env`, and linked repo directories at the workspace root.

## Development setup

Requirements:

- Go 1.22+

Useful local commands:

```powershell
go run . --help
go test ./...
go test ./cmd
go test ./internal/ai
go test ./internal/workspace
go test ./internal/git
```

Optional helper scripts:

```powershell
.\scripts\build.ps1
.\scripts\run-all-tests.ps1
```

## Contribution guidelines

- Preserve portable `${VAR}` placeholders in `.wsx.json`.
- Resolve placeholders at point of use rather than rewriting stored config.
- Keep `.wsx.env` local-only workspace state.
- On Windows, link creation must try symlinks first and fall back to directory
  junctions when permission errors occur.
- Keep command output parseable and stable.
- If behavior changes, update `README.md`, `SKILL.md`, and CLI help text in the
  same change.

## Pull requests

- Keep changes small and reviewable.
- Include or update tests for behavior changes when practical.
- Prefer user-facing examples that work on Windows PowerShell.
- Verify `go test ./...` passes before opening a PR.
