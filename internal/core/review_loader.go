package core

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type BaseBranchResolver interface {
	ResolveBaseBranch(repoPath string) (string, error)
}

type GitDiffLoader interface {
	LoadBranchDiff(repoPath, baseBranch string) (string, error)
}

type SyntaxTokenizer interface {
	Tokenize(filename string, lines []string) ([][]SyntaxToken, error)
}

type FileContentReader interface {
	ReadFileLines(repoPath, path string) ([]string, error)
}

type ReviewLoader struct {
	baseBranchResolver BaseBranchResolver
	diffLoader         GitDiffLoader
	syntaxTokenizer    SyntaxTokenizer
	fileContentReader  FileContentReader
}

func NewReviewLoader(baseBranchResolver BaseBranchResolver, diffLoader GitDiffLoader, syntaxTokenizer SyntaxTokenizer, fileContentReaders ...FileContentReader) *ReviewLoader {
	loader := &ReviewLoader{
		baseBranchResolver: baseBranchResolver,
		diffLoader:         diffLoader,
		syntaxTokenizer:    syntaxTokenizer,
	}
	if len(fileContentReaders) > 0 {
		loader.fileContentReader = fileContentReaders[0]
	}
	return loader
}

func (l *ReviewLoader) Load(repoPath string, contextWindow int) ([]ReviewFile, error) {
	if l == nil {
		return nil, fmt.Errorf("review loader is nil")
	}
	if l.baseBranchResolver == nil {
		return nil, fmt.Errorf("base branch resolver is nil")
	}
	if l.diffLoader == nil {
		return nil, fmt.Errorf("diff loader is nil")
	}

	baseBranch, err := l.baseBranchResolver.ResolveBaseBranch(repoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve base branch: %w", err)
	}

	diff, err := l.diffLoader.LoadBranchDiff(repoPath, baseBranch)
	if err != nil {
		return nil, fmt.Errorf("load branch diff: %w", err)
	}

	var files []ReviewFile
	if l.fileContentReader != nil {
		files, err = ParseUnifiedDiffWithFileContent(repoPath, diff, contextWindow, l.fileContentReader)
	} else {
		files, err = ParseUnifiedDiff(diff, contextWindow)
	}
	if err != nil {
		return nil, err
	}

	if l.syntaxTokenizer != nil {
		for i := range files {
			if err := applySyntaxTokens(&files[i], l.syntaxTokenizer); err != nil {
				return nil, err
			}
		}
	}

	return files, nil
}

var unifiedHunkHeaderPattern = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func ParseUnifiedDiff(diff string, contextWindow int) ([]ReviewFile, error) {
	return parseUnifiedDiff(diff, contextWindow, nil)
}

func ParseUnifiedDiffWithFileContent(repoPath, diff string, contextWindow int, reader FileContentReader) ([]ReviewFile, error) {
	if reader == nil {
		return ParseUnifiedDiff(diff, contextWindow)
	}
	return parseUnifiedDiff(diff, contextWindow, func(path string) ([]string, error) {
		return reader.ReadFileLines(repoPath, path)
	})
}

