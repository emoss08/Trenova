package agent

type AutonomyAnswer string

const (
	AutonomyRunsOnItsOwn  = AutonomyAnswer("runs_on_its_own")
	AutonomyConditional   = AutonomyAnswer("conditional")
	AutonomyNeedsApproval = AutonomyAnswer("needs_approval")
	AutonomyProposeOnly   = AutonomyAnswer("propose_only")
	AutonomySimulated     = AutonomyAnswer("simulated")
)

func (a AutonomyAnswer) IsValid() bool {
	switch a {
	case AutonomyRunsOnItsOwn,
		AutonomyConditional,
		AutonomyNeedsApproval,
		AutonomyProposeOnly,
		AutonomySimulated:
		return true
	default:
		return false
	}
}

func (a AutonomyAnswer) String() string { return string(a) }

func (a AutonomyAnswer) WithoutAPerson() bool {
	return a == AutonomyRunsOnItsOwn || a == AutonomyConditional
}

func (t AutonomyTier) Label() string {
	switch t {
	case TierActWithApproval:
		return "Ask first"
	case TierAutoExecute:
		return "Automatic"
	default:
		return "Propose"
	}
}

func AnswerForTier(tier AutonomyTier) AutonomyAnswer {
	switch tier {
	case TierAutoExecute:
		return AutonomyRunsOnItsOwn
	case TierActWithApproval:
		return AutonomyNeedsApproval
	default:
		return AutonomyProposeOnly
	}
}
