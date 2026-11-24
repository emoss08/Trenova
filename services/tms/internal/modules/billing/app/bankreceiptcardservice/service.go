package bankreceiptcardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Cards      bankreceiptports.DeskCardReader
	Candidates bankreceiptports.CandidateRepository
}

type Service struct {
	cards      bankreceiptports.DeskCardReader
	candidates bankreceiptports.CandidateRepository
}

func New(p Params) *Service {
	return &Service{cards: p.Cards, candidates: p.Candidates}
}

func (s *Service) Get(
	ctx context.Context,
	req *bankreceiptports.GetDeskCardRequest,
) (*bankreceiptports.DeskCard, error) {
	if req.ReceiptID.IsNil() {
		return nil, errortypes.NewValidationError(
			"receiptId",
			errortypes.ErrRequired,
			"A bank receipt is required",
		)
	}
	if req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
		return nil, errortypes.NewValidationError(
			"tenantInfo",
			errortypes.ErrRequired,
			"An organization is required",
		)
	}

	card, err := s.cards.GetDeskCard(ctx, req)
	if err != nil {
		return nil, err
	}
	card.Candidates = []bankreceiptports.DeskCardCandidate{}
	card.CandidatesWithheld = []string{}
	if card.Receipt == nil || card.Receipt.Status == bankreceipt.StatusMatched {
		return card, nil
	}
	kinds := make([]bankreceipt.CandidateKind, 0, 2)
	if req.IncludePayment {
		kinds = append(kinds, bankreceipt.CandidateCustomerPayment)
	} else {
		card.CandidatesWithheld = append(card.CandidatesWithheld,
			string(bankreceipt.CandidateCustomerPayment))
	}
	if req.IncludeInvoices {
		kinds = append(kinds, bankreceipt.CandidateInvoice)
	} else {
		card.CandidatesWithheld = append(
			card.CandidatesWithheld,
			string(bankreceipt.CandidateInvoice),
		)
	}
	if len(kinds) == 0 {
		return card, nil
	}
	views, err := s.candidates.ListForReceipt(ctx, &bankreceiptports.ListCandidatesRequest{
		TenantInfo: req.TenantInfo,
		ReceiptID:  req.ReceiptID,
		Kinds:      kinds,
	})
	if err != nil {
		return nil, err
	}
	for _, view := range views {
		card.Candidates = append(card.Candidates, candidateOf(view))
	}

	return card, nil
}

func candidateOf(view *bankreceiptports.CandidateView) bankreceiptports.DeskCardCandidate {
	c := view.Candidate

	return bankreceiptports.DeskCardCandidate{
		Rank:             c.Rank,
		Kind:             string(c.Kind),
		InvoiceID:        c.InvoiceID,
		InvoiceNumber:    view.InvoiceNumber,
		PaymentID:        c.CustomerPaymentID,
		PaymentReference: view.Reference,
		CustomerName:     view.CustomerName,
		Standing:         string(c.Standing),
		Evidence:         c.Evidence,
		OpenMinor:        c.OpenMinor,
		DifferenceMinor:  c.DifferenceMinor,
	}
}
