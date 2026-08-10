package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigForModule(t *testing.T) {
	t.Run("uses default check without config file", func(t *testing.T) {
		cfg, err := Load(t.TempDir())
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		got := cfg.ForModule(".")
		if len(got.Checks) != 1 || got.Checks[0] != "go test ./..." {
			t.Fatalf("default checks = %#v", got.Checks)
		}
	})

	t.Run("merges defaults and module overrides", func(t *testing.T) {
		root := t.TempDir()
		content := []byte(`[defaults]
checks = ["go test ./..."]

[modules.boot]
updates = ["./update-boot.sh"]
checks = ["go test ./boot/..."]
commit = "release(boot): {{ .Version }}"
`)
		if err := os.WriteFile(filepath.Join(root, ".modrel.toml"), content, 0o644); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}

		cfg, err := Load(root)
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}

		boot := cfg.ForModule("boot")
		if len(boot.Updates) != 1 || boot.Updates[0] != "./update-boot.sh" {
			t.Fatalf("boot updates = %#v", boot.Updates)
		}
		if len(boot.Checks) != 1 || boot.Checks[0] != "go test ./boot/..." {
			t.Fatalf("boot checks = %#v", boot.Checks)
		}
		if boot.Commit != "release(boot): {{ .Version }}" {
			t.Fatalf("boot commit = %q", boot.Commit)
		}
	})

	t.Run("loads discovery excludes", func(t *testing.T) {
		root := t.TempDir()
		content := []byte(`[discovery]
excludes = ["third_party/**"]
`)
		if err := os.WriteFile(filepath.Join(root, ".modrel.toml"), content, 0o644); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}

		cfg, err := Load(root)
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if len(cfg.Discovery.Excludes) != 1 || cfg.Discovery.Excludes[0] != "third_party/**" {
			t.Fatalf("discovery excludes = %#v", cfg.Discovery.Excludes)
		}
	})
}
