package settlementshared

import (
	"github.com/emoss08/trenova/internal/core/services/journalposting"
)

func JournalPlanFrom(posting *journalposting.Plan, description string) *JournalPlan {
	lines := make([]JournalLine, 0, len(posting.Lines))
	for _, line := range posting.Lines {
		lines = append(lines, JournalLine{
			AccountID:   line.GLAccountID,
			DebitMinor:  line.DebitAmount,
			CreditMinor: line.CreditAmount,
		})
	}
	return &JournalPlan{
		AccountingDate:   posting.AccountingDate,
		FiscalYearID:     posting.Period.FiscalYearID,
		FiscalPeriodID:   posting.Period.ID,
		EntryStatus:      posting.Workflow.EntryStatus,
		RequiresApproval: posting.Workflow.RequiresApproval,
		Description:      description,
		Lines:            lines,
	}
}
