package core

// DiffMode specifies the source of changes to load for review.
type DiffMode string

const (
	// DiffModeBranch compares the current branch against its base branch.
	DiffModeBranch DiffMode = "branch"
	// DiffModeWorking represents unstaged working tree changes.
	DiffModeWorking DiffMode = "working"
	// DiffModeStaged represents staged changes in the index.
	DiffModeStaged DiffMode = "staged"
	// DiffModeLocal represents all local uncommitted changes.
	DiffModeLocal DiffMode = "local"
	// DiffModeCommit represents a single commit's changes.
	DiffModeCommit DiffMode = "commit"
	// DiffModeRange represents changes across a commit or branch range.
	DiffModeRange DiffMode = "range"
	// DiffModeUpstream compares changes against the upstream branch.
	DiffModeUpstream DiffMode = "upstream"
)

// IsValid reports whether d is a recognized DiffMode constant.
func (d DiffMode) IsValid() bool {
	switch d {
	case DiffModeBranch, DiffModeWorking, DiffModeStaged, DiffModeLocal, DiffModeCommit, DiffModeRange, DiffModeUpstream:
		return true
	default:
		return false
	}
}

// ReviewRequest describes the review content to load.
// RepoPath is the repository root, ContextLines controls collapsed context size,
// and DiffMode selects the source of changes for the review. Revision is used
// by DiffModeCommit, BaseRevision and HeadRevision by DiffModeRange, and
// UpstreamRef by DiffModeUpstream. Other modes only require RepoPath,
// ContextLines, and DiffMode.
type ReviewRequest struct {
	RepoPath     string
	ContextLines int
	DiffMode     DiffMode
	Revision     string
	BaseRevision string
	HeadRevision string
	UpstreamRef  string
}

func (r ReviewRequest) modeOrDefault() DiffMode {
	if r.DiffMode == "" {
		return DiffModeBranch
	}
	return r.DiffMode
}
