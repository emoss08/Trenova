package agenttoolservice

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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

func (p *personOnlyPolicy) policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
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

func requirePersonsApproval(params *agentports.ToolExecuteParams) error {
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

func amountProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: description,
	}
}

func errUnknownValue(key, raw string, names []string) error {
	return fmt.Errorf("%s %q is not one of %s", key, raw, strings.Join(names, ", "))
}

func workerProperty() map[string]any {
	return agenttoolschema.RecordIDText(
		permission.ResourceWorker,
		"The driver, from search_worker or list_workers. Never guess one.",
	)
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
	advanceSources = agenttoolschema.Source("driverPay.advanceSource", []driverpay.AdvanceSource{
		driverpay.AdvanceSourceCash,
		driverpay.AdvanceSourceEFSMoneyCode,
		driverpay.AdvanceSourceComdataCode,
		driverpay.AdvanceSourceFuelCard,
		driverpay.AdvanceSourceOther,
	})
	deductionFrequencies = agenttoolschema.Source(
		"driverPay.deductionFrequency",
		[]driverpay.DeductionFrequency{
			driverpay.DeductionFrequencyEverySettlement,
			driverpay.DeductionFrequencyMonthly,
		},
	)
	deductionStatuses = agenttoolschema.Source(
		"driverPay.deductionStatus",
		[]driverpay.DeductionStatus{
			driverpay.DeductionStatusActive,
			driverpay.DeductionStatusPaused,
			driverpay.DeductionStatusCompleted,
		},
	)
	earningFrequencies = agenttoolschema.Source(
		"driverPay.earningFrequency",
		[]driverpay.EarningFrequency{
			driverpay.EarningFrequencyEverySettlement,
			driverpay.EarningFrequencyMonthly,
		},
	)
	earningStatuses = agenttoolschema.Source("driverPay.earningStatus", []driverpay.EarningStatus{
		driverpay.EarningStatusActive,
		driverpay.EarningStatusPaused,
		driverpay.EarningStatusCompleted,
	})
)
