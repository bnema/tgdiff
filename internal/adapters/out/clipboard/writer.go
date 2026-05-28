package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type SystemWriter struct{}

func NewSystemWriter() SystemWriter {
	return SystemWriter{}
}

func (w SystemWriter) WriteClipboard(ctx context.Context, text string) error {
	command, ok := w.selectCommand()
	if !ok {
		return fmt.Errorf("no supported clipboard tool found; install wl-copy for Wayland or xclip/xsel for X11")
	}
	if err := runCommand(ctx, command.name, command.args, text); err != nil {
		return fmt.Errorf("copy to clipboard with %s: %w", command.name, err)
	}
	return nil
}

type clipboardCommand struct {
	name string
	args []string
}

func (w SystemWriter) selectCommand() (clipboardCommand, bool) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if path, ok := w.findCommand("wl-copy"); ok {
			return clipboardCommand{name: path}, true
		}
	}
	if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
		if path, ok := w.findCommand("xclip"); ok {
			return clipboardCommand{name: path, args: []string{"-selection", "clipboard"}}, true
		}
		if path, ok := w.findCommand("xsel"); ok {
			return clipboardCommand{name: path, args: []string{"--clipboard", "--input"}}, true
		}
	}
	return clipboardCommand{}, false
}

func (w SystemWriter) findCommand(name string) (string, bool) {
	path, err := exec.LookPath(name)
	return path, err == nil
}

func runCommand(ctx context.Context, name string, args []string, input string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewBufferString(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return err
	}
	return nil
}
