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

	assert.Equal(t, core.SemanticTokenKeyword, SemanticTokenTypeFromChroma("Keyword"))
	assert.Equal(t, core.SemanticTokenFunction, SemanticTokenTypeFromChroma("Name.Function"))
	assert.Equal(t, core.SemanticTokenString, SemanticTokenTypeFromChroma("Literal.String.Double"))
	assert.Equal(t, core.SemanticTokenText, SemanticTokenTypeFromChroma("Text"))
}

func TestTokenizerTokenizeMapsSemanticTypes(t *testing.T) {
	t.Parallel()

	tokenizer := NewTokenizer()
	tokens, err := tokenizer.Tokenize("main.go", []string{"package main"})
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	require.NotEmpty(t, tokens[0])
	assert.Equal(t, core.SemanticTokenKeyword, tokens[0][0].Type)
}

func TestTokenizerTokenizePropagatesLexerErrors(t *testing.T) {
	t.Parallel()

	tokenizer := NewTokenizer()
	tokenizer.lexerCache.Store(".broken", &failingLexer{})

	tokens, err := tokenizer.Tokenize("file.broken", []string{"content"})
	require.Error(t, err)
	require.Len(t, tokens, 1)
	assert.Empty(t, tokens[0])
}
