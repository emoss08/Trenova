package agenttoolservice

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	paramDriverPayAmount = "amount"
	paramDriverPayNotes  = "notes"
	maxDriverPayNotes    = 1000
	maxPayReferenceChars = 100
)

var (
	ErrDriverPayNeedsAPerson = errors.New(
		"driver pay terms, advances and escrow change only once a person approves the proposal",
	)
	errNothingToChange = errors.New("give at least one value to change")
)

type personOnlyPolicy struct {
	name      string
	resource  permission.Resource
	operation permission.Operation
	artifact  string
	rationale string
}

func (p *personOnlyPolicy) policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          p.name,
		Kind:          agent.ToolKindAction,
		Resource:      p.resource,
		Operation:     p.operation,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agent.ToolEffectChange,
		Artifact:      p.artifact,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     p.rationale,
	}
}

func requirePersonsApproval(params *serviceports.ToolExecuteParams) error {
	if !params.ApprovedFromProposal() {
		return ErrDriverPayNeedsAPerson
	}

	return nil
}

func refusalOrFailure(err error) (refusal, failure error) {
	switch {
	case err == nil:
		return nil, nil
	case isRefusal(err) || errortypes.IsNotFoundError(err):
		return err, nil
	default:
		return nil, err
	}
}

func objectParams(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyAdditionalProperties: false,
	}
	if len(required) > 0 {
		schema[toolschema.KeyRequired] = required
	}

	return schema
}

func idProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: description,
	}
}

func amountProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: description,
	}
}

func dateProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: description + " YYYY-MM-DD.",
	}
}

func enumProperty[T ~string](description string, values []T) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyEnum:        enumNames(values),
		toolschema.KeyDescription: description,
	}
}

func optionalEnum[T ~string](
	params map[string]any,
	key string,
	values []T,
) (value T, ok bool, err error) {
	raw := optionalString(params, key)
	if raw == "" {
		return "", false, nil
	}
	for _, value := range values {
		if string(value) == raw {
			return value, true, nil
		}
	}
	return "", false, errUnknownValue(key, raw, enumNames(values))
}

func errUnknownValue(key, raw string, names []string) error {
	return fmt.Errorf("%s %q is not one of %s", key, raw, strings.Join(names, ", "))
}

func workerProperty() map[string]any {
	return idProperty("The driver, from search_worker or list_workers. Never guess one.")
}

var driverPayDateTypes = map[string]assistantartifact.DisplayType{
	"issuedDate":    assistantartifact.DisplayDate,
	"openedDate":    assistantartifact.DisplayDate,
	"closedDate":    assistantartifact.DisplayDate,
	"writtenOffAt":  assistantartifact.DisplayDate,
	"effectiveFrom": assistantartifact.DisplayDate,
	"effectiveTo":   assistantartifact.DisplayDate,
	"startDate":     assistantartifact.DisplayDate,
	"endDate":       assistantartifact.DisplayDate,
}

var (
	advanceSources = []driverpay.AdvanceSource{
		driverpay.AdvanceSourceCash,
		driverpay.AdvanceSourceEFSMoneyCode,
		driverpay.AdvanceSourceComdataCode,
		driverpay.AdvanceSourceFuelCard,
		driverpay.AdvanceSourceOther,
	}
	deductionFrequencies = []driverpay.DeductionFrequency{
		driverpay.DeductionFrequencyEverySettlement,
		driverpay.DeductionFrequencyMonthly,
	}
	deductionStatuses = []driverpay.DeductionStatus{
		driverpay.DeductionStatusActive,
		driverpay.DeductionStatusPaused,
		driverpay.DeductionStatusCompleted,
	}
	earningFrequencies = []driverpay.EarningFrequency{
		driverpay.EarningFrequencyEverySettlement,
		driverpay.EarningFrequencyMonthly,
	}
	earningStatuses = []driverpay.EarningStatus{
		driverpay.EarningStatusActive,
		driverpay.EarningStatusPaused,
		driverpay.EarningStatusCompleted,
	}
)
