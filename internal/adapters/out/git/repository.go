package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	ggit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"tgdiff/internal/core"
)

type RepositoryLoader struct{}

func NewRepositoryLoader() *RepositoryLoader {
	return &RepositoryLoader{}
}

func (l *RepositoryLoader) Open(path string) (*ggit.Repository, error) {
	return ggit.PlainOpen(path)
}

func (l *RepositoryLoader) ResolveBaseBranch(path string) (string, error) {
	repo, err := l.Open(path)
	if err != nil {
		return "", fmt.Errorf("open repository: %w", err)
	}

	originHead, err := repo.Reference(plumbing.ReferenceName("refs/remotes/origin/HEAD"), false)
	switch {
	case err == nil:
		branch, ok := branchName(originHead.Target())
		if ok {
			if _, targetErr := repo.Reference(originHead.Target(), true); targetErr == nil {
				return branch, nil
			} else if !errors.Is(targetErr, plumbing.ErrReferenceNotFound) {
				return "", fmt.Errorf("read origin/HEAD target %s: %w", originHead.Target().String(), targetErr)
			}
		}
	case !errors.Is(err, plumbing.ErrReferenceNotFound):
		return "", fmt.Errorf("read origin/HEAD: %w", err)
	}

	for _, candidate := range []struct {
		branch string
		ref    plumbing.ReferenceName
	}{
		{branch: "main", ref: remoteBranchReferenceName("main")},
		{branch: "master", ref: remoteBranchReferenceName("master")},
		{branch: "main", ref: plumbing.NewBranchReferenceName("main")},
		{branch: "master", ref: plumbing.NewBranchReferenceName("master")},
	} {
		_, err := repo.Reference(candidate.ref, true)
		if err == nil {
			return candidate.branch, nil
		}
		if !errors.Is(err, plumbing.ErrReferenceNotFound) {
			return "", fmt.Errorf("read branch reference %s: %w", candidate.ref.String(), err)
		}
	}

	return "", fmt.Errorf("resolve base branch: no origin/HEAD, origin/main, origin/master, main, or master reference found")
}

func (l *RepositoryLoader) ReadStartupState(path string) (core.StartupState, error) {
	output, err := runGit(path, "status", "--porcelain=v1", "--branch", "--untracked-files=normal")
	if err != nil {
		return core.StartupState{}, err
	}

	state := parseStartupStatus(output)
	if _, err := l.ResolveBaseBranch(path); err == nil {
		state.HasDefaultBranch = true
	}
	return state, nil
}

func (l *RepositoryLoader) LoadWorkingTreeDiff(path string) (string, error) {
	diff, err := runGitDiff(path)
	if err != nil {
		return "", err
	}
	untrackedDiff, err := l.loadUntrackedFilesDiff(path)
	if err != nil {
		return "", err
	}
	return diff + untrackedDiff, nil
}

func (l *RepositoryLoader) LoadStagedDiff(path string) (string, error) {
	return runGitDiff(path, "--staged")
}

func (l *RepositoryLoader) LoadLocalDiff(path string) (string, error) {
	diff, err := runGitDiff(path, "HEAD")
	if err != nil {
		return "", err
	}
	untrackedDiff, err := l.loadUntrackedFilesDiff(path)
	if err != nil {
		return "", err
	}
	return diff + untrackedDiff, nil
}

func (l *RepositoryLoader) LoadUpstreamDiff(path, upstreamRef string) (string, error) {
	return runGitDiff(path, upstreamRef+"...HEAD")
}

func (l *RepositoryLoader) LoadCommitDiff(path, revision string) (string, error) {
	return runGit(path, "diff-tree", "--root", "--no-commit-id", "-r", "-p", "--no-ext-diff", revision)
}

func (l *RepositoryLoader) LoadRangeDiff(path, baseRevision, headRevision string) (string, error) {
	return runGitDiff(path, baseRevision+"..."+headRevision)
}

func runGitDiff(path string, args ...string) (string, error) {
	gitArgs := append([]string{"diff", "--no-ext-diff"}, args...)
	return runGit(path, gitArgs...)
}

func runGit(path string, args ...string) (string, error) {
	cleanPath, err := cleanRepositoryPath(path)
	if err != nil {
		return "", err
	}
	cmdArgs := append([]string{"-c", "color.ui=false", "-c", "color.diff=false", "-C", cleanPath}, args...)
	cmd := exec.Command("git", cmdArgs...)
	stderr := &strings.Builder{}
	cmd.Stderr = stderr
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(output), nil
}

func cleanRepositoryPath(path string) (string, error) {
	absolute, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolve repository path: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("read repository path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repository path is not a directory: %s", absolute)
	}
	return absolute, nil
}

func (l *RepositoryLoader) LoadBranchDiff(path, baseBranch string) (string, error) {
	repo, err := l.Open(path)
	if err != nil {
		return "", fmt.Errorf("open repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("resolve head: %w", err)
	}
	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", fmt.Errorf("load head commit: %w", err)
	}

	baseRef, err := resolveBranchReference(repo, baseBranch)
	if err != nil {
		return "", err
	}
	baseCommit, err := repo.CommitObject(baseRef.Hash())
	if err != nil {
		return "", fmt.Errorf("load base commit %s: %w", baseRef.Name().String(), err)
	}

	mergeBases, err := headCommit.MergeBase(baseCommit)
	if err != nil {
		return "", fmt.Errorf("find merge base: %w", err)
	}
	if len(mergeBases) == 0 {
		return "", fmt.Errorf("find merge base: no common ancestor between %s and %s", head.Name().String(), baseRef.Name().String())
	}

	patch, err := mergeBases[0].Patch(headCommit)
	if err != nil {
		return "", fmt.Errorf("build patch: %w", err)
	}

	untrackedDiff, err := l.untrackedFilesDiff(repo, path)
	if err != nil {
		return "", err
	}
	return patch.String() + untrackedDiff, nil
}

