package core

type SemanticTokenType string

const (
	SemanticTokenKeyword     SemanticTokenType = "keyword"
	SemanticTokenFunction    SemanticTokenType = "function"
	SemanticTokenTypeName    SemanticTokenType = "type"
	SemanticTokenName        SemanticTokenType = "name"
	SemanticTokenString      SemanticTokenType = "string"
	SemanticTokenNumber      SemanticTokenType = "number"
	SemanticTokenComment     SemanticTokenType = "comment"
	SemanticTokenOperator    SemanticTokenType = "operator"
	SemanticTokenPunctuation SemanticTokenType = "punctuation"
	SemanticTokenText        SemanticTokenType = "text"
)

type SyntaxToken struct {
	Start      int
	End        int
	Type       SemanticTokenType
	SourceType string
}