func parseUnifiedDiff(diff string, contextWindow int, readFileLines func(path string) ([]string, error)) ([]ReviewFile, error) {
	var files []ReviewFile
	var currentPath string
	var currentLines []ReviewLine
	var fileLines []string
	oldLine := 1
	newLine := 1
	inHunk := false

	loadFileLines := func() ([]string, error) {
		if readFileLines == nil || currentPath == "" || fileLines != nil {
			return fileLines, nil
		}
		lines, err := readFileLines(currentPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fileLines = []string{}
				return fileLines, nil
			}
			return nil, err
		}
		fileLines = lines
		return fileLines, nil
	}

	appendUnchangedGap := func(nextOldLine, nextNewLine int) error {
		lines, err := loadFileLines()
		if err != nil {
			return err
		}
		countOld := nextOldLine - oldLine
		countNew := nextNewLine - newLine
		if len(lines) == 0 || countOld <= 0 || countNew <= 0 || countOld != countNew {
			oldLine = nextOldLine
			newLine = nextNewLine
			return nil
		}
		for offset := 0; offset < countNew && newLine+offset-1 < len(lines); offset++ {
			currentLines = append(currentLines, ReviewLine{
				OldLineNumber: oldLine + offset,
				NewLineNumber: newLine + offset,
				Content:       lines[newLine+offset-1],
				Kind:          LineKindUnchanged,
			})
		}
		oldLine = nextOldLine
		newLine = nextNewLine
		return nil
	}

	flushFile := func() error {
		if currentPath == "" {
			return nil
		}
		if lines, err := loadFileLines(); err != nil {
			return err
		} else if len(lines) > 0 && newLine <= len(lines) {
			if err := appendUnchangedGap(oldLine+len(lines)-newLine+1, len(lines)+1); err != nil {
				return err
			}
		}
		if len(currentLines) > 0 {
			files = append(files, BuildReviewFile(currentPath, currentLines, contextWindow))
		}
		currentPath = ""
		currentLines = nil
		fileLines = nil
		oldLine = 1
		newLine = 1
		inHunk = false
		return nil
	}

	for rawLine := range strings.SplitSeq(strings.ReplaceAll(diff, "\r\n", "\n"), "\n") {
		switch {
		case strings.HasPrefix(rawLine, "diff --git "):
			if err := flushFile(); err != nil {
				return nil, err
			}
			currentPath = parseDiffPath(rawLine)
		case strings.HasPrefix(rawLine, "@@ "):
			nextOldLine, nextNewLine, err := parseUnifiedHunkHeader(rawLine)
			if err != nil {
				return nil, err
			}
			if err := appendUnchangedGap(nextOldLine, nextNewLine); err != nil {
				return nil, err
			}
			oldLine = nextOldLine
			newLine = nextNewLine
			inHunk = true
		case strings.HasPrefix(rawLine, "+++ ") || strings.HasPrefix(rawLine, "--- ") || strings.HasPrefix(rawLine, "index "):
			continue
		case rawLine == `\ No newline at end of file`:
			continue
		default:
			if !inHunk || rawLine == "" {
				continue
			}

			switch rawLine[0] {
			case ' ':
				currentLines = append(currentLines, ReviewLine{
					OldLineNumber: oldLine,
					NewLineNumber: newLine,
					Content:       rawLine[1:],
					Kind:          LineKindUnchanged,
				})
				oldLine++
				newLine++
			case '+':
				currentLines = append(currentLines, ReviewLine{
					NewLineNumber: newLine,
					Content:       rawLine[1:],
					Kind:          LineKindAdded,
				})
				newLine++
			case '-':
				currentLines = append(currentLines, ReviewLine{
					OldLineNumber: oldLine,
					Content:       rawLine[1:],
					Kind:          LineKindDeleted,
				})
				oldLine++
			}
		}
	}

	if err := flushFile(); err != nil {
		return nil, err
	}
	return files, nil
}

func parseUnifiedHunkHeader(header string) (int, int, error) {
	match := unifiedHunkHeaderPattern.FindStringSubmatch(header)
	if len(match) != 3 {
		return 0, 0, fmt.Errorf("parse hunk header %q", header)
	}

	oldLine, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse old line in hunk header %q: %w", header, err)
	}
	newLine, err := strconv.Atoi(match[2])
	if err != nil {
		return 0, 0, fmt.Errorf("parse new line in hunk header %q: %w", header, err)
	}
	return oldLine, newLine, nil
}

func parseDiffPath(header string) string {
	parts := strings.Fields(header)
	if len(parts) < 4 {
		return ""
	}

	newPath := stripDiffPathPrefix(parts[3])
	if newPath != "/dev/null" {
		return newPath
	}
	return stripDiffPathPrefix(parts[2])
}

func stripDiffPathPrefix(path string) string {
	for _, prefix := range []string{"a/", "b/"} {
		if trimmed, ok := strings.CutPrefix(path, prefix); ok {
			return trimmed
		}
	}
	return path
}

func applySyntaxTokens(file *ReviewFile, tokenizer SyntaxTokenizer) error {
	lineContents := make([]string, 0)
	for _, section := range file.Sections {
		for _, line := range section.Lines {
			lineContents = append(lineContents, line.Content)
		}
	}
	if len(lineContents) == 0 {
		return nil
	}

	tokensByLine, err := tokenizer.Tokenize(file.Path, lineContents)
	if err != nil {
		return fmt.Errorf("tokenize %s: %w", file.Path, err)
	}
	if len(tokensByLine) != len(lineContents) {
		return fmt.Errorf("tokenize %s: expected %d token rows, got %d", file.Path, len(lineContents), len(tokensByLine))
	}

	lineIndex := 0
	for i := range file.Sections {
		for j := range file.Sections[i].Lines {
			file.Sections[i].Lines[j].SyntaxTokens = tokensByLine[lineIndex]
			lineIndex++
		}
	}

	return nil
}
