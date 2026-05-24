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
	cmd.SetArgs([]string{"--repo-path", "/tmp/repo", "--context-lines", "2"})

	err = cmd.Execute()
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "/tmp/repo", cfg.GetString("repo-path"))
	assert.Equal(t, 2, cfg.GetInt("context-lines"))
}
