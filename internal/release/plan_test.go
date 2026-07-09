package release

import (
	"testing"

	"github.com/win5do/modrel/internal/discovery"
)

func TestLatestTag(t *testing.T) {
	t.Run("root module uses plain semver tags", func(t *testing.T) {
		module := discovery.Module{Name: ".", RelPath: "."}
		tags := []string{"v1.2.3", "boot/v9.9.9", "v1.2.4-rc.1", "v1.2.4"}
		if got := LatestTag(module, tags); got != "v1.2.4" {
			t.Fatalf("LatestTag = %q, want v1.2.4", got)
		}
	})

	t.Run("submodule uses path prefix", func(t *testing.T) {
		module := discovery.Module{Name: "database/kafka", RelPath: "database/kafka"}
		tags := []string{"v9.9.9", "database/kafka/v1.2.3", "database/kafka/v1.2.4-rc.2", "database/redis/v9.9.9"}
		if got := LatestTag(module, tags); got != "database/kafka/v1.2.4-rc.2" {
			t.Fatalf("LatestTag = %q, want database/kafka/v1.2.4-rc.2", got)
		}
	})
}

func TestCommitMessage(t *testing.T) {
	plan := Plan{
		Module:  discovery.Module{Name: "boot", RelPath: "boot"},
		Version: "v1.2.3",
		Tag:     "boot/v1.2.3",
	}
	if got := commitMessage(plan); got != "release(boot): v1.2.3" {
		t.Fatalf("commitMessage = %q", got)
	}

	plan.CommitMessage = "release {{ .Tag }} from {{ .Path }}"
	if got := commitMessage(plan); got != "release boot/v1.2.3 from boot" {
		t.Fatalf("templated commitMessage = %q", got)
	}
}
