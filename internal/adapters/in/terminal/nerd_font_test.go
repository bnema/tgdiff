package terminal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectNerdFontFromEnvDefaultsToEnabled(t *testing.T) {
	t.Parallel()

	assert.True(t, DetectNerdFontFromEnv(nil))
}

func TestDetectNerdFontFromEnvSupportsExplicitOverrides(t *testing.T) {
	t.Parallel()

	assert.True(t, DetectNerdFontFromEnv([]string{"TGDIFF_NERD_FONT=1"}))
	assert.True(t, DetectNerdFontFromEnv([]string{"NERD_FONT=true"}))
	assert.False(t, DetectNerdFontFromEnv([]string{"TGDIFF_NERD_FONT=0"}))
	assert.False(t, DetectNerdFontFromEnv([]string{"NERD_FONT=false"}))
}
