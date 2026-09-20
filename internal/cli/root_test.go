package cli

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/win5do/modrel/internal/buildinfo"
)

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got, want := out.String(), buildinfo.Version+"\n"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestMajorDirectoryPlan(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := exec.Command("git", "init", "-q").Run(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("database/dmq/v2", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("database/dmq/v2/go.mod", []byte("module example.com/repo/database/dmq/v2\n\ngo 1.23\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args []string
		want string
		fail bool
	}{
		{[]string{"plan", "database/dmq/v2", "--type", "stable"}, "database/dmq/v2.0.0", false},
		{[]string{"plan", "database/dmq/v2", "--type", "rc"}, "database/dmq/v2.0.0-rc.1", false},
		{[]string{"plan", "database/dmq/v2", "--version", "v3.0.0"}, "", true},
	} {
		cmd := NewRootCommand()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(tc.args)
		err := cmd.Execute()
		if (err != nil) != tc.fail {
			t.Fatalf("args %v: %v", tc.args, err)
		}
		if !tc.fail && !strings.Contains(out.String(), tc.want) {
			t.Fatalf("output = %s", out.String())
		}
	}
}
