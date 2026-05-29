package keymap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReviewAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want Action
	}{
		{name: "quit", key: "q", want: ActionQuit},
		{name: "move down", key: "j", want: ActionMoveDown},
		{name: "open file search", key: "f", want: ActionOpenFileSearch},
		{name: "open grep search", key: "/", want: ActionOpenGrepSearch},
		{name: "open help", key: "?", want: ActionOpenHelp},
		{name: "unknown", key: "x", want: ActionNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, ReviewAction(tt.key))
		})
	}
}
