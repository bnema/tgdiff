package ports

// StartupStateReader reads repository state needed by startup mode detection.
// The concrete startup state remains in core; this port intentionally avoids
// owning domain/application data to keep hexagonal boundaries clear.
type StartupStateReader[T any] interface {
	ReadStartupState(repoPath string) (T, error)
}
