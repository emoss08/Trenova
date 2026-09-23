package agent

type ToolEffect string

const (
	ToolEffectLookup   = ToolEffect("lookup")
	ToolEffectChange   = ToolEffect("change")
	ToolEffectNavigate = ToolEffect("navigate")
	ToolEffectDiscover = ToolEffect("discover")
	ToolEffectPresent  = ToolEffect("present")
	ToolEffectAsk      = ToolEffect("ask")
	// ToolEffectDelegate hands a task to another agent.
	ToolEffectDelegate = ToolEffect("delegate")
)

func (e ToolEffect) IsValid() bool {
	switch e {
	case ToolEffectLookup,
		ToolEffectChange,
		ToolEffectNavigate,
		ToolEffectDiscover,
		ToolEffectPresent,
		ToolEffectAsk,
		ToolEffectDelegate:
		return true
	default:
		return false
	}
}
