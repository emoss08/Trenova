package invoiceservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/invoiceledger"
	"github.com/emoss08/trenova/internal/core/services/journalposting"
)

func (s *Service) invoiceLedger() (*invoiceledger.Poster, bool) {
	if s.accountingRepo == nil || s.journalRepo == nil || s.sequenceGenerator == nil ||
		s.validator == nil {
		return nil, false
	}

	poster := &invoiceledger.Poster{
		Controls: s.accountingRepo,
		Policy:   s.accountingPolicyService(),
		Writer: journalposting.Writer{
			Periods:  s.validator.fiscalPeriodRepo,
			Numbers:  s.sequenceGenerator,
			Journals: s.journalRepo,
		},
	}
	if s.customerLedgerRepo != nil {
		poster.Ledger = s.customerLedgerRepo
	}
	return poster, true
}

func invoiceLedgerRequest(
	entity *invoice.Invoice,
	actor *servicesports.RequestActor,
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
	actor *servicesports.RequestActor,
) (*invoiceledger.Plan, error) {
	poster, ok := s.invoiceLedger()
	if !ok || entity == nil || entity.PostedAt == nil {
		return nil, invoiceledger.ErrNoLedgerEntry
	}
	return poster.Plan(ctx, invoiceLedgerRequest(entity, actor))
}

func (s *Service) createInvoiceJournalPosting(
	ctx context.Context,
	entity *invoice.Invoice,
	actor *servicesports.RequestActor,
) error {
	poster, ok := s.invoiceLedger()
	if !ok || entity == nil || entity.PostedAt == nil {
		return nil
	}
	_, err := poster.Post(ctx, invoiceLedgerRequest(entity, actor))
	if errors.Is(err, invoiceledger.ErrNoLedgerEntry) {
		return nil
	}
	return err
}
