package terminal

import (
	"os"

	"golang.org/x/term"
)

func IsInteractive() bool {
	return isTerminal(os.Stdin) && isTerminal(os.Stdout)
}

func isTerminal(file *os.File) bool {
	return term.IsTerminal(int(file.Fd()))
}
