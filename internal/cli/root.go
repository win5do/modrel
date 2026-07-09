package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/win5do/modrel/internal/discovery"
	"github.com/win5do/modrel/internal/git"
	"github.com/win5do/modrel/internal/prompt"
	"github.com/win5do/modrel/internal/release"
	"github.com/win5do/modrel/internal/version"
)

type options struct {
	version string
	typ     string
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

	return cmd
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
			modules, err := discovery.Discover(root)
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

func runPlan(ctx context.Context, out io.Writer, target string, opts *options) error {
	root, err := git.Root(ctx, ".")
	if err != nil {
		return err
	}

	modules, err := discovery.Discover(root)
	if err != nil {
		return err
	}

	var module discovery.Module
	if target == "" {
		module, err = prompt.SelectModule(modules)
		if err != nil {
			return err
		}
	} else {
		module, err = discovery.Resolve(root, modules, target)
		if err != nil {
			return err
		}
	}

	tags, err := git.Tags(ctx, root)
	if err != nil {
		return err
	}

	latest := release.LatestTag(module, tags)
	resolvedVersion := opts.version
	if resolvedVersion == "" {
		releaseType := opts.typ
		if releaseType == "" {
			releaseType, resolvedVersion, err = prompt.SelectVersionMode()
			if err != nil {
				return err
			}
		}
		if resolvedVersion == "" {
			resolvedVersion, err = nextVersion(releaseType, latest)
		}
		if err != nil {
			return err
		}
	}
	if err := version.Validate(resolvedVersion); err != nil {
		return err
	}

	plan := release.Plan{
		Module:    module,
		Version:   resolvedVersion,
		Tag:       module.TagFor(resolvedVersion),
		LatestTag: latest,
	}
	return release.PrintPlan(out, plan)
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
