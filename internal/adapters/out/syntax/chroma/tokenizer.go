package chroma

import (
	"path/filepath"
	"strings"
	"sync"

	basechroma "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"

	"tgdiff/internal/core"
)

type Tokenizer struct {
	lexerCache sync.Map
}

func NewTokenizer() *Tokenizer {
	return &Tokenizer{}
}

func (t *Tokenizer) Language(filename string) string {
	lexer := t.lexer(filename)
	if cfg := lexer.Config(); cfg != nil {
		return cfg.Name
	}
	return ""
}

func (t *Tokenizer) Tokenize(filename string, lines []string) ([][]core.SyntaxToken, error) {
	result := make([][]core.SyntaxToken, len(lines))
	if len(lines) == 0 {
		return result, nil
	}

	iterator, err := t.lexer(filename).Tokenise(nil, strings.Join(lines, "\n"))
	if err != nil {
		return result, err
	}

	lineIndex := 0
	lineOffset := 0

	for token := iterator(); token != basechroma.EOF; token = iterator() {
		value := token.Value
		semanticType := SemanticTokenTypeFromChroma(token.Type.String())

		for len(value) > 0 {
			if lineIndex >= len(lines) {
				break
			}

			newlineIndex := strings.Index(value, "\n")
			if newlineIndex == -1 {
				if len(value) > 0 {
					runeLen := len([]rune(value))
					result[lineIndex] = append(result[lineIndex], core.SyntaxToken{
						Start: lineOffset,
						End:   lineOffset + runeLen,
						Type:  semanticType,
					})
					lineOffset += runeLen
				}
				break
			}

			beforeNewline := value[:newlineIndex]
			if len(beforeNewline) > 0 {
				runeLen := len([]rune(beforeNewline))
				result[lineIndex] = append(result[lineIndex], core.SyntaxToken{
					Start: lineOffset,
					End:   lineOffset + runeLen,
					Type:  semanticType,
				})
			}

			lineIndex++
			lineOffset = 0
			value = value[newlineIndex+1:]
		}
	}

	return result, nil
}

func SemanticTokenTypeFromChroma(tokenType string) core.SemanticTokenType {
	switch {
	case strings.HasPrefix(tokenType, "Keyword"):
		return core.SemanticTokenKeyword
	case strings.HasPrefix(tokenType, "Name.Function"):
		return core.SemanticTokenFunction
	case strings.HasPrefix(tokenType, "Name.Class"), strings.HasPrefix(tokenType, "Name.Builtin"):
		return core.SemanticTokenTypeName
	case strings.HasPrefix(tokenType, "Name"):
		return core.SemanticTokenName
	case strings.HasPrefix(tokenType, "Literal.String"):
		return core.SemanticTokenString
	case strings.HasPrefix(tokenType, "Literal.Number"):
		return core.SemanticTokenNumber
	case strings.HasPrefix(tokenType, "Comment"):
		return core.SemanticTokenComment
	case strings.HasPrefix(tokenType, "Operator"):
		return core.SemanticTokenOperator
	case strings.HasPrefix(tokenType, "Punctuation"):
		return core.SemanticTokenPunctuation
	default:
		return core.SemanticTokenText
	}
}

func (t *Tokenizer) lexer(filename string) basechroma.Lexer {
	cacheKey := strings.ToLower(filepath.Ext(filename))
	if cacheKey == "" {
		cacheKey = strings.ToLower(filepath.Base(filename))
	}

	if cached, ok := t.lexerCache.Load(cacheKey); ok {
		return cached.(basechroma.Lexer)
	}

	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	lexer = basechroma.Coalesce(lexer)
	t.lexerCache.Store(cacheKey, lexer)
	return lexer
}
