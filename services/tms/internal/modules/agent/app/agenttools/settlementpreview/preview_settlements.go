package settlementpreview

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/shared/pulid"
)

type settlementJournalView struct {
	AccountingDate   int64    `json:"accountingDate"`
	FiscalPeriodID   pulid.ID `json:"fiscalPeriodId"`
	EntryStatus      string   `json:"entryStatus"`
	RequiresApproval bool     `json:"requiresApproval"`
	Description      string   `json:"description"`
	LineCount        int      `json:"lineCount"`
}

func PayeeClause(payee string) string {
	if strings.TrimSpace(payee) == "" {
		return ""
	}

	return " for " + payee
}

func SettlementMoney(before, after *SettlementFacts) *agentpreview.MoneyPreview {
	lines := make([]agentpreview.MoneyLine, 0, max(len(before.Totals), len(after.Totals)))
	for idx, total := range after.Totals {
		line := agentpreview.MoneyLine{
			Label: total.Label,
			After: changepreview.MinorAmount(total.Minor),
		}
		if idx < len(before.Totals) {
			line.Before = changepreview.MinorAmount(before.Totals[idx].Minor)
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
) (*agentpreview.RecordChange, error) {
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

	lines := make([]agentpreview.MoneyLine, 0, len(plan.Lines))
	for _, line := range plan.Lines {
		if line.DebitMinor > 0 {
			lines = append(lines, agentpreview.MoneyLine{
				Label: "Debit",
				After: changepreview.MinorAmount(line.DebitMinor),
			})
		}
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(settlement.Currency, lines...),
		toolpreview.SensitiveAs("totalAmount"))

	return change, nil
}
