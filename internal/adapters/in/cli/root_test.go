package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCommandExecutesRunFuncAndBindsConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		expectRepo    string
		expectContext int
	}{
		{
			name:          "binds repo path and context lines",
			args:          []string{"--repo-path", "/tmp/repo", "--context-lines", "2"},
			expectRepo:    "/tmp/repo",
			expectContext: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := viper.New()
			called := false

			cmd, err := NewRootCommand(cfg, func() error {
				called = true
				return nil
			})
			require.NoError(t, err)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(tt.args)

			err = cmd.Execute()
			require.NoError(t, err)
			assert.True(t, called)
			assert.Equal(t, tt.expectRepo, cfg.GetString("repo-path"))
			assert.Equal(t, tt.expectContext, cfg.GetInt("context-lines"))
		})
	}
}
