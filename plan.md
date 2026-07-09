# modrel Plan

## Goal

`modrel` is a small CLI for releasing Go modules in single-module and multi-module repositories.

The initial scope is intentionally narrow:

- Release Go modules only.
- Discover modules from `go.mod`.
- Use Go module tag rules.
- Support stable and RC versions.
- Allow manual version input with strict validation.
- Keep project-specific file updates outside the core engine through hooks.

## Command Shape

Primary usage:

```bash
modrel <path>
```

Examples:

```bash
modrel .
modrel boot
modrel database/kafka
```

If `<path>` is omitted, `modrel` should discover releasable modules and prompt the user to select one:

```bash
modrel
```

The selected path is always a Go module directory, meaning it contains a `go.mod`.

## Version Format

Initial version rules are fixed and conservative.

Stable version:

```text
v1.2.3
```

RC version:

```text
v1.2.3-rc.1
```

Manual input must match one of:

```text
vMAJOR.MINOR.PATCH
vMAJOR.MINOR.PATCH-rc.N
```

Validation regex:

```text
^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-rc\.(0|[1-9][0-9]*))?$
```

Notes:

- The leading `v` is required.
- `v1.2.3` is valid.
- `v1.2.3-rc.1` is valid.
- `1.2.3` is invalid.
- `v1.2` is invalid.
- `v1.2.3-rc1` is invalid.
- `v01.2.3` is invalid.

## Tag Rules

Use standard Go module repository tag conventions.

Root module:

```text
v1.2.3
v1.2.3-rc.1
```

Submodule:

```text
path/v1.2.3
path/v1.2.3-rc.1
```

Examples:

```text
boot/v1.2.3
database/kafka/v1.2.3
configcenter/configx/plugin/source/ark/v1.2.3-rc.1
```

The path part is the module directory relative to the repository root.

## Module Discovery

`modrel` should discover modules by walking the repository and finding `go.mod` files.

Default exclusions:

```text
vendor/**
**/vendor/**
.git/**
**/testdata/**
```

A discovered module has:

```text
Name: relative path from repo root, "." for root
Path: filesystem path
ModulePath: module path parsed from go.mod
TagPrefix: "" for root, "<relative-path>/" for submodules
```

Example:

```text
.                         pkg.poizon.com/golang/abtest-sdk-go
boot                      pkg.poizon.com/golang/abtest-sdk-go/boot
database/kafka            pkg.poizon.com/golang/go-contrib/database/kafka
```

## Interactive Flow

When running `modrel <path>`:

1. Resolve repository root.
2. Resolve `<path>` to a discovered Go module.
3. Fetch remote tags.
4. Detect the latest tag for that module.
5. Ask the user to choose version mode:
   - Stable version
   - RC version
   - Manual input
6. If stable or RC is selected, compute the next version from the latest matching tag.
7. If manual input is selected, validate the version string.
8. Show a release plan.
9. Ask for confirmation.
10. Run release steps.

When running `modrel` with no path:

1. Discover modules.
2. Prompt the user to select one module.
3. Continue with the same flow as `modrel <path>`.

## Version Selection

Initial interactive options:

```text
? Select release type:
  Stable
  RC
  Manual
```

Stable behavior:

- If latest stable is `v1.2.3`, propose `v1.2.4`.
- If latest tag is only RC, ask whether to promote the RC base or enter manually.

RC behavior:

- If latest stable is `v1.2.3`, propose `v1.2.4-rc.1`.
- If latest RC is `v1.2.4-rc.1`, propose `v1.2.4-rc.2`.
- RC sequence is scoped to the same base version.

Manual behavior:

- User enters the full version including `v`.
- The input must pass strict validation.
- The computed tag must not already exist.

## Release Plan Output

Before making changes, show a summary:

```text
Module:      boot
Module path: pkg.poizon.com/golang/abtest-sdk-go/boot
Version:     v1.2.3
Tag:         boot/v1.2.3
Latest tag:  boot/v1.2.2

Steps:
  1. Check clean worktree
  2. Run update hooks
  3. Show git diff
  4. Run check hooks
  5. Commit release changes
  6. Create git tag
  7. Push commit and tag
```

## Hooks

The core tool should not hard-code how version source files are updated.

Each repository can provide hooks in a config file, for example `.modrel.yaml`.

Example:

