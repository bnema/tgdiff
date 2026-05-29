# Ero

Ero is a reusable Git diff review TUI in Go.

## v1 scope

- diff the current branch against the repo default branch
- resolve the default branch from `origin/HEAD`, then fall back to `main` or `master`
- present diffs per file in a GitHub-style review flow
- show collapsed unchanged context with expand-above, expand-below, and expand-all actions
- keep the project reusable for other CLI agents via adapters and ports

## Architecture

Detailed layout: `docs/architecture.md`

Key rules:

- `cmd/ero/main.go` stays minimal
- `internal/app` is the composition root and dependency injection point
- `internal/core` owns domain types plus core business/application services
- `ports` live alongside `core`
- Cobra/Viper live in the CLI adapter
- Bubble Tea root model stays thin
- reusable TUI components own their own rendering responsibilities
- there is no separate `usecase` package; orchestration stays in `internal/core`
- the core owns review-section and context-expansion rules

## Current implementation

- `cmd/ero/main.go` is a minimal process entrypoint
- `internal/app` wires the CLI, review loader, git adapter, syntax tokenizer, and TUI runner
- git access uses `github.com/go-git/go-git/v5` (`v5.19.1`, latest stable; v6 is currently alpha)
- the review loader:
  - resolves the default branch from `origin/HEAD`, then remote/local `main` or `master`
  - computes branch-vs-default diffs from the merge base
  - parses unified diff output into per-file review sections
  - applies Chroma-backed syntax tokens to rendered review lines
- the TUI provides:
  - per-file navigation
  - floating file find with `f`
  - floating reference grep with `/`
  - collapsed-context expander rows
  - expand above / below / all actions
  - syntax-aware diff line rendering
- Mock generation is standardized with Mockery v3 via `.mockery.yml`

## Commands

```bash
make test
make mocks
make install
go run ./cmd/ero --help
```

Start modes:

```bash
ero                  # smart startup detection
ero branch           # working branch vs default branch
ero working          # unstaged working tree changes
ero staged           # staged changes
ero local            # staged + unstaged local changes
ero upstream [ref]   # against upstream, default @{upstream}
ero commit <rev>     # one commit
ero range <base> <head>
```

Smart startup detection chooses the first safe review scope:

1. staged + unstaged or untracked changes: prompt for Staged, Unstaged, or All local changes
2. unstaged or untracked changes only: `working`
3. staged changes only: `staged`
4. local commits ahead of a configured upstream: `upstream`
5. no upstream but a default branch exists: `branch`
6. behind-only: exit with `branch is behind upstream; pull first or choose an explicit diff mode`
7. detached HEAD: exit with `detached HEAD has no safe default diff; choose an explicit diff mode`
8. no reviewable changes: exit with `no local changes or upstream/default branch diff detected`

Here, unstaged means modified tracked files; untracked means new files. Both are counted as
worktree changes for `working` and `local` reviews.

When mixed local changes are detected in a non-interactive terminal, choose explicitly with
`ero staged`, `ero working`, or `ero local`.
