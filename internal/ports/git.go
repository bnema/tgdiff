package ports

// GitMetadataReader provides read-only access to repository metadata independent
// of diff loading. It is used to populate the review context before a review
// session starts.
type GitMetadataReader interface {
	WorktreeRoot(repoPath string) (string, error)
	CurrentBranch(repoPath string) (string, error)
	HeadSHA(repoPath string) (string, error)
	Remotes(repoPath string) ([]GitRemoteInfo, error)
	MergeBase(repoPath, baseRef, headRef string) (string, error)
	ResolveRevision(repoPath, revision string) (string, error)
	DefaultBranch(repoPath string) (string, error)
}

// GitRemoteInfo represents a named Git remote.
type GitRemoteInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type BaseBranchResolver interface {
	ResolveBaseBranch(repoPath string) (string, error)
}

type GitDiffLoader interface {
	LoadBranchDiff(repoPath, baseBranch string) (string, error)
	LoadWorkingTreeDiff(repoPath string) (string, error)
	LoadStagedDiff(repoPath string) (string, error)
	LoadLocalDiff(repoPath string) (string, error)
	LoadUpstreamDiff(repoPath, upstreamRef string) (string, error)
	LoadCommitDiff(repoPath, revision string) (string, error)
	LoadRangeDiff(repoPath, baseRevision, headRevision string) (string, error)
}

type FileContentReader interface {
	ReadFileLines(repoPath, path string) ([]string, error)
}
