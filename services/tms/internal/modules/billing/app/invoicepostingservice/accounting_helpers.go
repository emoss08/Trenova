package invoicepostingservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/accounting/app/journalposting"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

func (s *Service) invoiceLedger() *invoiceledger.Poster {
	return &invoiceledger.Poster{
		Controls: s.accountingRepo,
		Policy:   s.accountingPolicyService(),
		Ledger:   s.customerLedgerRepo,
		Writer: journalposting.Writer{
			Periods:  s.fiscalPeriodRepo,
			Numbers:  s.sequenceGenerator,
			Journals: s.journalRepo,
		},
	}
}

func invoiceLedgerRequest(
	entity *invoice.Invoice,
	actor *permissionports.RequestActor,
) *invoiceledger.Request {
	req := &invoiceledger.Request{Invoice: entity}
	if actor != nil {
		req.ActorID = actor.UserID
	}
	return req
}

func (s *Service) planInvoiceJournal(
	ctx context.Context,
	entity *invoice.Invoice,
	actor *permissionports.RequestActor,
) (*invoiceledger.Plan, error) {
	if entity == nil || entity.PostedAt == nil {
		return nil, invoiceledger.ErrNoLedgerEntry
	}
	return s.invoiceLedger().Plan(ctx, invoiceLedgerRequest(entity, actor))
}

func (s *Service) createInvoiceJournalPosting(
	ctx context.Context,
	entity *invoice.Invoice,
	actor *permissionports.RequestActor,
) error {
	if entity == nil || entity.PostedAt == nil {
		return nil
	}
	_, err := s.invoiceLedger().Post(ctx, invoiceLedgerRequest(entity, actor))
	if errors.Is(err, invoiceledger.ErrNoLedgerEntry) {
		return nil
	}
	return err
}
