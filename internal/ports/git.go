package ports

type BaseBranchResolver interface {
	ResolveBaseBranch(repoPath string) (string, error)
}

type GitDiffLoader interface {
	LoadBranchDiff(repoPath, baseBranch string) (string, error)
}

type FileContentReader interface {
	ReadFileLines(repoPath, path string) ([]string, error)
}
