package invoicepostingservice

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// PreviewPostSummary is PreviewPost as the person posting reads it: the
// journal lines named by their accounts, and every refusal worded on its own.
func (s *Service) PreviewPostSummary(
	ctx context.Context,
	req *invoiceports.PostInvoiceRequest,
	actor *permissionports.RequestActor,
) (*invoiceports.InvoicePostPreview, error) {
	preview, err := s.PreviewPost(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	before := preview.Before
	summary := &invoiceports.InvoicePostPreview{
		InvoiceID:       before.ID,
		Number:          before.Number,
		BillToName:      before.BillToName,
		CurrencyCode:    before.CurrencyCode,
		TotalMinor:      before.TotalAmountMinor,
		Refusals:        refusalMessages(preview.Refusal),
		AlreadyPosted:   preview.AlreadyPosted,
		ShipmentsBilled: make([]string, 0, len(preview.Legs)),
		QueueItemPosted: preview.QueueAfter != nil,
		AccountingSync: make(
			[]invoiceports.InvoicePostSyncDestination,
			0,
			len(preview.AccountingSync),
		),
	}
	if preview.Refused() {
		return summary, nil
	}

	for _, leg := range preview.Legs {
		summary.ShipmentsBilled = append(summary.ShipmentsBilled, leg.Before.ProNumber)
	}
	for _, destination := range preview.AccountingSync {
		summary.AccountingSync = append(
			summary.AccountingSync,
			invoiceports.InvoicePostSyncDestination{
				Integration:   destination.Integration,
				Company:       destination.Company,
				AwaitsRelease: destination.AwaitsRelease,
			},
		)
	}
	if plan := preview.EDI; plan != nil && plan.Enabled {
		summary.EDI = &invoiceports.InvoicePostEDI{
			PartnerName: plan.PartnerName,
			WillSend:    plan.AutoSend && len(plan.Blockers) == 0,
			Blockers:    plan.Blockers,
		}
	}
	if preview.Journal != nil {
		if summary.Journal, err = s.namedJournal(ctx, preview.Journal, req.TenantInfo); err != nil {
			return nil, err
		}
	}

	return summary, nil
}

func (s *Service) namedJournal(
	ctx context.Context,
	journal *kernelports.JournalPreview,
	tenantInfo pagination.TenantInfo,
) (*invoiceports.InvoicePostJournal, error) {
	named := &invoiceports.InvoicePostJournal{
		AccountingDate:   journal.AccountingDate,
		EntryStatus:      journal.EntryStatus,
		RequiresApproval: journal.RequiresApproval,
		Lines:            make([]invoiceports.InvoicePostJournalLine, 0, len(journal.Lines)),
	}

	accounts, err := s.journalAccounts(ctx, journal, tenantInfo)
	if err != nil {
		return nil, err
	}
	for _, line := range journal.Lines {
		named.Lines = append(named.Lines, invoiceports.InvoicePostJournalLine{
			GLAccountID: line.GLAccountID,
			AccountCode: accounts[line.GLAccountID].code,
			AccountName: accounts[line.GLAccountID].name,
			Description: line.Description,
			DebitMinor:  line.DebitMinor,
			CreditMinor: line.CreditMinor,
		})
	}

	return named, nil
}

type accountName struct {
	code string
	name string
}

func (s *Service) journalAccounts(
	ctx context.Context,
	journal *kernelports.JournalPreview,
	tenantInfo pagination.TenantInfo,
) (map[pulid.ID]accountName, error) {
	if s.glAccounts == nil || len(journal.Lines) == 0 {
		return nil, nil //nolint:nilnil // no journal lines means no account names to look up
	}

	ids := make([]pulid.ID, 0, len(journal.Lines))
	seen := make(map[pulid.ID]struct{}, len(journal.Lines))
	for _, line := range journal.Lines {
		if _, ok := seen[line.GLAccountID]; ok || line.GLAccountID.IsNil() {
			continue
		}
		seen[line.GLAccountID] = struct{}{}
		ids = append(ids, line.GLAccountID)
	}

	found, err := s.glAccounts.GetByIDs(ctx, glaccountports.GetGLAccountsByIDsRequest{
		TenantInfo:   tenantInfo,
		GLAccountIDs: ids,
	})
	if err != nil {
		return nil, err
	}

	accounts := make(map[pulid.ID]accountName, len(found))
	for _, account := range found {
		accounts[account.ID] = accountName{code: account.AccountCode, name: account.Name}
	}

	return accounts, nil
}

// refusalMessages words a refusal as the reasons a person can act on: one per
// field a validation error names, or the refusal itself.
func refusalMessages(err error) []string {
	if err == nil {
		return []string{}
	}

	if multiErr, ok := errors.AsType[*errortypes.MultiError](err); ok && multiErr.HasErrors() {
		messages := make([]string, 0, len(multiErr.Errors))
		for _, fieldErr := range multiErr.Errors {
			if fieldErr != nil {
				messages = append(messages, strings.TrimSpace(fieldErr.Error()))
			}
		}

		return messages
	}

	return []string{strings.TrimSpace(err.Error())}
}
