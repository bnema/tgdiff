package ports

type Terminal interface {
	SupportsNerdFont() bool
}
