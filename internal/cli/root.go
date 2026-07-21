package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/win5do/modrel/internal/buildinfo"
	"github.com/win5do/modrel/internal/config"
	"github.com/win5do/modrel/internal/discovery"
	"github.com/win5do/modrel/internal/git"
	"github.com/win5do/modrel/internal/prompt"
	"github.com/win5do/modrel/internal/release"
	"github.com/win5do/modrel/internal/version"
)

type options struct {
	version string
	typ     string
	noPush  bool
	push    bool
	dryRun  bool
	yes     bool
}

// NewRootCommand builds the modrel command tree.
func NewRootCommand() *cobra.Command {
	opts := &options{}

	cmd := &cobra.Command{
		Use:   "modrel [path]",
		Short: "Release Go modules from single-module and multi-module repositories",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			return runPlan(cmd.Context(), cmd.OutOrStdout(), target, opts)
		},
	}

	cmd.Flags().StringVar(&opts.version, "version", "", "release version, for example v1.2.3 or v1.2.3-rc.1")
	cmd.Flags().StringVar(&opts.typ, "type", "", "version type to propose when --version is omitted: stable or rc")

	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newPlanCommand())
	cmd.AddCommand(newApplyCommand())
	cmd.AddCommand(newVersionCommand())

	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the modrel version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), buildinfo.Version)
		},
	}
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List discovered Go modules",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := git.Root(cmd.Context(), ".")
			if err != nil {
				return err
			}
			cfg, err := config.Load(root)
			if err != nil {
				return err
			}
			modules, err := discovery.Discover(root, discovery.Options{Exclude: cfg.Discovery.Exclude})
			if err != nil {
				return err
			}
			for _, module := range modules {
				fmt.Fprintf(cmd.OutOrStdout(), "%-32s %s\n", module.Name, module.ModulePath)
			}
			return nil
		},
	}
}

func newPlanCommand() *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:   "plan [path]",
		Short: "Print a release plan without changing files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			return runPlan(cmd.Context(), cmd.OutOrStdout(), target, opts)
		},
	}
	cmd.Flags().StringVar(&opts.version, "version", "", "release version, for example v1.2.3 or v1.2.3-rc.1")
	cmd.Flags().StringVar(&opts.typ, "type", "", "version type to propose when --version is omitted: stable or rc")
	return cmd
}

func newApplyCommand() *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:   "apply [path]",
		Short: "Apply a release plan by committing and tagging",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			repoRoot, plan, err := buildPlan(cmd.Context(), target, opts)
			if err != nil {
				return err
			}
			if err := release.PrintPlan(cmd.OutOrStdout(), plan); err != nil {
				return err
			}
			if !opts.yes && !opts.dryRun {
				confirmed, err := prompt.ConfirmApply(plan.Tag)
				if err != nil {
					return err
				}
				if !confirmed {
					return fmt.Errorf("release cancelled")
				}
			}
			return release.Apply(cmd.Context(), cmd.OutOrStdout(), repoRoot, plan, release.ApplyOptions{
				Push:   opts.push && !opts.noPush,
				DryRun: opts.dryRun,
			})
		},
	}
	cmd.Flags().StringVar(&opts.version, "version", "", "release version, for example v1.2.3 or v1.2.3-rc.1")
	cmd.Flags().StringVar(&opts.typ, "type", "", "version type to propose when --version is omitted: stable or rc")
	cmd.Flags().BoolVar(&opts.push, "push", false, "push the release commit and tag after creating them")
	cmd.Flags().BoolVar(&opts.noPush, "no-push", false, "deprecated: releases are local by default unless --push is set")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "print the release plan without changing files")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "skip confirmation prompts")
	return cmd
}

func runPlan(ctx context.Context, out io.Writer, target string, opts *options) error {
	_, plan, err := buildPlan(ctx, target, opts)
	if err != nil {
		return err
	}
	return release.PrintPlan(out, plan)
}

func buildPlan(ctx context.Context, target string, opts *options) (string, release.Plan, error) {
	root, err := git.Root(ctx, ".")
	if err != nil {
		return "", release.Plan{}, err
	}
	hasOrigin, err := git.HasRemote(ctx, root, "origin")
	if err != nil {
		return "", release.Plan{}, err
	}
	if hasOrigin {
		if err := git.FetchTags(ctx, root, "origin"); err != nil {
			return "", release.Plan{}, err
		}
	}

	cfg, err := config.Load(root)
	if err != nil {
		return "", release.Plan{}, err
	}

	modules, err := discovery.Discover(root, discovery.Options{Exclude: cfg.Discovery.Exclude})
	if err != nil {
		return "", release.Plan{}, err
	}

	var module discovery.Module
	if target == "" {
		module, err = prompt.SelectModule(modules)
		if err != nil {
			return "", release.Plan{}, err
		}
	} else {
		module, err = discovery.Resolve(root, modules, target)
		if err != nil {
			return "", release.Plan{}, err
		}
	}

	tags, err := git.Tags(ctx, root)
	if err != nil {
		return "", release.Plan{}, err
	}
	moduleConfig := cfg.ForModule(module.Name)

	latest := release.LatestTag(module, tags)
	resolvedVersion := opts.version
	if resolvedVersion == "" {
		releaseType := opts.typ
		if releaseType == "" {
			releaseType, resolvedVersion, err = prompt.SelectVersionMode()
			if err != nil {
				return "", release.Plan{}, err
			}
		}
		if resolvedVersion == "" {
			resolvedVersion, err = nextVersion(releaseType, latest)
		}
		if err != nil {
			return "", release.Plan{}, err
		}
	}
	if err := version.Validate(resolvedVersion); err != nil {
		return "", release.Plan{}, err
	}

	plan := release.Plan{
		Module:        module,
		Version:       resolvedVersion,
		Tag:           module.TagFor(resolvedVersion),
		LatestTag:     latest,
		UpdateHooks:   moduleConfig.Update,
		CheckHooks:    moduleConfig.Checks,
		CommitMessage: moduleConfig.Commit,
	}
	if tagExists(plan.Tag, tags) {
		return "", release.Plan{}, fmt.Errorf("target tag %q already exists", plan.Tag)
	}
	return root, plan, nil
}

func tagExists(tag string, tags []string) bool {
	for _, existing := range tags {
		if existing == tag {
			return true
		}
	}
	return false
}

func nextVersion(typ string, latest string) (string, error) {
	switch typ {
	case "stable":
		return version.NextStable(version.TrimTagPrefix(latest))
	case "rc":
		return version.NextRC(version.TrimTagPrefix(latest))
	default:
		return "", fmt.Errorf("unsupported release type %q: expected stable or rc", typ)
	}
}
