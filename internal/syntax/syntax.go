package syntax

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

// Token represents a lexical token with position and semantic classification.
// Type is the normalized SemanticTokenType used by core rendering decisions.
// SourceType preserves the original lexer token kind for adapters that can use
// source-specific detail, such as fine-grained syntax highlighting.
type Token struct {
	Start      int
	End        int
	Type       SemanticTokenType
	SourceType string
}
