package tui

import "strings"

type DiffMode string

const (
	DiffModeBranch   DiffMode = "branch"
	DiffModeWorking  DiffMode = "working"
	DiffModeStaged   DiffMode = "staged"
	DiffModeLocal    DiffMode = "local"
	DiffModeCommit   DiffMode = "commit"
	DiffModeRange    DiffMode = "range"
	DiffModeUpstream DiffMode = "upstream"
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

var allDiffModes = []DiffMode{
	DiffModeBranch,
	DiffModeWorking,
	DiffModeStaged,
	DiffModeLocal,
	DiffModeCommit,
	DiffModeRange,
	DiffModeUpstream,
}

var selectableDiffModes = []DiffMode{
	DiffModeBranch,
}

func (m DiffMode) Label(nerdFont bool) string {
	plain := m.PlainLabel()
	if !nerdFont {
		return plain
	}
	icon := m.NerdFontIcon()
	if icon == "" {
		return plain
	}
	return icon + " " + strings.TrimSuffix(plain, " diff")
}

func (m DiffMode) PlainLabel() string {
	switch m {
	case DiffModeBranch:
		return "branch diff"
	case DiffModeWorking:
		return "working diff"
	case DiffModeStaged:
		return "staged diff"
	case DiffModeLocal:
		return "local diff"
	case DiffModeCommit:
		return "commit diff"
	case DiffModeRange:
		return "range diff"
	case DiffModeUpstream:
		return "upstream diff"
	default:
		return "diff"
	}
}

func (m DiffMode) NerdFontIcon() string {
	switch m {
	case DiffModeBranch:
		return nerdIconBranch
	case DiffModeWorking:
		return nerdIconWorking
	case DiffModeStaged:
		return nerdIconStaged
	case DiffModeLocal:
		return nerdIconLocal
	case DiffModeCommit:
		return nerdIconCommit
	case DiffModeRange:
		return nerdIconRange
	case DiffModeUpstream:
		return nerdIconUpstream
	default:
		return ""
	}
}
