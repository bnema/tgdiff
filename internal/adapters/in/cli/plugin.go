package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"ero/internal/adapters/in/tui/component"
	pluginadapter "ero/internal/adapters/out/plugin"
)

// PluginManager is the interface the CLI needs from the plugin adapter.
type PluginManager interface {
	Install(ctx context.Context, source string) (pluginadapter.InstallResult, error)
	List(ctx context.Context) ([]pluginadapter.InstalledPlugin, error)
	Update(ctx context.Context, source string) ([]pluginadapter.UpdateResult, error)
	Remove(ctx context.Context, nameOrSource string) (pluginadapter.RemoveResult, error)
}

// NewPluginCommand creates the "plugin" parent command with list, install,
// update, and remove subcommands. Each supports --json for machine-readable
// output.
func NewPluginCommand(manager PluginManager, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage Ero review provider plugins",
		Long: `Manage Ero plugins that provide review provider capabilities.

Plugins can be installed from Git URLs or local Git repositories. The plugin
system stores metadata in your Ero config directory and manages plugin data
under the Ero data directory. Local plugins are tracked by reference; their
repositories are never deleted by Ero.`,
	}

	var jsonOutput bool
	cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	cmd.AddCommand(newPluginListCommand(manager, out, &jsonOutput))
	cmd.AddCommand(newPluginInstallCommand(manager, out, &jsonOutput))
	cmd.AddCommand(newPluginUpdateCommand(manager, out, &jsonOutput))
	cmd.AddCommand(newPluginRemoveCommand(manager, out, &jsonOutput))

	return cmd
}

func newPluginListCommand(manager PluginManager, out io.Writer, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			plugins, err := manager.List(ctx)
			if err != nil {
				return err
			}

			writer := commandOut(cmd, out)
			if *jsonOutput {
				enc := json.NewEncoder(writer)
				enc.SetIndent("", "  ")
				return enc.Encode(plugins)
			}

			renderPluginList(writer, plugins)
			return nil
		},
	}
}

func newPluginInstallCommand(manager PluginManager, out io.Writer, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "install <source>",
		Short: "Install a plugin from a Git URL or local path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			result, err := manager.Install(ctx, args[0])
			if err != nil {
				return err
			}

			writer := commandOut(cmd, out)
			if *jsonOutput {
				enc := json.NewEncoder(writer)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			fmt.Fprintf(writer, "Installed plugin %s v%s\n", result.Name, result.Version)
			fmt.Fprintf(writer, "  Source: %s\n", result.Source)
			fmt.Fprintf(writer, "  Path:   %s\n", result.Path)
			return nil
		},
	}
}

func newPluginUpdateCommand(manager PluginManager, out io.Writer, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "update [source]",
		Short: "Update installed plugins to their latest version",
		Long: `Update plugins to the latest version from their source.

When a source is specified, only that plugin is updated. When no source is
given, all installed plugins are updated. Pinned plugins and local sources
are reported as skipped.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			filter := ""
			if len(args) > 0 {
				filter = args[0]
			}

			results, err := manager.Update(ctx, filter)
			if err != nil {
				return err
			}

			writer := commandOut(cmd, out)
			if *jsonOutput {
				enc := json.NewEncoder(writer)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}

			for _, r := range results {
				if r.Message != "" {
					fmt.Fprintf(writer, "%s: %s\n", r.Name, r.Message)
				} else {
					fmt.Fprintf(writer, "Updated %s: %s → %s\n", r.Name, shortSHA(r.PreviousRef), shortSHA(r.UpdatedRef))
				}
			}
			return nil
		},
	}
}

func newPluginRemoveCommand(manager PluginManager, out io.Writer, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name|source>",
		Short: "Remove an installed plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			result, err := manager.Remove(ctx, args[0])
			if err != nil {
				return err
			}

			writer := commandOut(cmd, out)
			if *jsonOutput {
				enc := json.NewEncoder(writer)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			action := "Removed plugin reference"
			if result.RemovedRepo {
				action = "Removed plugin"
			}
			fmt.Fprintf(writer, "%s %s\n", action, result.Name)
			return nil
		},
	}
}

func commandOut(cmd *cobra.Command, configured io.Writer) io.Writer {
	if configured != nil {
		return configured
	}
	return cmd.OutOrStdout()
}

// renderPluginList writes a human-readable table of installed plugins.
func renderPluginList(out io.Writer, plugins []pluginadapter.InstalledPlugin) {
	items := make([]component.PluginListItem, 0, len(plugins))
	for _, p := range plugins {
		items = append(items, component.PluginListItem{
			Name:          p.Name,
			Version:       p.Version,
			Source:        p.Source,
			Contributions: p.Contributions,
		})
	}
	fmt.Fprintln(out, component.RenderPluginList(items, 100))
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
