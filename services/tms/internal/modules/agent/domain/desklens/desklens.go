package desklens

import "github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"

// Lens is one of the Desk's views over what waits on people. The Watchtower
// raises every kind of item into one feed; the Desk reads it through a
// lens: the changes agents propose are Decisions, mail is Triage, and
// everything else is the Watchtower's own. Everything is all of them.
type Lens string

const (
	Everything = Lens("Everything")
	Watchtower = Lens("Watchtower")
	Triage     = Lens("Triage")
	Decisions  = Lens("Decisions")
)

func (l Lens) IsValid() bool {
	switch l {
	case Everything, Watchtower, Triage, Decisions:
		return true
	}

	return false
}

func (l Lens) String() string { return string(l) }

// Of is the lens a kind of item is read through.
func Of(kind watchtower.SourceKind) Lens {
	switch kind { //nolint:exhaustive // every other kind is the Watchtower's own
	case watchtower.SourceAgentProposal, watchtower.SourceAgentPlan:
		return Decisions
	case watchtower.SourceInboundMessage:
		return Triage
	default:
		return Watchtower
	}
}

// Kinds are the kinds of item the lens shows. Nil for Everything, which
// narrows nothing.
func (l Lens) Kinds() []watchtower.SourceKind {
	if l == Everything || !l.IsValid() {
		return nil
	}
	all := watchtower.AllSourceKinds()
	kinds := make([]watchtower.SourceKind, 0, len(all))
	for _, kind := range all {
		if Of(kind) == l {
			kinds = append(kinds, kind)
		}
	}

	return kinds
}

// Narrow keeps the kinds asked for that the lens shows; with none asked
// for, it is every kind the lens shows.
func (l Lens) Narrow(kinds []watchtower.SourceKind) []watchtower.SourceKind {
	if len(kinds) == 0 {
		return l.Kinds()
	}
	out := make([]watchtower.SourceKind, 0, len(kinds))
	for _, kind := range kinds {
		if l == Everything || Of(kind) == l {
			out = append(out, kind)
		}
	}

	return out
}
