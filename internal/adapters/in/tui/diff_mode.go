package tui

import (
	"strings"

	"ero/internal/core"
)

const (
	nerdIconBranch   = "\ue725"
	nerdIconWorking  = "\U000f0704"
	nerdIconStaged   = "\U000f0c52"
	nerdIconLocal    = "\U000f0993"
	nerdIconCommit   = "\ueafc"
	nerdIconRange    = "\U000f062c"
	nerdIconUpstream = "\U000f02a2"
)

var selectableDiffModes = []core.DiffMode{
	core.DiffModeBranch,
	core.DiffModeWorking,
	core.DiffModeStaged,
	core.DiffModeLocal,
	core.DiffModeUpstream,
	core.DiffModeCommit,
	core.DiffModeRange,
}

func diffModeLabel(mode core.DiffMode, nerdFont bool) string {
	plain := diffModePlainLabel(mode)
	if !nerdFont {
		return plain
	}
	icon := diffModeNerdFontIcon(mode)
	if icon == "" {
		return plain
	}
	return icon + " " + strings.TrimSuffix(plain, " diff")
}

func diffModePlainLabel(mode core.DiffMode) string {
	switch mode {
	case core.DiffModeBranch:
		return "branch diff"
	case core.DiffModeWorking:
		return "working diff"
	case core.DiffModeStaged:
		return "staged diff"
	case core.DiffModeLocal:
		return "local diff"
	case core.DiffModeCommit:
		return "commit diff"
	case core.DiffModeRange:
		return "range diff"
	case core.DiffModeUpstream:
		return "upstream diff"
	default:
		return "diff"
	}
}

func diffModeNerdFontIcon(mode core.DiffMode) string {
	switch mode {
	case core.DiffModeBranch:
		return nerdIconBranch
	case core.DiffModeWorking:
		return nerdIconWorking
	case core.DiffModeStaged:
		return nerdIconStaged
	case core.DiffModeLocal:
		return nerdIconLocal
	case core.DiffModeCommit:
		return nerdIconCommit
	case core.DiffModeRange:
		return nerdIconRange
	case core.DiffModeUpstream:
		return nerdIconUpstream
	default:
		return ""
	}
}
