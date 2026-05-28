package clipboard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemWriterUsesPlatformClipboardCommands(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		tools     []string
		wantTool  string
		wantInput string
		wantErr   string
	}{
		{
			name:      "wayland uses wl-copy when available",
			env:       map[string]string{"WAYLAND_DISPLAY": "wayland-1", "DISPLAY": ":0"},
			tools:     []string{"wl-copy", "xclip"},
			wantTool:  "wl-copy",
			wantInput: "hello",
		},
		{
			name:      "wayland falls back to xclip when wl-copy is unavailable",
			env:       map[string]string{"WAYLAND_DISPLAY": "wayland-1", "DISPLAY": ":0"},
			tools:     []string{"xclip"},
			wantTool:  "xclip",
			wantInput: "hello",
		},
		{
			name:      "x11 uses xclip when available",
			env:       map[string]string{"DISPLAY": ":0"},
			tools:     []string{"xclip"},
			wantTool:  "xclip",
			wantInput: "hello",
		},
		{
			name:      "x11 falls back to xsel when xclip is unavailable",
			env:       map[string]string{"DISPLAY": ":0"},
			tools:     []string{"xsel"},
			wantTool:  "xsel",
			wantInput: "hello",
		},
		{
			name:    "returns actionable error when no clipboard tool is available",
			env:     map[string]string{"WAYLAND_DISPLAY": "wayland-1", "DISPLAY": ":0"},
			tools:   []string{},
			wantErr: "no supported clipboard tool found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binDir := t.TempDir()
			outDir := t.TempDir()
			for _, tool := range tt.tools {
				writeClipboardTool(t, binDir, outDir, tool)
			}
			t.Setenv("PATH", binDir)
			t.Setenv("WAYLAND_DISPLAY", tt.env["WAYLAND_DISPLAY"])
			t.Setenv("DISPLAY", tt.env["DISPLAY"])

			writer := NewSystemWriter()
			err := writer.WriteClipboard(t.Context(), "hello")
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			got, err := os.ReadFile(filepath.Join(outDir, tt.wantTool))
			require.NoError(t, err)
			assert.Equal(t, tt.wantInput, string(got))
		})
	}
}

func TestSystemWriterIncludesRuntimeFailureStderr(t *testing.T) {
	binDir := t.TempDir()
	writeFailingClipboardTool(t, binDir, "wl-copy", "permission denied")
	t.Setenv("PATH", binDir)
	t.Setenv("WAYLAND_DISPLAY", "wayland-1")
	t.Setenv("DISPLAY", "")

	err := NewSystemWriter().WriteClipboard(t.Context(), "hello")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "copy to clipboard with")
	assert.Contains(t, err.Error(), "permission denied")
}

func writeClipboardTool(t *testing.T, binDir, outDir, name string) {
	t.Helper()
	script := "#!/bin/sh\n/bin/cat > " + filepath.Join(outDir, name) + "\n"
	err := os.WriteFile(filepath.Join(binDir, name), []byte(script), 0o755)
	require.NoError(t, err)
}

func writeFailingClipboardTool(t *testing.T, binDir, name, stderr string) {
	t.Helper()
	script := "#!/bin/sh\necho " + shellQuote(stderr) + " >&2\nexit 1\n"
	err := os.WriteFile(filepath.Join(binDir, name), []byte(script), 0o755)
	require.NoError(t, err)
}

func shellQuote(value string) string {
	return "'" + value + "'"
}
