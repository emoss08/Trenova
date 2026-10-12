package agentsubjectservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptworkitemports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeContent struct {
	documentports.DocumentContentService

	draft  *documentshipmentdraft.DocumentShipmentDraft
	lastID pulid.ID
}

func (f *fakeContent) GetShipmentDraft(
	_ context.Context,
	documentID pulid.ID,
	_ pagination.TenantInfo,
) (*documentshipmentdraft.DocumentShipmentDraft, error) {
	f.lastID = documentID

	return f.draft, nil
}

// A run woken by document.extracted starts with the draft in front of it,
// the way a run woken by a service failure starts with the shipment.
func TestService_DescribesADocumentByItsDraft(t *testing.T) {
	t.Parallel()

	attached := pulid.MustNew("shp_")
	content := &fakeContent{draft: &documentshipmentdraft.DocumentShipmentDraft{
		Status:             documentshipmentdraft.StatusReady,
		DocumentKind:       "rate_confirmation",
		Confidence:         0.82,
		AttachedShipmentID: &attached,
		DraftData: map[string]any{
			"fields": map[string]any{"bol": map[string]any{"value": "BOL-778", "confidence": 0.97}},
		},
	}}
	subjects := &Service{content: content, logger: zap.NewNop()}

	docID := pulid.MustNew("doc_")
	subject, err := subjects.Describe(
		t.Context(),
		pagination.TenantInfo{},
		agent.SubjectDocument,
		docID,
	)
	require.NoError(t, err)

	assert.Equal(t, docID, content.lastID)
	assert.Equal(t, "Document (rate confirmation)", subject.Label)
	assert.Contains(t, subject.Notes, "BOL-778")
	assert.Contains(t, subject.Notes, attached.String())
	assert.Contains(t, subject.Notes, "do not create another")
}

func TestService_DocumentWithoutContentServiceStillHasAnID(t *testing.T) {
	t.Parallel()

	subjects := &Service{logger: zap.NewNop()}
	docID := pulid.MustNew("doc_")

	subject, err := subjects.Describe(
		t.Context(),
		pagination.TenantInfo{},
		agent.SubjectDocument,
		docID,
	)
	require.NoError(t, err)
	assert.Equal(t, docID.String(), subject.ID)
	assert.Equal(t, "Document", subject.Label)
	assert.Empty(t, subject.Notes)
}

type fakeInsightRepo struct {
	insightports.InsightRepository

	found  *insight.Insight
	lastID pulid.ID
}

func (f *fakeInsightRepo) GetByID(
	_ context.Context,
	req insightports.GetInsightByIDRequest,
) (*insight.Insight, error) {
	f.lastID = req.ID

	return f.found, nil
}

// A run woken by insight.detected starts with the finding in front of it:
// what was measured, about whom, and what the detector suggests.
func TestService_DescribesAnInsightByItsFinding(t *testing.T) {
	t.Parallel()

	found := &insight.Insight{
		ID:             pulid.MustNew("inst_"),
		Category:       insight.CategoryCashFlow,
		Severity:       insight.SeverityCritical,
		Status:         insight.StatusActive,
		Subject:        "Acme Foods",
		Headline:       "$48,200 of delivered work for Acme Foods is not yet billed",
		Recommendation: "Move the ready items through the billing queue.",
		Metrics: []insight.Metric{
			{Key: "unbilled", Label: "Unbilled", Value: decimal.NewFromInt(48200)},
		},
	}
	repo := &fakeInsightRepo{found: found}
	subjects := &Service{insights: repo, logger: zap.NewNop()}

	subject, err := subjects.Describe(
		t.Context(),
		pagination.TenantInfo{},
		agent.SubjectInsight,
		found.ID,
	)
	require.NoError(t, err)

	assert.Equal(t, found.ID, repo.lastID)
	assert.Equal(t, "Insight: "+found.Headline, subject.Label)
	assert.Contains(t, subject.Notes, "Acme Foods")
	assert.Contains(t, subject.Notes, "48200")
	assert.Contains(t, subject.Notes, "billing queue")
	assert.NotContains(t, subject.Notes, "no longer active")
}

func TestService_WarnsWhenTheInsightIsNoLongerActive(t *testing.T) {
	t.Parallel()

	found := &insight.Insight{
		ID:       pulid.MustNew("inst_"),
		Status:   insight.StatusResolved,
		Headline: "Detention at Acme Foods dock 4 has stopped",
	}
	subjects := &Service{insights: &fakeInsightRepo{found: found}, logger: zap.NewNop()}

	subject, err := subjects.Describe(
		t.Context(),
		pagination.TenantInfo{},
		agent.SubjectInsight,
		found.ID,
	)
	require.NoError(t, err)
	assert.Contains(t, subject.Notes, "no longer active")
}

func TestService_InsightWithoutRepositoryStillHasAnID(t *testing.T) {
	t.Parallel()

	subjects := &Service{logger: zap.NewNop()}
	id := pulid.MustNew("inst_")

	subject, err := subjects.Describe(
		t.Context(),
		pagination.TenantInfo{},
		agent.SubjectInsight,
		id,
	)
	require.NoError(t, err)
	assert.Equal(t, id.String(), subject.ID)
	assert.Equal(t, "Insight", subject.Label)
}

