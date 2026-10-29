package factoringreadinessservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/factoringservicetest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringreadiness"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubStates struct {
	factoringports.InvoiceStateRepository
	saved map[pulid.ID]*factoringreadiness.InvoiceState
}

func (s *stubStates) Get(
	_ context.Context,
	req factoringports.GetInvoiceStateRequest,
) (*factoringreadiness.InvoiceState, error) {
	if state, ok := s.saved[req.InvoiceID]; ok {
		return state, nil
	}
	return nil, errortypes.NewNotFoundError(
		"FactoringInvoiceState not found within your organization",
	)
}

func (s *stubStates) Upsert(_ context.Context, entity *factoringreadiness.InvoiceState) error {
	s.saved[entity.InvoiceID] = entity
	return nil
}

type stubAgreements struct {
	factoringports.AgreementRepository
}

func (stubAgreements) GetInForce(
	context.Context,
	factoringports.GetAgreementInForceRequest,
) (*factoring.Agreement, error) {
	return nil, errortypes.NewNotFoundError("FactoringAgreement not found within your organization")
}

type stubNoas struct {
	factoringports.NoaRepository
}

func (stubNoas) GetOpen(
	context.Context,
	*factoringports.GetOpenNoaRequest,
) (*factoring.Noa, error) {
	return nil, errortypes.NewNotFoundError("FactoringNoa not found within your organization")
}

type stubInvoices struct {
	invoiceports.InvoiceRepository
	inv *invoice.Invoice
}

func (s *stubInvoices) GetByID(
	_ context.Context,
	req invoiceports.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	if s.inv != nil && s.inv.ID == req.ID {
		return s.inv, nil
	}
	return nil, errortypes.NewNotFoundError("Invoice not found within your organization")
}

type stubDocuments struct {
	documentports.DocumentRepository
}

func (stubDocuments) GetByResourceIDs(
	context.Context,
	*documentports.GetDocumentsByResourceIDsRequest,
) ([]*document.Document, error) {
	return nil, nil
}

func TestEvaluateFirstTimeWithoutAgreementOrNotice(t *testing.T) {
	t.Parallel()

	factorID := pulid.MustNew("fac_")
	posted := int64(1_800_000_000)
	inv := &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		OrganizationID:     pulid.MustNew("org_"),
		BusinessUnitID:     pulid.MustNew("bu_"),
		CustomerID:         pulid.MustNew("cus_"),
		Number:             "INV-1",
		Status:             invoice.StatusPosted,
		BillType:           billingqueue.BillTypeInvoice,
		PostedAt:           &posted,
		TotalAmountMinor:   100000,
		FactoringCompanyID: &factorID,
	}
	states := &stubStates{saved: map[pulid.ID]*factoringreadiness.InvoiceState{}}
	svc := NewReadinessService(ReadinessParams{
		States:      states,
		Agreements:  stubAgreements{},
		Noas:        stubNoas{},
		Submissions: factoringservicetest.NewFakeSubmissions(),
		Invoices:    &stubInvoices{inv: inv},
		Documents:   stubDocuments{},
		Logger:      zap.NewNop(),
	})
	tenant := pagination.TenantInfo{OrgID: inv.OrganizationID, BuID: inv.BusinessUnitID}

	result, err := svc.Evaluate(t.Context(), &factoringports.EvaluateInvoicesRequest{
		TenantInfo: tenant,
		InvoiceIDs: []pulid.ID{inv.ID, pulid.MustNew("inv_")},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Evaluated)
	assert.Equal(t, 0, result.Failed)

	state := states.saved[inv.ID]
	require.NotNil(t, state)
	assert.Equal(t, factoringreadiness.VerdictNotReady, state.Verdict)
	failed := make([]factoringreadiness.CheckCode, 0, 2)
	for _, check := range state.Checks {
		if !check.Passed {
			failed = append(failed, check.Code)
		}
	}
	assert.Equal(
		t,
		[]factoringreadiness.CheckCode{
			factoringreadiness.CheckAgreement,
			factoringreadiness.CheckNoa,
		},
		failed,
	)
}
