package git

import (
	"testing"

	ggit "github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/require"
)

func TestRepositoryLoaderOpenExistingRepository(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "opens initialized repository"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			_, err := ggit.PlainInit(dir, false)
			require.NoError(t, err)

			loader := NewRepositoryLoader()
			repo, err := loader.Open(dir)
			require.NoError(t, err)
			require.NotNil(t, repo)
		})
	}
}
