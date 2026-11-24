package bankreceiptcandidateservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Repo   bankreceiptports.CandidateRepository
	Logger *zap.Logger
}

type Service struct {
	repo bankreceiptports.CandidateRepository
	l    *zap.Logger
}

var _ bankreceiptports.CandidateRefresher = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		repo: p.Repo,
		l:    p.Logger.Named("service.bank-receipt-candidates"),
	}
}

func (s *Service) Refresh(
	ctx context.Context,
	tenant pagination.TenantInfo,
	subjectIDs []pulid.ID,
) (int, error) {
	receipts, err := s.repo.ReceiptsTouching(ctx, &bankreceiptports.ReceiptsTouchingRequest{
		TenantInfo: tenant,
		SubjectIDs: subjectIDs,
	})
	if err != nil {
		return 0, err
	}
	cleared := make([]pulid.ID, 0)
	for _, id := range subjectIDs {
		if bankreceipt.IsReceiptID(id) {
			cleared = append(cleared, id)
		}
	}
	if err = s.write(ctx, tenant, receipts, cleared); err != nil {
		return 0, err
	}

	return len(receipts), nil
}

func (s *Service) RefreshReceipt(
	ctx context.Context,
	receipt *bankreceipt.BankReceipt,
) ([]*bankreceipt.Candidate, error) {
	if receipt == nil {
		return nil, nil
	}
	tenant := pagination.TenantInfo{OrgID: receipt.OrganizationID, BuID: receipt.BusinessUnitID}
	scored, err := s.score(ctx, []*bankreceipt.BankReceipt{receipt})
	if err != nil {
		return nil, err
	}
	if err = s.repo.Replace(ctx, &bankreceiptports.ReplaceCandidatesRequest{
		TenantInfo: tenant,
		ReceiptIDs: []pulid.ID{receipt.ID},
		Candidates: scored,
	}); err != nil {
		return nil, err
	}

	return scored, nil
}

func (s *Service) write(
	ctx context.Context,
	tenant pagination.TenantInfo,
	receipts []*bankreceipt.BankReceipt,
	cleared []pulid.ID,
) error {
	ids := make([]pulid.ID, 0, len(receipts)+len(cleared))
	ids = append(ids, cleared...)
	open := make([]*bankreceipt.BankReceipt, 0, len(receipts))
	for _, receipt := range receipts {
		ids = append(ids, receipt.ID)
		if receipt.Status != bankreceipt.StatusMatched {
			open = append(open, receipt)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	scored, err := s.score(ctx, open)
	if err != nil {
		return err
	}

	return s.repo.Replace(ctx, &bankreceiptports.ReplaceCandidatesRequest{
		TenantInfo: tenant,
		ReceiptIDs: ids,
		Candidates: scored,
	})
}

func (s *Service) score(
	ctx context.Context,
	receipts []*bankreceipt.BankReceipt,
) ([]*bankreceipt.Candidate, error) {
	if len(receipts) == 0 {
		return nil, nil
	}
	payments, err := s.repo.PaymentEvidence(ctx, receipts)
	if err != nil {
		return nil, err
	}
	numbers := make([]string, 0, len(receipts))
	for _, receipt := range receipts {
		numbers = append(numbers, receipt.LookupNumbers()...)
	}
	invoices, err := s.repo.OpenInvoicesByNumber(ctx, numbers)
	if err != nil {
		return nil, err
	}
	out := make([]*bankreceipt.Candidate, 0, len(receipts))
	for _, receipt := range receipts {
		out = append(out, bankreceipt.Score(receipt, payments[receipt.ID], invoices)...)
	}

	return out, nil
}
