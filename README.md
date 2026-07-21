# modrel

`modrel` releases Go modules in single-module and multi-module repositories.

It discovers modules from `go.mod`, follows Go module tag conventions, and leaves project-specific version file updates to hooks.

## Install From Source

```bash
make install
```

This installs `modrel` to `$(go env GOPATH)/bin/modrel`.

## Commands

List discovered modules:

```bash
modrel list
```

Print a release plan:

```bash
modrel plan .
modrel plan boot --type rc
modrel plan database/kafka --version v1.2.3
```

Apply a release:

```bash
modrel apply . --version v1.2.3
modrel apply boot --type rc
modrel apply database/kafka --version v1.2.3 --dry-run --yes
modrel apply . --version v1.2.3 --push
```

The root command is a planning shortcut:

```bash
modrel .
```

When no path is provided, `modrel` prompts for a discovered module.

## Versions

Stable versions use:

```text
v1.2.3
```

RC versions use:

```text
v1.2.3-rc.1
```

Manual versions must include the leading `v` and must match one of:

```text
vMAJOR.MINOR.PATCH
vMAJOR.MINOR.PATCH-rc.N
```

## Tags

Root module tags:

```text
v1.2.3
v1.2.3-rc.1
```

Submodule tags:

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

## Configuration

Configuration is optional. Put `.modrel.yaml` at the git repository root.

```yaml
discovery:
  exclude:
    - "third_party/**"
    - "cmd/demo"

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
    checks:
      - "go test ./..."
    commit: "release(boot): {{ .Version }}"
```

Without config, `modrel` uses:

```text
checks: go test ./...
root commit: release: <version>
submodule commit: release(<path>): <version>
```

`apply` requires release file changes. In normal use, provide an update hook that modifies the version file, `go.mod`, or other release metadata.

This repository uses `modrel` for both its root module and the example module at `examples/hello`. Both modules keep their current release version in a `VERSION` file, updated by `scripts/release/update-version.sh`:

```yaml
modules:
  ".":
    update:
      - 'sh "$MODREL_REPO_ROOT/scripts/release/update-version.sh"'

  "examples/hello":
    update:
      - 'sh "$MODREL_REPO_ROOT/scripts/release/update-version.sh"'
```

Their tags are `vX.Y.Z` and `examples/hello/vX.Y.Z`, respectively.

## Hook Environment

Hooks run from the selected module directory and receive:

```text
MODREL_REPO_ROOT
MODREL_MODULE
MODREL_MODULE_DIR
MODREL_MODULE_PATH
MODREL_VERSION
MODREL_TAG
MODREL_LATEST_TAG
```

Example update hook:

```bash
#!/usr/bin/env bash
set -euo pipefail

cd "$MODREL_REPO_ROOT"
perl -0pi -e \
  "s/const SDKVersion = \"[^\"]+\"/const SDKVersion = \"$MODREL_VERSION\"/" \
  internal/version/version.go
```

## Safety Flags

```text
--dry-run   Print the plan without changing files, commits, tags, or remotes.
--yes       Skip confirmation prompts.
--push      Push the release commit and tag after creating them.
```
