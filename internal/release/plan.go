package release

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/win5do/modrel/internal/discovery"
	"github.com/win5do/modrel/internal/version"
)

type Plan struct {
	Module    discovery.Module
	Version   string
	Tag       string
	LatestTag string
}

func LatestTag(module discovery.Module, tags []string) string {
	prefix := module.TagPrefix()
	matches := make([]string, 0, len(tags))
	for _, tag := range tags {
		if !strings.HasPrefix(tag, prefix) {
			continue
		}
		raw := strings.TrimPrefix(tag, prefix)
		if version.Validate(raw) == nil {
			matches = append(matches, tag)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return version.Compare(version.TrimTagPrefix(matches[i]), version.TrimTagPrefix(matches[j])) > 0
	})
	if len(matches) == 0 {
		return ""
	}
	return matches[0]
}

func PrintPlan(out io.Writer, plan Plan) error {
	latest := plan.LatestTag
	if latest == "" {
		latest = "(none)"
	}
	_, err := fmt.Fprintf(out, `Module:      %s
Module path: %s
Version:     %s
Tag:         %s
Latest tag:  %s

Steps:
  1. Check clean worktree
  2. Run update hooks
  3. Show git diff
  4. Run check hooks
  5. Commit release changes
  6. Create git tag
  7. Push commit and tag
`, plan.Module.Name, plan.Module.ModulePath, plan.Version, plan.Tag, latest)
	return err
}
