package release

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/win5do/modrel/internal/discovery"
	"github.com/win5do/modrel/internal/git"
	"github.com/win5do/modrel/internal/version"
)

type Plan struct {
	Module        discovery.Module
	Version       string
	Tag           string
	LatestTag     string
	UpdateHooks   []string
	CheckHooks    []string
	CommitMessage string
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

type ApplyOptions struct {
	NoPush bool
}

func Apply(ctx context.Context, out io.Writer, repoRoot string, plan Plan, opts ApplyOptions) error {
	clean, err := git.IsClean(ctx, repoRoot)
	if err != nil {
		return err
	}
	if !clean {
		return fmt.Errorf("worktree is not clean")
	}

	for _, hook := range plan.UpdateHooks {
		if err := runCommand(ctx, out, plan.Module.Dir, "update hook", hook); err != nil {
			return err
		}
	}

	status, err := git.Status(ctx, repoRoot)
	if err != nil {
		return err
	}
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("release produced no file changes; configure an update hook or make changes before apply")
	}

	diff, err := git.Diff(ctx, repoRoot)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "Changed files:")
	fmt.Fprintln(out, strings.TrimSpace(status))
	if strings.TrimSpace(diff) != "" {
		fmt.Fprintln(out, diff)
	}

	for _, check := range plan.CheckHooks {
		if err := runCommand(ctx, out, plan.Module.Dir, "check", check); err != nil {
			return err
		}
	}

	if err := git.AddAll(ctx, repoRoot); err != nil {
		return err
	}
	if err := git.Commit(ctx, repoRoot, commitMessage(plan)); err != nil {
		return err
	}
	if err := git.Tag(ctx, repoRoot, plan.Tag); err != nil {
		return err
	}

	if opts.NoPush {
		fmt.Fprintf(out, "Created commit and tag locally. Skipped push because --no-push was set.\n")
		return nil
	}
	if err := git.PushHEAD(ctx, repoRoot); err != nil {
		return err
	}
	if err := git.PushTag(ctx, repoRoot, plan.Tag); err != nil {
		return err
	}
	fmt.Fprintf(out, "Pushed release %s.\n", plan.Tag)
	return nil
}

func runCommand(ctx context.Context, out io.Writer, dir string, label string, command string) error {
	fmt.Fprintf(out, "Running %s: %s\n", label, command)
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed in %s: %w", label, filepath.ToSlash(dir), err)
	}
	return nil
}

func commitMessage(plan Plan) string {
	if plan.CommitMessage != "" {
		return renderTemplate(plan.CommitMessage, plan)
	}
	if plan.Module.RelPath == "." {
		return fmt.Sprintf("release: %s", plan.Version)
	}
	return fmt.Sprintf("release(%s): %s", plan.Module.RelPath, plan.Version)
}

func renderTemplate(tmpl string, plan Plan) string {
	replacer := strings.NewReplacer(
		"{{ .Version }}", plan.Version,
		"{{ .Tag }}", plan.Tag,
		"{{ .Module }}", plan.Module.Name,
		"{{ .Path }}", plan.Module.RelPath,
	)
	return replacer.Replace(tmpl)
}