func parseStartupStatus(output string) core.StartupState {
	var state core.StartupState
	for line := range strings.SplitSeq(strings.TrimSuffix(output, "\n"), "\n") {
		if strings.HasPrefix(line, "## ") {
			parseStartupBranchLine(line, &state)
			continue
		}
		if len(line) < 2 {
			continue
		}
		indexStatus := line[0]
		worktreeStatus := line[1]
		switch {
		case indexStatus == '?' && worktreeStatus == '?':
			state.HasUntrackedFiles = true
		default:
			if indexStatus != ' ' && indexStatus != '?' {
				state.HasStagedChanges = true
			}
			if worktreeStatus != ' ' && worktreeStatus != '?' {
				state.HasUnstagedChanges = true
			}
		}
	}
	return state
}

func parseStartupBranchLine(line string, state *core.StartupState) {
	branch := strings.TrimPrefix(line, "## ")
	if strings.HasPrefix(branch, "HEAD ") || branch == "HEAD" {
		state.DetachedHead = true
		return
	}
	if !strings.Contains(branch, "...") {
		return
	}
	state.HasUpstream = true
	metadataStart := strings.Index(branch, "[")
	metadataEnd := strings.LastIndex(branch, "]")
	if metadataStart < 0 || metadataEnd <= metadataStart {
		return
	}
	for part := range strings.SplitSeq(branch[metadataStart+1:metadataEnd], ",") {
		part = strings.TrimSpace(part)
		if value, ok := strings.CutPrefix(part, "ahead "); ok {
			state.Ahead = parsePositiveInt(value)
		}
		if value, ok := strings.CutPrefix(part, "behind "); ok {
			state.Behind = parsePositiveInt(value)
		}
	}
}

func parsePositiveInt(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func branchName(name plumbing.ReferenceName) (string, bool) {
	for _, prefix := range []string{"refs/remotes/origin/", "refs/heads/"} {
		if branch, ok := strings.CutPrefix(name.String(), prefix); ok {
			return branch, true
		}
	}
	return "", false
}

func resolveBranchReference(repo *ggit.Repository, branch string) (*plumbing.Reference, error) {
	candidates := []plumbing.ReferenceName{
		remoteBranchReferenceName(branch),
		plumbing.NewBranchReferenceName(branch),
	}

	for _, candidate := range candidates {
		ref, err := repo.Reference(candidate, true)
		if err == nil {
			return ref, nil
		}
		if !errors.Is(err, plumbing.ErrReferenceNotFound) {
			return nil, fmt.Errorf("read branch reference %s: %w", candidate.String(), err)
		}
	}

	return nil, fmt.Errorf("resolve base branch %q: reference not found", branch)
}

func remoteBranchReferenceName(branch string) plumbing.ReferenceName {
	return plumbing.ReferenceName("refs/remotes/origin/" + branch)
}

func (l *RepositoryLoader) loadUntrackedFilesDiff(repoPath string) (string, error) {
	repo, err := l.Open(repoPath)
	if err != nil {
		return "", fmt.Errorf("open repository: %w", err)
	}
	return l.untrackedFilesDiff(repo, repoPath)
}

func (l *RepositoryLoader) untrackedFilesDiff(repo *ggit.Repository, repoPath string) (string, error) {
	worktree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("open worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return "", fmt.Errorf("read worktree status: %w", err)
	}

	paths := make([]string, 0)
	for path, fileStatus := range status {
		if fileStatus.Staging == ggit.Untracked && fileStatus.Worktree == ggit.Untracked {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)

	var diff strings.Builder
	for _, path := range paths {
		content, err := os.ReadFile(filepath.Join(repoPath, path))
		if err != nil {
			return "", fmt.Errorf("read untracked file %s: %w", path, err)
		}
		writeAddedFileDiff(&diff, path, content)
	}

	return diff.String(), nil
}

func writeAddedFileDiff(diff *strings.Builder, path string, content []byte) {
	text := strings.TrimSuffix(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	lines := []string{}
	if text != "" {
		for line := range strings.SplitSeq(text, "\n") {
			lines = append(lines, line)
		}
	}

	fmt.Fprintf(diff, "diff --git a/%s b/%s\n", path, path)
	fmt.Fprintln(diff, "new file mode 100644")
	fmt.Fprintln(diff, "index 0000000..0000000")
	fmt.Fprintln(diff, "--- /dev/null")
	fmt.Fprintf(diff, "+++ b/%s\n", path)
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(diff, "@@ -0,0 +1,%d @@\n", len(lines))
	for _, line := range lines {
		fmt.Fprintf(diff, "+%s\n", line)
	}
}

func (l *RepositoryLoader) ReadFileLines(repoPath, path string) ([]string, error) {
	content, err := os.ReadFile(filepath.Join(repoPath, path))
	if err != nil {
		return nil, err
	}

	text := strings.TrimSuffix(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	if text == "" {
		return nil, nil
	}

	var lines []string
	for line := range strings.SplitSeq(text, "\n") {
		lines = append(lines, line)
	}
	return lines, nil
}
