package tui

import "ero/internal/adapters/in/tui/component"

func renderHelpPane(width, height int) string {
	return component.RenderHelpPane(width, height, enterKeyLabel(), commentSubmitKeyLabel())
}
