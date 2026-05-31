package core

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type reviewBaseBranchResolver interface {
	ResolveBaseBranch(repoPath string) (string, error)
}

type reviewGitDiffLoader interface {
	LoadBranchDiff(repoPath, baseBranch string) (string, error)
	LoadWorkingTreeDiff(repoPath string) (string, error)
	LoadStagedDiff(repoPath string) (string, error)
	LoadLocalDiff(repoPath string) (string, error)
	LoadUpstreamDiff(repoPath, upstreamRef string) (string, error)
	LoadCommitDiff(repoPath, revision string) (string, error)
	LoadRangeDiff(repoPath, baseRevision, headRevision string) (string, error)
}

type reviewSyntaxTokenizer interface {
	Tokenize(filename string, lines []string) ([][]SyntaxToken, error)
	Language(filename string) string
}

type reviewFileContentReader interface {
	ReadFileLines(repoPath, path string) ([]string, error)
}

type ReviewLoader struct {
	baseBranchResolver reviewBaseBranchResolver
	diffLoader         reviewGitDiffLoader
	syntaxTokenizer    reviewSyntaxTokenizer
	fileContentReader  reviewFileContentReader
}

func NewReviewLoader(baseBranchResolver reviewBaseBranchResolver, diffLoader reviewGitDiffLoader, syntaxTokenizer reviewSyntaxTokenizer, fileContentReaders ...reviewFileContentReader) *ReviewLoader {
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
	return l.LoadReview(ReviewRequest{RepoPath: repoPath, ContextLines: contextWindow, DiffMode: DiffModeBranch})
}

// LoadReview loads review files for the requested repository, context window, and diff mode.
func (l *ReviewLoader) LoadReview(request ReviewRequest) ([]ReviewFile, error) {
	if l == nil {
		return nil, fmt.Errorf("review loader is nil")
	}
	if l.baseBranchResolver == nil {
		return nil, fmt.Errorf("base branch resolver is nil")
	}
	if l.diffLoader == nil {
		return nil, fmt.Errorf("diff loader is nil")
	}

	diff, err := l.loadDiff(request)
	if err != nil {
		return nil, err
	}

	var files []ReviewFile
	if l.fileContentReader != nil {
		files, err = ParseUnifiedDiffWithFileContent(request.RepoPath, diff, request.ContextLines, l.fileContentReader)
	} else {
		files, err = ParseUnifiedDiff(diff, request.ContextLines)
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

func (l *ReviewLoader) loadDiff(request ReviewRequest) (string, error) {
	mode := request.modeOrDefault()
	if !mode.IsValid() {
		return "", fmt.Errorf("invalid diff mode %q", request.DiffMode)
	}

	switch mode {
	case DiffModeBranch:
		baseBranch, err := l.baseBranchResolver.ResolveBaseBranch(request.RepoPath)
		if err != nil {
			return "", fmt.Errorf("resolve base branch: %w", err)
		}
		diff, err := l.diffLoader.LoadBranchDiff(request.RepoPath, baseBranch)
		if err != nil {
			return "", fmt.Errorf("load branch diff: %w", err)
		}
		return diff, nil
	case DiffModeWorking:
		return l.diffLoader.LoadWorkingTreeDiff(request.RepoPath)
	case DiffModeStaged:
		return l.diffLoader.LoadStagedDiff(request.RepoPath)
	case DiffModeLocal:
		return l.diffLoader.LoadLocalDiff(request.RepoPath)
	case DiffModeUpstream:
		upstreamRef := request.UpstreamRef
		if upstreamRef == "" {
			upstreamRef = "@{upstream}"
		}
		return l.diffLoader.LoadUpstreamDiff(request.RepoPath, upstreamRef)
	case DiffModeCommit:
		revision := request.Revision
		if revision == "" {
			revision = "HEAD"
		}
		return l.diffLoader.LoadCommitDiff(request.RepoPath, revision)
	case DiffModeRange:
		baseRevision := request.BaseRevision
		if baseRevision == "" {
			var err error
			baseRevision, err = l.baseBranchResolver.ResolveBaseBranch(request.RepoPath)
			if err != nil {
				return "", fmt.Errorf("resolve base branch: %w", err)
			}
		}
		headRevision := request.HeadRevision
		if headRevision == "" {
			headRevision = "HEAD"
		}
		return l.diffLoader.LoadRangeDiff(request.RepoPath, baseRevision, headRevision)
	default:
		return "", fmt.Errorf("unsupported diff mode %q", mode)
	}
}

var unifiedHunkHeaderPattern = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func ParseUnifiedDiff(diff string, contextWindow int) ([]ReviewFile, error) {
	return parseUnifiedDiff(diff, contextWindow, nil)
}

func ParseUnifiedDiffWithFileContent(repoPath, diff string, contextWindow int, reader reviewFileContentReader) ([]ReviewFile, error) {
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
	var currentOldPath string
	currentStatus := ReviewFileStatusModified
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
			files = append(files, BuildReviewFileWithMetadata(currentPath, currentOldPath, currentStatus, currentLines, contextWindow))
		}
		currentPath = ""
		currentOldPath = ""
		currentStatus = ReviewFileStatusModified
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
			currentOldPath = parseOldDiffPath(rawLine)
			currentStatus = ReviewFileStatusModified
		case strings.HasPrefix(rawLine, "new file mode "):
			currentStatus = ReviewFileStatusAdded
			currentOldPath = ""
		case strings.HasPrefix(rawLine, "deleted file mode "):
			currentStatus = ReviewFileStatusDeleted
		case strings.HasPrefix(rawLine, "rename from "):
			currentStatus = ReviewFileStatusRenamed
			currentOldPath = strings.TrimPrefix(rawLine, "rename from ")
		case strings.HasPrefix(rawLine, "rename to "):
			currentStatus = ReviewFileStatusRenamed
			currentPath = strings.TrimPrefix(rawLine, "rename to ")
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

func parseOldDiffPath(header string) string {
	parts := strings.Fields(header)
	if len(parts) < 4 {
		return ""
	}
	oldPath := stripDiffPathPrefix(parts[2])
	if oldPath == "/dev/null" {
		return ""
	}
	return oldPath
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

func applySyntaxTokens(file *ReviewFile, tokenizer reviewSyntaxTokenizer) error {
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
