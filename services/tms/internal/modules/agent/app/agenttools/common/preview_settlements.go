package common

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type settlementJournalView struct {
	AccountingDate   int64    `json:"accountingDate"`
	FiscalPeriodID   pulid.ID `json:"fiscalPeriodId"`
	EntryStatus      string   `json:"entryStatus"`
	RequiresApproval bool     `json:"requiresApproval"`
	Description      string   `json:"description"`
	LineCount        int      `json:"lineCount"`
}

func WouldFail(preview *agent.ToolPreview, refusal error) (*agent.ToolPreview, error) {
	return WarnWouldFail(preview, refusal), nil
}

func MinorAmount(minor int64) decimal.NullDecimal {
	return KnownAmount(money.DecimalFromMinor(minor))
}

func PayeeClause(payee string) string {
	if strings.TrimSpace(payee) == "" {
		return ""
	}

	return " for " + payee
}

func SettlementMoney(before, after *SettlementFacts) *agent.MoneyPreview {
	lines := make([]agent.MoneyLine, 0, max(len(before.Totals), len(after.Totals)))
	for idx, total := range after.Totals {
		line := agent.MoneyLine{Label: total.Label, After: MinorAmount(total.Minor)}
		if idx < len(before.Totals) {
			line.Before = MinorAmount(before.Totals[idx].Minor)
		}
		lines = append(lines, line)
	}

	return toolpreview.MoneyBlock(after.Currency, lines...)
}

func JournalSentence(plan *settlementshared.JournalPlan) string {
	if plan.RequiresApproval {
		return fmt.Sprintf(
			"A %s journal entry would be booked and wait for its approval.",
			strings.ToLower(plan.EntryStatus),
		)
	}

	return fmt.Sprintf("A %s journal entry would be booked.", strings.ToLower(plan.EntryStatus))
}

func SettlementJournalChange(
	plan *settlementshared.JournalPlan,
	settlement *SettlementFacts,
) (*agent.RecordChange, error) {
	change, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceJournalEntry,
			Label:    plan.Description,
		},
		&settlementJournalView{
			AccountingDate:   plan.AccountingDate,
			FiscalPeriodID:   plan.FiscalPeriodID,
			EntryStatus:      plan.EntryStatus,
			RequiresApproval: plan.RequiresApproval,
			Description:      plan.Description,
			LineCount:        len(plan.Lines),
		},
		toolpreview.WithRefs(map[string]permission.Resource{
			"fiscalPeriodId": permission.ResourceFiscalPeriod,
		}),
		toolpreview.Types(map[string]assistantartifact.DisplayType{
			"accountingDate": assistantartifact.DisplayDate,
		}),
	)
	if err != nil {
		return nil, err
	}

	lines := make([]agent.MoneyLine, 0, len(plan.Lines))
	for _, line := range plan.Lines {
		if line.DebitMinor > 0 {
			lines = append(lines, agent.MoneyLine{
				Label: "Debit",
				After: MinorAmount(line.DebitMinor),
			})
		}
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(settlement.Currency, lines...),
		toolpreview.SensitiveAs("totalAmount"))

	return change, nil
}
