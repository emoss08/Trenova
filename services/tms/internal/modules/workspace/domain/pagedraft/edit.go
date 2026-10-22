package pagedraft

import (
	"slices"
	"strconv"
	"strings"
)

type Action string

const (
	ActionAcceptField        = Action("intake_accept_field")
	ActionAcceptAllConfident = Action("intake_accept_confident")
	ActionSetFieldValue      = Action("intake_set_field")
	ActionSetRequiredField   = Action("intake_set_required_field")
	ActionSetStopLocation    = Action("intake_set_stop_location")
	ActionSetStopSchedule    = Action("intake_set_stop_schedule")
	ActionProposeFormula     = Action("formula_propose")
	ActionProposeLibrary     = Action("formula_library_propose")
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
		ActionProposeLibrary,
	}
}

func (a Action) Surface() Surface {
	switch a {
	case ActionProposeFormula:
		return SurfaceFormula
	case ActionProposeLibrary:
		return SurfaceFormulaLibrary
	case ActionAcceptField, ActionAcceptAllConfident, ActionSetFieldValue,
		ActionSetRequiredField, ActionSetStopLocation, ActionSetStopSchedule:
		return SurfaceShipmentImport
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
	Surface     Surface         `json:"surface"`
	Action      Action          `json:"action"`
	FieldKey    string          `json:"fieldKey,omitempty"`
	Value       string          `json:"value,omitempty"`
	Label       string          `json:"label,omitempty"`
	StopIndex   *int            `json:"stopIndex,omitempty"`
	WindowStart int64           `json:"windowStart,omitempty"`
	WindowEnd   int64           `json:"windowEnd,omitempty"`
	Formula     *SourceProposal `json:"formula,omitempty"`
	Library     *SourceProposal `json:"formulaLibrary,omitempty"`
}

type ProposalTests struct {
	Total  int `json:"total"`
	Passed int `json:"passed"`
}

type SourceProposal struct {
	Name        string        `json:"name,omitempty"`
	Description string        `json:"description,omitempty"`
	Source      string        `json:"source"`
	Explanation string        `json:"explanation"`
	Inputs      []string      `json:"inputs"`
	Imports     []string      `json:"imports"`
	Charges     []string      `json:"charges"`
	Warnings    []string      `json:"warnings"`
	Tests       ProposalTests `json:"tests"`
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
	case ActionProposeLibrary:
		return "Proposed library"
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
	default:
		return "a required field"
	}
}
