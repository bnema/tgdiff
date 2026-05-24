package chroma

import (
	"errors"
	"testing"

	basechroma "github.com/alecthomas/chroma/v2"

	"tgdiff/internal/core"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingLexer struct{}

func (f *failingLexer) Config() *basechroma.Config {
	return &basechroma.Config{Name: "broken"}
}

func (f *failingLexer) Tokenise(*basechroma.TokeniseOptions, string) (basechroma.Iterator, error) {
	return nil, errors.New("boom")
}

func (f *failingLexer) SetRegistry(*basechroma.LexerRegistry) basechroma.Lexer {
	return f
}

func (f *failingLexer) SetAnalyser(func(string) float32) basechroma.Lexer {
	return f
}

func (f *failingLexer) AnalyseText(string) float32 {
	return 0
}

func TestSemanticTokenTypeFromChroma(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		chroma   string
		expected core.SemanticTokenType
	}{
		{name: "keyword", chroma: "Keyword", expected: core.SemanticTokenKeyword},
		{name: "function", chroma: "Name.Function", expected: core.SemanticTokenFunction},
		{name: "string", chroma: "Literal.String.Double", expected: core.SemanticTokenString},
		{name: "text", chroma: "Text", expected: core.SemanticTokenText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, SemanticTokenTypeFromChroma(tt.chroma))
		})
	}
}

func TestTokenizerTokenizeMapsSemanticTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		filename     string
		lines        []string
		expectedType core.SemanticTokenType
	}{
		{name: "go package keyword", filename: "main.go", lines: []string{"package main"}, expectedType: core.SemanticTokenKeyword},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tokenizer := NewTokenizer()
			tokens, err := tokenizer.Tokenize(tt.filename, tt.lines)
			require.NoError(t, err)
			require.Len(t, tokens, 1)
			require.NotEmpty(t, tokens[0])
			assert.Equal(t, tt.expectedType, tokens[0][0].Type)
		})
	}
}

func TestTokenizerTokenizePreservesFullChromaTokenType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		filename      string
		lines         []string
		expectedTypes []string
	}{
		{
			name:          "go function declaration",
			filename:      "main.go",
			lines:         []string{"func main() {", "\treturn", "}"},
			expectedTypes: []string{"KeywordDeclaration", "NameFunction"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tokenizer := NewTokenizer()
			tokens, err := tokenizer.Tokenize(tt.filename, tt.lines)
			require.NoError(t, err)
			require.Len(t, tokens, len(tt.lines))
			require.NotEmpty(t, tokens[0])

			var tokenTypes []string
			for _, token := range tokens[0] {
				tokenTypes = append(tokenTypes, token.ChromaType)
			}
			for _, expectedType := range tt.expectedTypes {
				assert.Contains(t, tokenTypes, expectedType)
			}
		})
	}
}

func TestTokenizerTokenizePropagatesLexerErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		lines    []string
	}{
		{name: "lexer error", filename: "file.broken", lines: []string{"content"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tokenizer := NewTokenizer()
			tokenizer.lexerCache.Store(".broken", &failingLexer{})

			tokens, err := tokenizer.Tokenize(tt.filename, tt.lines)
			require.Error(t, err)
			require.Len(t, tokens, len(tt.lines))
			assert.Empty(t, tokens[0])
		})
	}
}
