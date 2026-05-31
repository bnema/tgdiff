package ports

import "ero/internal/core"

type SemanticTokenType = core.SemanticTokenType

const (
	SemanticTokenKeyword     = core.SemanticTokenKeyword
	SemanticTokenFunction    = core.SemanticTokenFunction
	SemanticTokenTypeName    = core.SemanticTokenTypeName
	SemanticTokenName        = core.SemanticTokenName
	SemanticTokenString      = core.SemanticTokenString
	SemanticTokenNumber      = core.SemanticTokenNumber
	SemanticTokenComment     = core.SemanticTokenComment
	SemanticTokenOperator    = core.SemanticTokenOperator
	SemanticTokenPunctuation = core.SemanticTokenPunctuation
	SemanticTokenText        = core.SemanticTokenText
)

type SyntaxToken = core.SyntaxToken

type SyntaxTokenizer interface {
	Tokenize(filename string, lines []string) ([][]SyntaxToken, error)
	Language(filename string) string
}
