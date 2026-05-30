package ports

import syntaxdto "ero/internal/syntax"

type SemanticTokenType = syntaxdto.SemanticTokenType

const (
	SemanticTokenKeyword     = syntaxdto.SemanticTokenKeyword
	SemanticTokenFunction    = syntaxdto.SemanticTokenFunction
	SemanticTokenTypeName    = syntaxdto.SemanticTokenTypeName
	SemanticTokenName        = syntaxdto.SemanticTokenName
	SemanticTokenString      = syntaxdto.SemanticTokenString
	SemanticTokenNumber      = syntaxdto.SemanticTokenNumber
	SemanticTokenComment     = syntaxdto.SemanticTokenComment
	SemanticTokenOperator    = syntaxdto.SemanticTokenOperator
	SemanticTokenPunctuation = syntaxdto.SemanticTokenPunctuation
	SemanticTokenText        = syntaxdto.SemanticTokenText
)

type SyntaxToken = syntaxdto.Token

type SyntaxTokenizer interface {
	Tokenize(filename string, lines []string) ([][]SyntaxToken, error)
	Language(filename string) string
}
