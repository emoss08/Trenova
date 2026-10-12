package pagedraft

import (
	"slices"
	"strconv"
	"strings"
)

type Action string

const (
	ActionAcceptField        = Action("accept_field")
	ActionAcceptAllConfident = Action("accept_all_confident")
	ActionSetFieldValue      = Action("set_field_value")
	ActionSetRequiredField   = Action("set_required_field")
	ActionSetStopLocation    = Action("set_stop_location")
	ActionSetStopSchedule    = Action("set_stop_schedule")
	ActionProposeFormula     = Action("propose_formula")
)

func (a Action) IsValid() bool {
	return slices.Contains(AllActions(), a)
}

func AllActions() []Action {
	return []Action{
		ActionAcceptField,
		ActionAcceptAllConfident,
		ActionSetFieldValue,
		ActionSetRequiredField,
		ActionSetStopLocation,
		ActionSetStopSchedule,
		ActionProposeFormula,
	}
}

func (a Action) Surface() Surface {
	if a == ActionProposeFormula {
		return SurfaceFormula
	}

	return SurfaceShipmentImport
}

func (a Action) ToolName() string {
	return string(a)
}

func EditToolNames() []string {
	actions := AllActions()
	names := make([]string, 0, len(actions))
	for _, action := range actions {
		names = append(names, action.ToolName())
	}

	return names
}

func IsEditTool(name string) bool {
	return Action(name).IsValid()
}

type Edit struct {
	Surface     Surface          `json:"surface"`
	Action      Action           `json:"action"`
	FieldKey    string           `json:"fieldKey,omitempty"`
	Value       string           `json:"value,omitempty"`
	Label       string           `json:"label,omitempty"`
	StopIndex   *int             `json:"stopIndex,omitempty"`
	WindowStart int64            `json:"windowStart,omitempty"`
	WindowEnd   int64            `json:"windowEnd,omitempty"`
	Formula     *FormulaProposal `json:"formula,omitempty"`
}

type FormulaCheck struct {
	Valid  bool   `json:"valid"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type PricedScenario struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Variables   map[string]any `json:"variables"`
	Amount      string         `json:"amount,omitempty"`
	Valid       bool           `json:"valid"`
	Error       string         `json:"error,omitempty"`
}

type FormulaProposal struct {
	SchemaID    string            `json:"schemaId"`
	Expression  string            `json:"expression"`
	Variables   []FormulaVariable `json:"variables"`
	Explanation string            `json:"explanation"`
	Check       FormulaCheck      `json:"check"`
	Scenarios   []PricedScenario  `json:"scenarios"`
}

type EditResult struct {
	Draft Edit   `json:"draft"`
	Note  string `json:"note"`
}

func (e *Edit) Title() string {
	switch e.Action {
	case ActionAcceptField:
		return "Accepted " + e.fieldName()
	case ActionAcceptAllConfident:
		return "Accepted every confident field"
	case ActionSetFieldValue:
		return "Set " + e.fieldName()
	case ActionSetRequiredField:
		if label := strings.TrimSpace(e.Label); label != "" {
			return "Set " + requiredFieldName(RequiredField(e.FieldKey)) + " to " + label
		}

		return "Set " + requiredFieldName(RequiredField(e.FieldKey))
	case ActionSetStopLocation:
		if label := strings.TrimSpace(e.Label); label != "" {
			return "Matched " + e.stopName() + " to " + label
		}

		return "Matched " + e.stopName() + " to a location"
	case ActionSetStopSchedule:
		return "Scheduled " + e.stopName()
	case ActionProposeFormula:
		return "Proposed formula"
	default:
		return "Draft change"
	}
}

func (e *Edit) fieldName() string {
	if label := strings.TrimSpace(e.Label); label != "" {
		return label
	}
	if key := strings.TrimSpace(e.FieldKey); key != "" {
		return key
	}

	return "a field"
}

func (e *Edit) stopName() string {
	if e.StopIndex == nil {
		return "a stop"
	}

	return "stop " + strconv.Itoa(*e.StopIndex+1)
}

func requiredFieldName(field RequiredField) string {
	switch field {
	case RequiredCustomer:
		return "the customer"
	case RequiredServiceType:
		return "the service type"
	case RequiredShipmentType:
		return "the shipment type"
	case RequiredFormulaTemplate:
		return "the rating method"
	default:
		return "a required field"
	}
}
