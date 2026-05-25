package terminal

import (
	"os"
	"strings"
)

type Capabilities struct{}

func NewCapabilities() Capabilities {
	return Capabilities{}
}

func (Capabilities) SupportsNerdFont() bool {
	return DetectNerdFontFromEnv(os.Environ())
}

func DetectNerdFont() bool {
	return NewCapabilities().SupportsNerdFont()
}

func DetectNerdFontFromEnv(env []string) bool {
	values := map[string]string{}
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}

	if value, ok := values["TGDIFF_NERD_FONT"]; ok {
		return truthy(value)
	}
	if value, ok := values["NERD_FONT"]; ok {
		return truthy(value)
	}

	return true
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
