package ports

import "context"

type ClipboardWriter interface {
	WriteClipboard(ctx context.Context, text string) error
}
