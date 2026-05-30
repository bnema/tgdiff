package component

import (
	"fmt"
	"strings"

	"ero/internal/adapters/in/tui/theme"
)

// PluginListItem holds the data for one row in a plugin list.
type PluginListItem struct {
	Name          string
	Version       string
	Source        string
	Status        string
	Contributions []string
}

// RenderPluginList renders a list of plugins using the existing theme styles.
// Human-readable only; JSON output should use the CLI's json.Encoder path.
// The returned string does not include ANSI codes beyond what theme styles
// already use.
func RenderPluginList(items []PluginListItem, width int) string {
	if len(items) == 0 {
		return theme.MutedStyle.Render("No plugins installed.")
	}

	var b strings.Builder

	for i, item := range items {
		if i > 0 {
			b.WriteByte('\n')
		}

		// Plugin name + version in bold.
		header := fmt.Sprintf("%s v%s", item.Name, item.Version)
		if item.Status != "" {
			header += " " + theme.MutedStyle.Render("("+item.Status+")")
		}
		b.WriteString(theme.StatusAppStyle.Render(TruncateRunes(header, max(width, 20))))

		// Contributions line.
		if len(item.Contributions) > 0 {
			b.WriteByte('\n')
			contribs := strings.Join(item.Contributions, ", ")
			b.WriteString(theme.StatusInfoStyle.Render("  ↳ " + TruncateRunes(contribs, max(width-4, 10))))
		}

		// Source line.
		b.WriteByte('\n')
		sourceLabel := "  Source: " + item.Source
		b.WriteString(theme.MutedStyle.Render(TruncateRunes(sourceLabel, max(width, 20))))
	}

	return b.String()
}
