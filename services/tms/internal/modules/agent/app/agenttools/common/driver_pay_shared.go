package common

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const ParamDriverPayAmount = "amount"

const MaxDriverPayNotes = 1000

const MaxPayReferenceChars = 100

var ErrDriverPayNeedsAPerson = errors.New(
	"driver pay terms, advances and escrow change only once a person approves the proposal",
)

var ErrNothingToChange = errors.New("give at least one value to change")

type PersonOnlyPolicy struct {
	Name      string               `json:"-"`
	Resource  permission.Resource  `json:"-"`
	Operation permission.Operation `json:"-"`
	artifact  string
	Rationale string `json:"-"`
}

func (p *PersonOnlyPolicy) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          p.Name,
		Kind:          agent.ToolKindAction,
		Resource:      p.Resource,
		Operation:     p.Operation,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agent.ToolEffectChange,
		Artifact:      p.artifact,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     p.Rationale,
	}
}

func RequirePersonsApproval(params *agentports.ToolExecuteParams) error {
	if !params.ApprovedFromProposal() {
		return ErrDriverPayNeedsAPerson
	}

	return nil
}

func RefusalOrFailure(err error) (refusal, failure error) {
	switch {
	case err == nil:
		return nil, nil
	case IsRefusal(err) || errortypes.IsNotFoundError(err):
		return err, nil
	default:
		return nil, err
	}
}

func ObjectParams(properties map[string]any, required ...string) map[string]any {
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

func AmountProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: description,
	}
}

func ErrUnknownValue(key, raw string, names []string) error {
	return fmt.Errorf("%s %q is not one of %s", key, raw, strings.Join(names, ", "))
}

func WorkerProperty() map[string]any {
	return agenttoolschema.RecordIDText(
		permission.ResourceWorker,
		"The driver, from worker_list. Never guess one.",
	)
}

var DriverPayDateTypes = map[string]assistantartifact.DisplayType{
	"issuedDate":    assistantartifact.DisplayDate,
	"openedDate":    assistantartifact.DisplayDate,
	"closedDate":    assistantartifact.DisplayDate,
	"writtenOffAt":  assistantartifact.DisplayDate,
	"effectiveFrom": assistantartifact.DisplayDate,
	"effectiveTo":   assistantartifact.DisplayDate,
	"startDate":     assistantartifact.DisplayDate,
	"endDate":       assistantartifact.DisplayDate,
}

var AdvanceSources = agenttoolschema.Source("driverPay.advanceSource", []driverpay.AdvanceSource{
	driverpay.AdvanceSourceCash,
	driverpay.AdvanceSourceEFSMoneyCode,
	driverpay.AdvanceSourceComdataCode,
	driverpay.AdvanceSourceFuelCard,
	driverpay.AdvanceSourceOther,
})

var DeductionFrequencies = agenttoolschema.Source(
	"driverPay.deductionFrequency",
	[]driverpay.DeductionFrequency{
		driverpay.DeductionFrequencyEverySettlement,
		driverpay.DeductionFrequencyMonthly,
	},
)

var DeductionStatuses = agenttoolschema.Source(
	"driverPay.deductionStatus",
	[]driverpay.DeductionStatus{
		driverpay.DeductionStatusActive,
		driverpay.DeductionStatusPaused,
		driverpay.DeductionStatusCompleted,
	},
)

var EarningFrequencies = agenttoolschema.Source(
	"driverPay.earningFrequency",
	[]driverpay.EarningFrequency{
		driverpay.EarningFrequencyEverySettlement,
		driverpay.EarningFrequencyMonthly,
	},
)

var EarningStatuses = agenttoolschema.Source("driverPay.earningStatus", []driverpay.EarningStatus{
	driverpay.EarningStatusActive,
	driverpay.EarningStatusPaused,
	driverpay.EarningStatusCompleted,
})
