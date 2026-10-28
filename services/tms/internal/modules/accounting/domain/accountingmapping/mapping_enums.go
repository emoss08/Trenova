package accountingmapping

type MappingState string

const (
	MappingStateUnmatched = MappingState("Unmatched")
	MappingStateProposed  = MappingState("Proposed")
	MappingStateConfirmed = MappingState("Confirmed")
)

func (s MappingState) String() string { return string(s) }

func (s MappingState) IsValid() bool {
	switch s {
	case MappingStateUnmatched, MappingStateProposed, MappingStateConfirmed:
		return true
	default:
		return false
	}
}

func AllMappingStates() []MappingState {
	return []MappingState{MappingStateUnmatched, MappingStateProposed, MappingStateConfirmed}
}

type MappingSource string

const (
	MappingSourceSuggested         = MappingSource("Suggested")
	MappingSourceModel             = MappingSource("Model")
	MappingSourceManual            = MappingSource("Manual")
	MappingSourceCreatedInProvider = MappingSource("CreatedInProvider")
	MappingSourceAgent             = MappingSource("Agent")
)

func (s MappingSource) String() string { return string(s) }

func (s MappingSource) IsValid() bool {
	switch s {
	case MappingSourceSuggested,
		MappingSourceModel,
		MappingSourceManual,
		MappingSourceCreatedInProvider,
		MappingSourceAgent:
		return true
	default:
		return false
	}
}

func AllMappingSources() []MappingSource {
	return []MappingSource{
		MappingSourceSuggested,
		MappingSourceModel,
		MappingSourceManual,
		MappingSourceCreatedInProvider,
		MappingSourceAgent,
	}
}

func (s MappingSource) Deliberate() bool {
	return s == MappingSourceManual || s == MappingSourceCreatedInProvider ||
		s == MappingSourceAgent
}
