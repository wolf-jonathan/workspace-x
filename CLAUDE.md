# wsx - AI Workspace Manager

## What this is

A Go CLI that manages a workspace directory by linking existing local
repositories into one AI-friendly workspace. The workspace stays portable
because committed config keeps `${VAR}` placeholders and local machine state
lives in `.wsx.env`.

## Project layout

- `cmd/`: Cobra command entrypoints. Keep command files thin.
- `internal/workspace/`: config, env loading, path resolution, and link handling.
- `internal/git/`: shared git command runner and repo operations.
- `internal/ai/`: AI-facing tree, grep, prompt, agent, and skill helpers.
- `README.md`: public product and usage documentation.
- `SKILL.md`: bundled agent guidance shipped with the repo.

## Core invariants

- `.wsx.json` stores portable paths and must keep `${VAR}` placeholders intact.
- Placeholder resolution happens at point of use via `internal/workspace/env.go`.
- `.wsx.env` is local-only state and must never be committed by generated workspaces.
- `link_type` is runtime state. Detect it from disk; do not persist it in `.wsx.json`.
- On Windows, link creation tries symlinks first and falls back to junctions on permission errors.
- `wsx exec` forwards argv directly to process execution. Shell behavior is opt-in and explicit.
- Commands intended for AI use should emit clean plain text and support `--json` where structured output is useful.
- Do not emit ANSI color when stdout is not a TTY.

## Working rules

- Treat `README.md`, CLI help text, and tests as the product source of truth unless explicitly changed.
- Preserve the workspace model: workspace root plus `.wsx.json`, `.wsx.env`, and linked repo directories.
- Prefer small, reviewable changes and reuse shared internal seams instead of duplicating logic in commands.
- If behavior changes, update `README.md` and `SKILL.md` so the docs stay aligned with the CLI.
