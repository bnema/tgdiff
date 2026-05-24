# tgdiff

Reusable Git diff review TUI in Go.

## v1 scope

- diff the current branch against the repo default branch
- resolve the default branch from `origin/HEAD`, then fall back to `main` or `master`
- present diffs per file in a GitHub-style review flow
- show collapsed unchanged context with expand-above, expand-below, and expand-all actions
- keep the project reusable for other CLI agents via adapters and ports

## Architecture

Detailed layout: `docs/architecture.md`

Key rules:

- `cmd/tgdiff/main.go` stays minimal
- `internal/app` is the composition root and dependency injection point
- `internal/core` owns domain types plus core business/application services
- `ports` live alongside `core`
- Cobra/Viper live in the CLI adapter
- Bubble Tea root model stays thin
- reusable TUI components own their own rendering responsibilities
- there is no separate `usecase` package; orchestration stays in `internal/core`
- the core owns review-section and context-expansion rules

## Current implementation

- `cmd/tgdiff/main.go` is a minimal process entrypoint
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
go run ./cmd/tgdiff --help
```
