package release

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/win5do/modrel/internal/discovery"
)

func TestApplyCreatesCommitAndTag(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "modrel@example.com")
	runGit(t, root, "config", "user.name", "modrel test")

	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.23\n")
	writeFile(t, filepath.Join(root, "main.go"), "package main\n")
	runGit(t, root, "add", "--all")
	runGit(t, root, "commit", "-m", "initial")

	plan := Plan{
		Module: discovery.Module{
			Name:       ".",
			Dir:        root,
			RelPath:    ".",
			ModulePath: "example.com/app",
		},
		Version:     "v1.2.3",
		Tag:         "v1.2.3",
		UpdateHooks: []string{`printf "%s\n%s\n%s\n" "$MODREL_VERSION" "$MODREL_TAG" "$MODREL_MODULE_PATH" > VERSION`},
		CheckHooks:  []string{"go test ./..."},
	}

	var out bytes.Buffer
	err := Apply(context.Background(), &out, root, plan, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply returned error: %v\noutput:\n%s", err, out.String())
	}

	versionFile, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		t.Fatalf("ReadFile VERSION returned error: %v", err)
	}
	if got, want := string(versionFile), "v1.2.3\nv1.2.3\nexample.com/app\n"; got != want {
		t.Fatalf("VERSION = %q, want %q", got, want)
	}

	tag := strings.TrimSpace(runGit(t, root, "tag", "--list", "v1.2.3"))
	if tag != "v1.2.3" {
		t.Fatalf("tag = %q, want v1.2.3", tag)
	}
	log := runGit(t, root, "log", "-1", "--pretty=%s")
	if strings.TrimSpace(log) != "release: v1.2.3" {
		t.Fatalf("commit subject = %q", strings.TrimSpace(log))
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String()
}

func TestApplyWithoutUpdateChanges(t *testing.T) {
	for _, tc := range []struct {
		name            string
		updates, checks []string
		fail, commit    bool
	}{
		{name: "no hooks"},
		{name: "noop update", updates: []string{"true"}, checks: []string{"true"}},
		{name: "failed check", checks: []string{"exit 1"}, fail: true},
		{name: "check creates file", checks: []string{"echo checked > checked.txt"}, commit: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			runGit(t, root, "init")
			runGit(t, root, "config", "user.email", "modrel@example.com")
			runGit(t, root, "config", "user.name", "modrel test")
			writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.23\n")
			runGit(t, root, "add", "--all")
			runGit(t, root, "commit", "-m", "initial")
			head := runGit(t, root, "rev-parse", "HEAD")
			plan := Plan{Module: discovery.Module{Name: ".", Dir: root, RelPath: ".", ModulePath: "example.com/app"}, Version: "v1.0.0", Tag: "v1.0.0", UpdateHooks: tc.updates, CheckHooks: tc.checks}
			var out bytes.Buffer
			err := Apply(context.Background(), &out, root, plan, ApplyOptions{})
			if (err != nil) != tc.fail {
				t.Fatalf("Apply error = %v; output %s", err, out.String())
			}
			after := runGit(t, root, "rev-parse", "HEAD")
			if (after != head) != tc.commit {
				t.Fatalf("unexpected HEAD change: %s -> %s", head, after)
			}
			tags := strings.TrimSpace(runGit(t, root, "tag", "--list"))
			if tc.fail {
				if tags != "" {
					t.Fatal("failed check created tag")
				}
				return
			}
			if tags != plan.Tag {
				t.Fatalf("tags = %q", tags)
			}
			if got := runGit(t, root, "rev-parse", plan.Tag+"^{commit}"); got != after {
				t.Fatal("tag does not point to HEAD")
			}
			if got := strings.TrimSpace(runGit(t, root, "status", "--porcelain")); got != "" {
				t.Fatalf("dirty tree: %s", got)
			}
		})
	}
}
