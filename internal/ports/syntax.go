package ports

import "tgdiff/internal/core"

type SyntaxTokenizer interface {
	Tokenize(filename string, lines []string) ([][]core.SyntaxToken, error)
	Language(filename string) string
}
