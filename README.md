# ero

A terminal UI for reviewing Git diffs file by file.

`ero` opens the current repository in a GitHub-style review flow, with syntax-highlighted diffs, file navigation, search, and expandable context.

## Features

- smart startup mode for local changes, staged changes, upstream commits, or branch diffs
- explicit review modes for branch, working tree, staged, local, upstream, commit, and range diffs
- per-file navigation with a floating file finder
- reference search across the loaded review
- expandable unchanged context around diff hunks
- syntax-aware diff rendering
- selection copy support

## Install

Download a binary from the GitHub releases page, or build from source:

```bash
go install ./cmd/ero
```

## Usage

```bash
ero                  # choose a safe review scope automatically
ero branch           # current branch vs default branch
ero working          # unstaged and untracked changes
ero staged           # staged changes
ero local            # staged + unstaged + untracked changes
ero upstream [ref]   # changes against upstream, default @{upstream}
ero commit <rev>     # one commit
ero range <base> <head>
```

Useful flags:

```bash
ero --repo-path /path/to/repo
ero --context-lines 5
```

## Development

```bash
make test
make fmt
make tidy
```
