package tui

const (
	nerdIconArrowRight     = "\uf061"
	nerdIconComment        = "\uf075"
	nerdIconKeyboardReturn = "\U000f0311"
)

func enterKeyLabel() string {
	return nerdIconKeyboardReturn
}

func commentSubmitKeyLabel() string {
	return "ctrl+s/" + enterKeyLabel()
}