type fakeReceiptService struct {
	bankreceiptports.BankReceiptService

	receipt     *bankreceipt.BankReceipt
	suggestions []*bankreceiptports.BankReceiptMatchSuggestion
}

func (f *fakeReceiptService) Get(
	_ context.Context,
	_ *bankreceiptports.GetBankReceiptRequest,
) (*bankreceipt.BankReceipt, error) {
	return f.receipt, nil
}

func (f *fakeReceiptService) SuggestMatches(
	_ context.Context,
	_ *bankreceiptports.GetBankReceiptRequest,
) ([]*bankreceiptports.BankReceiptMatchSuggestion, error) {
	return f.suggestions, nil
}

type fakeWorkItemRepo struct {
	bankreceiptworkitemports.BankReceiptWorkItemRepository

	item *bankreceiptworkitem.WorkItem
}

func (f *fakeWorkItemRepo) GetActiveByReceiptID(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (*bankreceiptworkitem.WorkItem, error) {
	if f.item == nil {
		return nil, errortypes.NewNotFoundError("bank receipt work item not found")
	}

	return f.item, nil
}

// A run woken by bank_receipt.exception starts with the receipt, the scored
// candidates and the queue entry in front of it, with the amount in money.
func TestService_DescribesABankReceiptWithItsCandidates(t *testing.T) {
	t.Parallel()

	receipt := &bankreceipt.BankReceipt{
		ID:              pulid.MustNew("brcpt_"),
		AmountMinor:     125_000,
		ReferenceNumber: "ACH 4471",
		Memo:            "ACME FOODS INC",
		Status:          bankreceipt.StatusException,
		ExceptionReason: "No unique customer payment match found for bank receipt",
	}
	paymentID := pulid.MustNew("cpay_")
	item := &bankreceiptworkitem.WorkItem{
		ID:            pulid.MustNew("brwi_"),
		BankReceiptID: receipt.ID,
		Status:        bankreceiptworkitem.StatusOpen,
	}
	subjects := &Service{
		receipts: &fakeReceiptService{
			receipt: receipt,
			suggestions: []*bankreceiptports.BankReceiptMatchSuggestion{{
				CustomerPaymentID: paymentID, AmountMinor: 125_000, Score: 60, Reason: "Reference matches",
			}},
		},
		workItems: &fakeWorkItemRepo{item: item},
		logger:    zap.NewNop(),
	}

	subject, err := subjects.Describe(
		t.Context(),
		pagination.TenantInfo{},
		agent.SubjectBankReceipt,
		receipt.ID,
	)
	require.NoError(t, err)

	assert.Equal(t, "Bank receipt 1250.00 ref ACH 4471", subject.Label)
	assert.Contains(t, subject.Notes, "ACME FOODS INC")
	assert.Contains(t, subject.Notes, paymentID.String())
	assert.Contains(t, subject.Notes, item.ID.String())
	assert.NotContains(t, subject.Notes, "already matched")
}

func TestService_WarnsWhenTheReceiptIsAlreadyMatched(t *testing.T) {
	t.Parallel()

	receipt := &bankreceipt.BankReceipt{
		ID:          pulid.MustNew("brcpt_"),
		AmountMinor: 5_000,
		Status:      bankreceipt.StatusMatched,
	}
	subjects := &Service{
		receipts:  &fakeReceiptService{receipt: receipt},
		workItems: &fakeWorkItemRepo{},
		logger:    zap.NewNop(),
	}

	subject, err := subjects.Describe(
		t.Context(),
		pagination.TenantInfo{},
		agent.SubjectBankReceipt,
		receipt.ID,
	)
	require.NoError(t, err)
	assert.Contains(t, subject.Notes, "already matched")
	assert.NotContains(t, subject.Notes, "workItem")
}

func TestService_BankReceiptWithoutServiceStillHasAnID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("brcpt_")
	subject, err := (&Service{logger: zap.NewNop()}).
		Describe(t.Context(), pagination.TenantInfo{}, agent.SubjectBankReceipt, id)
	require.NoError(t, err)
	assert.Equal(t, id.String(), subject.ID)
	assert.Equal(t, "Bank receipt", subject.Label)
}

type fakeWorkers struct {
	workerports.WorkerRepository

	worker *worker.Worker
}

func (f *fakeWorkers) GetByID(
	context.Context,
	workerports.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	return f.worker, nil
}

// Nothing Confidential reaches a model, whoever is asking. A driver's licence
// number is Confidential; its expiry is what a desk's work turns on.
func TestService_DescribesADriverWithoutTheirLicenceNumber(t *testing.T) {
	t.Parallel()

	subjects := &Service{
		workers: &fakeWorkers{worker: &worker.Worker{
			FirstName: "Ada",
			LastName:  "Lovelace",
			Profile: &worker.WorkerProfile{
				LicenseNumber: "D1234-5678-9012",
				LicenseExpiry: 1893456000,
			},
		}},
		logger: zap.NewNop(),
	}

	subject, err := subjects.Describe(
		t.Context(),
		pagination.TenantInfo{},
		agent.SubjectWorker,
		pulid.MustNew("wrk_"),
	)

	require.NoError(t, err)
	assert.NotContains(t, subject.Notes, "D1234-5678-9012")
	assert.NotContains(t, subject.Notes, "licenceNumber")
	assert.Contains(t, subject.Notes, "licenceExpiry")
}
