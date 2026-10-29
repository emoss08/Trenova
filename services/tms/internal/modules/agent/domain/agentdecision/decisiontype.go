package agentdecision

type DecisionType string

const (
	DecisionAccepted = DecisionType("Accepted")
	DecisionModified = DecisionType("Modified")
	DecisionRejected = DecisionType("Rejected")
)

func (d DecisionType) IsValid() bool {
	switch d {
	case DecisionAccepted, DecisionModified, DecisionRejected:
		return true
	default:
		return false
	}
}
