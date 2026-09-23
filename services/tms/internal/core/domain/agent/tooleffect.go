package agent

type ToolEffect string

const (
	ToolEffectLookup   = ToolEffect("lookup")
	ToolEffectChange   = ToolEffect("change")
	ToolEffectNavigate = ToolEffect("navigate")
	ToolEffectDiscover = ToolEffect("discover")
	ToolEffectPresent  = ToolEffect("present")
	ToolEffectAsk      = ToolEffect("ask")
)

func (e ToolEffect) IsValid() bool {
	switch e {
	case ToolEffectLookup,
		ToolEffectChange,
		ToolEffectNavigate,
		ToolEffectDiscover,
		ToolEffectPresent,
		ToolEffectAsk:
		return true
	default:
		return false
	}
}
