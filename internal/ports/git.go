package ports

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
