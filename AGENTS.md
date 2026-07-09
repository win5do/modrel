# AGENTS.md

## Project Positioning

`modrel` is a Go CLI for releasing Go modules in single-module and multi-module repositories.

The first version should stay focused on Go module release workflows:

- Discover modules from `go.mod`.
- Use standard Go module tag rules.
- Support stable versions like `v1.2.3`.
- Support RC versions like `v1.2.3-rc.1`.
- Keep repository-specific file updates in hooks instead of hard-coding updater logic.

## Working Rules

- Keep changes small and milestone-oriented.
- After completing each clear implementation or documentation step, create a git commit.
- Do not batch unrelated steps into one commit.
- Before committing, run the narrowest useful verification for the changed area.
- If verification cannot be run, state that in the commit context or final response.
- Do not rewrite or amend existing commits unless explicitly requested.

## Implementation Defaults

- Use Go.
- Use Cobra for CLI command parsing.
- Use `huh` for interactive prompts.
- Use `os/exec` to call the local `git` binary.
- Use `golang.org/x/mod/modfile` for parsing `go.mod`.
- Keep full-screen Bubble Tea UI out of the first version.

## First-Version Boundaries

Do not implement these in the first version unless explicitly requested:

- Multi-module dependency graph orchestration.
- Publishing multiple modules in one command.
- Changelog generation.
- GitHub Release creation.
- Release PR automation.
- Custom tag templates.
- Non-Go module release support.
- Built-in source updaters such as `go-const` or `go-mod-require`.

## File And Style Rules

- Prefer simple package boundaries under `internal/`.
- Keep release behavior testable by wrapping git and prompt interactions behind small interfaces.
- Prefer explicit errors with actionable context.
- Keep documentation in sync with behavior whenever workflow semantics change.

