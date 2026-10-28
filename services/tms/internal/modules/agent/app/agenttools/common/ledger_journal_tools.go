package common

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/manualjournalports"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/shared/pulid"
)

type LedgerAccounts interface {
	GetByIDs(
		ctx context.Context,
		req glaccountports.GetGLAccountsByIDsRequest,
	) ([]*glaccount.GLAccount, error)
}

type JournalRevision struct {
	ID          pulid.ID                                     `json:"-"`
	Description string                                       `json:"-"`
	Reason      string                                       `json:"-"`
	HasReason   bool                                         `json:"-"`
	Date        int64                                        `json:"-"`
	Currency    string                                       `json:"-"`
	Lines       []*manualjournalports.ManualJournalLineInput `json:"-"`
}

func (r *JournalRevision) Over(
	current *manualjournal.Request,
	params *agenttoolports.ToolExecuteParams,
) *manualjournalports.UpdateManualJournalDraftRequest {
	req := &manualjournalports.UpdateManualJournalDraftRequest{
		RequestID:      current.ID,
		Description:    current.Description,
		Reason:         current.Reason,
		AccountingDate: current.AccountingDate,
		CurrencyCode:   current.CurrencyCode,
		Lines:          linesOf(current),
		TenantInfo:     TenantFrom(*params),
	}
	if r.Description != "" {
		req.Description = r.Description
	}
	if r.HasReason {
		req.Reason = r.Reason
	}
	if r.Date != 0 {
		req.AccountingDate = r.Date
	}
	if r.Currency != "" {
		req.CurrencyCode = r.Currency
	}
	if r.Lines != nil {
		req.Lines = r.Lines
	}

	return req
}

func linesOf(current *manualjournal.Request) []*manualjournalports.ManualJournalLineInput {
	lines := make([]*manualjournalports.ManualJournalLineInput, 0, len(current.Lines))
	for _, line := range current.Lines {
		if line == nil {
			continue
		}
		lines = append(lines, &manualjournalports.ManualJournalLineInput{
			GLAccountID:  line.GLAccountID,
			Description:  line.Description,
			DebitAmount:  line.DebitAmount,
			CreditAmount: line.CreditAmount,
			CustomerID:   line.CustomerID,
			LocationID:   line.LocationID,
		})
	}

	return lines
}

func LedgerMoneySpec(spec *ReceivableSpec) *ReceivableSpec {
	spec.Egress = agent.EgressMoney
	spec.DefaultTier = agent.TierPropose
	spec.MaxTier = agent.TierPropose
	spec.PersonOnly = true

	return spec
}
