# wsx

Use `wsx` when you need to inspect or operate on a multi-repo workspace built
from links to existing local repositories.

## What `wsx` manages

- A workspace root containing `.wsx.json`, `.wsx.env`, and linked repo directories.
- Portable committed config in `.wsx.json`.
- Local machine path variables in `.wsx.env`.
- Symlinks or Windows junctions created at the workspace root.

## Required invariants

- Keep `${VAR}` placeholders in `.wsx.json` when available. Do not rewrite stored paths to machine-specific absolute paths.
- Resolve `${VAR}` placeholders only at point of use.
- Treat `.wsx.env` as local-only state. It should be gitignored and never committed from generated workspaces.
- Treat `link_type` as runtime state. Detect it from disk instead of storing it in `.wsx.json`.
- On Windows, expect link creation to try symlinks first and fall back to junctions on permission errors.
- `wsx exec` forwards argv directly. Shell operators work only if the caller explicitly invokes a shell.

## Recommended workflow

1. Run `wsx doctor --json` first when you enter an unfamiliar workspace.
2. Use `wsx list --json` to inspect linked repos and resolved paths.
3. Use `wsx status --json`, `wsx fetch --json`, or `wsx exec --json -- ...` for structured multi-repo automation.
4. Use `wsx tree`, `wsx grep`, `wsx dump`, and `wsx prompt` for AI-oriented workspace inspection.

## Command guidance

- `wsx init`: creates `.wsx.json`, `.wsx.env`, and ensures `.wsx.env` is in `.gitignore`.
- `wsx add`: accepts absolute or parameterized paths, rejects circular refs, and creates the runtime link.
- `wsx remove`: removes the workspace link and config entry only. It must not touch the target repo.
- `wsx list`: reports live link health and runtime link type.
- `wsx doctor`: distinguishes interactive TTY use from non-interactive agent or CI use.
- `wsx dump`: requires a narrowing filter unless `--all-files` is explicitly set.

## Repo guidance

- The design source of truth is `docs/wsx-design-plan.md`.
- The execution and handoff source of truth is `docs/implementation-plan.md`.
- Keep top-level agent docs aligned with actual CLI behavior.
