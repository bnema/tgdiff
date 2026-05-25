package tui

import (
	"path/filepath"
	"sort"
	"strings"

	"tgdiff/internal/core"
)

type SearchResult struct {
	FileIndex    int
	SectionIndex int
	LineIndex    int
	Path         string
	LineNumber   int
	Preview      string
	DiffMode     core.DiffMode
}

type scoredSearchResult struct {
	result SearchResult
	score  int
}

func fuzzyFileResults(files []core.ReviewFile, query string) []SearchResult {
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		results := make([]SearchResult, 0, len(files))
		for fileIndex, file := range files {
			results = append(results, SearchResult{FileIndex: fileIndex, Path: file.Path})
		}
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].Path < results[j].Path
		})
		return results
	}

	scoredResults := make([]scoredSearchResult, 0, len(files))
	for fileIndex, file := range files {
		if score, ok := fuzzyScore(file.Path, trimmedQuery); ok {
			scoredResults = append(scoredResults, scoredSearchResult{
				result: SearchResult{FileIndex: fileIndex, Path: file.Path},
				score:  score,
			})
		}
	}
	sort.SliceStable(scoredResults, func(i, j int) bool {
		if scoredResults[i].score != scoredResults[j].score {
			return scoredResults[i].score < scoredResults[j].score
		}
		return scoredResults[i].result.Path < scoredResults[j].result.Path
	})

	results := make([]SearchResult, 0, len(scoredResults))
	for _, scoredResult := range scoredResults {
		results = append(results, scoredResult.result)
	}
	return results
}

// grepResults searches all review lines, including hidden context lines. Jumping to
// a hidden-context result expands that context section before resolving its anchor.
func grepResults(files []core.ReviewFile, query string) []SearchResult {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	needle := strings.ToLower(query)
	var results []SearchResult
	for fileIndex, file := range files {
		for sectionIndex, section := range file.Sections {
			for lineIndex, line := range section.Lines {
				if !strings.Contains(strings.ToLower(line.Content), needle) {
					continue
				}
				results = append(results, SearchResult{
					FileIndex:    fileIndex,
					SectionIndex: sectionIndex,
					LineIndex:    lineIndex,
					Path:         file.Path,
					LineNumber:   displayLineNumber(line),
					Preview:      strings.TrimSpace(line.Content),
				})
			}
		}
	}
	return results
}

func diffModeResults(query string, nerdFont bool) []SearchResult {
	query = strings.TrimSpace(query)
	results := make([]SearchResult, 0, len(selectableDiffModes))
	for _, mode := range selectableDiffModes {
		label := diffModeLabel(mode, nerdFont)
		if query != "" {
			plain := diffModePlainLabel(mode)
			if _, ok := fuzzyScore(plain, query); !ok && !strings.Contains(strings.ToLower(label), strings.ToLower(query)) {
				continue
			}
		}
		results = append(results, SearchResult{Path: label, DiffMode: mode})
	}
	return results
}

func displayLineNumber(line core.ReviewLine) int {
	if line.NewLineNumber > 0 {
		return line.NewLineNumber
	}
	return line.OldLineNumber
}

func fuzzyScore(path, query string) (int, bool) {
	haystack := strings.ToLower(path)
	needle := strings.ToLower(query)
	last := -1
	score := 0
	for _, r := range needle {
		idx := strings.IndexRune(haystack[last+1:], r)
		if idx < 0 {
			return 0, false
		}
		pos := last + 1 + idx
		gap := pos - last - 1
		score += gap*2 + pos
		last = pos
	}

	base := strings.ToLower(filepath.Base(path))
	switch {
	case base == needle:
		score -= 1000
	case strings.HasPrefix(base, needle):
		score -= 500
	case strings.Contains(base, needle):
		score -= 200
	case strings.HasPrefix(haystack, needle):
		score -= 100
	}
	return score, true
}
