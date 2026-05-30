package app

import (
	"path/filepath"

	"ero/internal/core"
	"ero/internal/ports"
	"ero/pkg/plugin/protocol"
)

func buildReviewContext(request core.ReviewRequest, files []core.ReviewFile, metadata ports.GitMetadataReader, version string) core.ReviewContext {
	if request.RepoPath == "" {
		request.RepoPath = "."
	}
	ctx := core.ReviewContext{
		Repository: core.RepositoryMetadata{RepoPath: request.RepoPath},
		Target: core.ReviewTargetMetadata{
			Mode:    request.DiffMode,
			BaseRef: request.BaseRevision,
			HeadRef: request.HeadRevision,
		},
		Diff: core.DiffMetadata{FilesChanged: len(files)},
		Session: core.ReviewSessionMetadata{
			EroVersion:      version,
			ProtocolVersion: protocol.ProtocolVersion,
		},
	}
	if ctx.Target.Mode == "" {
		ctx.Target.Mode = core.DiffModeBranch
	}
	if ctx.Target.Mode == core.DiffModeCommit {
		ctx.Target.HeadRef = request.Revision
	}
	if ctx.Target.Mode == core.DiffModeUpstream {
		ctx.Target.BaseRef = request.UpstreamRef
		ctx.Target.HeadRef = "HEAD"
	}

	ctx.Files = make([]core.ReviewFileMetadata, 0, len(files))
	for _, file := range files {
		status := file.Status
		if status == "" {
			status = core.ReviewFileStatusModified
		}
		meta := core.ReviewFileMetadata{Path: file.Path, OldPath: file.OldPath, Status: status, Language: languageFromPath(file.Path)}
		for _, section := range file.Sections {
			if section.Kind == core.SectionKindChanged {
				anchor := core.ReviewHunkAnchor{SectionID: section.ID}
				for _, line := range section.VisibleLines() {
					if anchor.OldStartLine == 0 && line.OldLineNumber > 0 {
						anchor.OldStartLine = line.OldLineNumber
					}
					if anchor.NewStartLine == 0 && line.NewLineNumber > 0 {
						anchor.NewStartLine = line.NewLineNumber
					}
					if anchor.OldStartLine > 0 && anchor.NewStartLine > 0 {
						break
					}
				}
				meta.Hunks = append(meta.Hunks, anchor)
			}
			for _, line := range section.VisibleLines() {
				if line.Kind == core.LineKindAdded {
					ctx.Diff.Additions++
				}
				if line.Kind == core.LineKindDeleted {
					ctx.Diff.Deletions++
				}
				meta.LineAnchors = append(meta.LineAnchors, core.NewReviewLineAnchor(file.Path, line))
			}
		}
		ctx.Files = append(ctx.Files, meta)
	}

	if metadata != nil {
		if value, err := metadata.WorktreeRoot(request.RepoPath); err == nil {
			ctx.Repository.WorktreeRoot = value
		}
		if value, err := metadata.CurrentBranch(request.RepoPath); err == nil {
			ctx.Repository.CurrentBranch = value
		}
		if value, err := metadata.DefaultBranch(request.RepoPath); err == nil {
			ctx.Repository.DefaultBranch = value
			if ctx.Target.Mode == core.DiffModeBranch && ctx.Target.BaseRef == "" {
				ctx.Target.BaseRef = value
			}
		}
		if value, err := metadata.HeadSHA(request.RepoPath); err == nil {
			ctx.Repository.HeadSHA = value
			ctx.Target.HeadSHA = value
		}
		if remotes, err := metadata.Remotes(request.RepoPath); err == nil {
			for _, remote := range remotes {
				ctx.Repository.Remotes = append(ctx.Repository.Remotes, core.GitRemote{Name: remote.Name, URL: remote.URL})
			}
		}
		if ctx.Target.BaseRef != "" {
			if value, err := metadata.ResolveRevision(request.RepoPath, ctx.Target.BaseRef); err == nil {
				ctx.Target.BaseSHA = value
			}
		}
		if ctx.Target.HeadRef != "" {
			if value, err := metadata.ResolveRevision(request.RepoPath, ctx.Target.HeadRef); err == nil {
				ctx.Target.HeadSHA = value
			}
		}
		if ctx.Target.BaseRef != "" && ctx.Target.HeadRef != "" {
			if value, err := metadata.MergeBase(request.RepoPath, ctx.Target.BaseRef, ctx.Target.HeadRef); err == nil {
				ctx.Target.MergeBaseSHA = value
			}
		}
	}
	return ctx.Normalize()
}

func languageFromPath(path string) string {
	ext := filepath.Ext(path)
	if len(ext) > 1 {
		return ext[1:]
	}
	return ""
}