```yaml
defaults:
  checks:
    - "go test ./..."

modules:
  ".":
    update:
      - "./scripts/release/update-main.sh"
    commit: "release: {{ .Version }}"

  "boot":
    update:
      - "../scripts/release/update-boot.sh"
    commit: "release(boot): {{ .Version }}"
```

Hook environment variables:

```text
MODREL_REPO_ROOT=/repo
MODREL_MODULE=boot
MODREL_MODULE_DIR=/repo/boot
MODREL_MODULE_PATH=pkg.poizon.com/golang/abtest-sdk-go/boot
MODREL_VERSION=v1.2.3
MODREL_TAG=boot/v1.2.3
MODREL_LATEST_TAG=boot/v1.2.2
```

Hooks run from the module directory by default.

## Git Steps

Initial release execution:

1. `git fetch --tags --prune origin`
2. Verify clean worktree.
3. Verify target tag does not exist locally or remotely.
4. Run update hooks.
5. Show `git diff`.
6. Run check hooks.
7. Create commit.
8. Create annotated or lightweight tag.
9. Push commit.
10. Push tag.

Default tag type can be lightweight for the first version.

## Config Scope

The first version should keep config optional.

Without config:

- Discover modules automatically.
- Use default Go module tag rules.
- Run `go test ./...` in the selected module.
- Commit message: `release(<module>): <version>` for submodules.
- Commit message: `release: <version>` for root module.

With config:

- Override update hooks.
- Override check hooks.
- Override commit message.
- Add ignore patterns for module discovery.

## Technical Choices

Use Go for the CLI implementation.

Recommended dependencies for the first version:

```text
CLI framework:       github.com/spf13/cobra
Prompt UI:           github.com/charmbracelet/huh
Config parsing:      gopkg.in/yaml.v3
Go module parsing:   golang.org/x/mod/modfile
Version helpers:     golang.org/x/mod/semver where useful
Git execution:       os/exec calling the local git binary
```

### Cobra

Use Cobra for command parsing, help output, flags, and subcommands.

Initial command shape:

```text
modrel [path]
modrel list
modrel plan [path]
modrel apply [path]
```

`modrel [path]` can be the ergonomic interactive entrypoint. `plan` and `apply` should remain available for explicit and scriptable workflows.

### huh

Use `huh` for terminal prompts:

- Module selection when no path is provided.
- Release type selection: stable, RC, or manual.
- Manual version input.
- Final confirmation before applying changes.

Do not use a full-screen TUI framework in the first version. Bubble Tea can be introduced later if module browsing or release dashboards become complex, but `huh` is enough for forms and confirmations.

### Git

Use the local `git` binary through `os/exec` for the first version.

Reasons:

- Release behavior should match what maintainers already do manually.
- Remote tag behavior, credentials, signing, and hooks stay under normal git configuration.
- It avoids reimplementing edge cases through a git library too early.

Wrap git calls behind a small internal interface so tests can use fakes.

### Go Module Files

Use `golang.org/x/mod/modfile` to parse `go.mod` files during module discovery.

Do not update dependency versions inside the core tool in the first version. Project-specific updates remain hook-driven.

### Suggested Package Layout

```text
cmd/modrel/main.go
internal/cli        Cobra command setup
internal/config     .modrel.yaml loading and defaults
internal/discovery  go.mod discovery and module resolution
internal/git        git command wrapper
internal/prompt     huh prompts
internal/release    release plan and apply workflow
internal/version    strict version parsing and next-version logic
```

## Non-Goals For The First Version

- No dependency graph between modules.
- No automatic publishing of multiple modules in one command.
- No changelog generation.
- No GitHub Release creation.
- No release PR workflow.
- No custom tag template.
- No non-Go module support.
- No built-in source updater such as `go-const` or `go-mod-require`.

## Suggested Implementation Milestones

### Milestone 1: Read-only planning

- Initialize Go CLI.
- Discover git repo root.
- Discover Go modules.
- Resolve `modrel <path>`.
- List latest tags for selected module.
- Validate manual version input.
- Print release plan.

### Milestone 2: Interactive version selection

- Add module selector when path is omitted.
- Add stable / RC / manual prompt.
- Compute next stable and RC versions.
- Detect duplicate target tags.

### Milestone 3: Local release execution

- Check clean worktree.
- Run update hooks.
- Show diff.
- Run check hooks.
- Create commit.
- Create tag.
- Support `--no-push`.

### Milestone 4: Push and safety options

- Push commit and tag.
- Add `--dry-run`.
- Add `--yes`.
- Add config file support.
- Improve errors and recovery hints.
