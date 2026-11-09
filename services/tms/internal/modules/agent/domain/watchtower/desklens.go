package watchtower

type DeskLens string

const (
	DeskLensEverything = DeskLens("Everything")
	DeskLensWatchtower = DeskLens("Watchtower")
	DeskLensTriage     = DeskLens("Triage")
	DeskLensDecisions  = DeskLens("Decisions")
)

func (l DeskLens) IsValid() bool {
	switch l {
	case DeskLensEverything, DeskLensWatchtower, DeskLensTriage, DeskLensDecisions:
		return true
	}

	return false
}

func (l DeskLens) String() string { return string(l) }

func (k SourceKind) DeskLens() DeskLens {
	switch k {
	case SourceAgentProposal, SourceAgentPlan:
		return DeskLensDecisions
	case SourceInboundMessage:
		return DeskLensTriage
	default:
		return DeskLensWatchtower
	}
}

func (l DeskLens) Kinds() []SourceKind {
	if l == DeskLensEverything || !l.IsValid() {
		return nil
	}
	kinds := make([]SourceKind, 0, len(AllSourceKinds()))
	for _, kind := range AllSourceKinds() {
		if kind.DeskLens() == l {
			kinds = append(kinds, kind)
		}
	}

	return kinds
}

func (l DeskLens) Narrow(kinds []SourceKind) []SourceKind {
	if len(kinds) == 0 {
		return l.Kinds()
	}
	out := make([]SourceKind, 0, len(kinds))
	for _, kind := range kinds {
		if l == DeskLensEverything || kind.DeskLens() == l {
			out = append(out, kind)
		}
	}

	return out
}
