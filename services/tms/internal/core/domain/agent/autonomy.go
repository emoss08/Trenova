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

// TierSource is what set the tier a call ran or was held at: the tool's own
// policy and the agent's ceilings, a person choosing the tool's tier on the
// agent, a tier the tool earned through a clean record of approvals, or the
// exemption that lets a person's change to their own records run unasked.
type TierSource string

const (
	TierSourcePolicyDefault     = TierSource("PolicyDefault")
	TierSourcePersonSetting     = TierSource("PersonSetting")
	TierSourceTrustEarned       = TierSource("TrustEarned")
	TierSourcePersonalExemption = TierSource("PersonalExemption")
)

func (s TierSource) IsValid() bool {
	switch s {
	case TierSourcePolicyDefault,
		TierSourcePersonSetting,
		TierSourceTrustEarned,
		TierSourcePersonalExemption:
		return true
	default:
		return false
	}
}

func AllTierSources() []TierSource {
	return []TierSource{
		TierSourcePolicyDefault,
		TierSourcePersonSetting,
		TierSourceTrustEarned,
		TierSourcePersonalExemption,
	}
}
