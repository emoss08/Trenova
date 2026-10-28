package agentreach

// AudienceCoverage is how much of an agent a role could use: all of its
// tools, some of them, or none because the role cannot use the assistant.
type AudienceCoverage string

const (
	CoverageFull    = AudienceCoverage("Full")
	CoveragePartial = AudienceCoverage("Partial")
	CoverageNone    = AudienceCoverage("None")
)

func (c AudienceCoverage) IsValid() bool {
	switch c {
	case CoverageFull, CoveragePartial, CoverageNone:
		return true
	default:
		return false
	}
}
