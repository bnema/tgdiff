package core

import "tgdiff/internal/ports"

type SemanticTokenType = ports.SemanticTokenType

const (
	SemanticTokenKeyword     = ports.SemanticTokenKeyword
	SemanticTokenFunction    = ports.SemanticTokenFunction
	SemanticTokenTypeName    = ports.SemanticTokenTypeName
	SemanticTokenName        = ports.SemanticTokenName
	SemanticTokenString      = ports.SemanticTokenString
	SemanticTokenNumber      = ports.SemanticTokenNumber
	SemanticTokenComment     = ports.SemanticTokenComment
	SemanticTokenOperator    = ports.SemanticTokenOperator
	SemanticTokenPunctuation = ports.SemanticTokenPunctuation
	SemanticTokenText        = ports.SemanticTokenText
)

type SyntaxToken = ports.SyntaxToken
