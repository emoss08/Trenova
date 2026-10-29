package settlementpreview

import (
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payadvance"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay"
)

const ParamDriverPayAmount = "amount"

const MaxDriverPayNotes = 1000

const MaxPayReferenceChars = 100

var ErrDriverPayNeedsAPerson = errors.New(
	"driver pay terms, advances and escrow change only once a person approves the proposal",
)

type PersonOnlyPolicy struct {
	Name      string               `json:"-"`
	Resource  permission.Resource  `json:"-"`
	Operation permission.Operation `json:"-"`
	artifact  string
	Rationale string `json:"-"`
}

func (p *PersonOnlyPolicy) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          p.Name,
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      p.Resource,
		Operation:     p.Operation,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agenttooltrust.ToolEffectChange,
		Artifact:      p.artifact,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     p.Rationale,
	}
}

func RequirePersonsApproval(params *agenttoolports.ToolExecuteParams) error {
	if !params.ApprovedFromProposal() {
		return ErrDriverPayNeedsAPerson
	}

	return nil
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

var AdvanceSources = agenttoolschema.Source("driverPay.advanceSource", []payadvance.AdvanceSource{
	payadvance.AdvanceSourceCash,
	payadvance.AdvanceSourceEFSMoneyCode,
	payadvance.AdvanceSourceComdataCode,
	payadvance.AdvanceSourceFuelCard,
	payadvance.AdvanceSourceOther,
})

var DeductionFrequencies = agenttoolschema.Source(
	"driverPay.deductionFrequency",
	[]recurringpay.DeductionFrequency{
		recurringpay.DeductionFrequencyEverySettlement,
		recurringpay.DeductionFrequencyMonthly,
	},
)

var DeductionStatuses = agenttoolschema.Source(
	"driverPay.deductionStatus",
	[]recurringpay.DeductionStatus{
		recurringpay.DeductionStatusActive,
		recurringpay.DeductionStatusPaused,
		recurringpay.DeductionStatusCompleted,
	},
)

var EarningFrequencies = agenttoolschema.Source(
	"driverPay.earningFrequency",
	[]recurringpay.EarningFrequency{
		recurringpay.EarningFrequencyEverySettlement,
		recurringpay.EarningFrequencyMonthly,
	},
)

var EarningStatuses = agenttoolschema.Source(
	"driverPay.earningStatus",
	[]recurringpay.EarningStatus{
		recurringpay.EarningStatusActive,
		recurringpay.EarningStatusPaused,
		recurringpay.EarningStatusCompleted,
	},
)
