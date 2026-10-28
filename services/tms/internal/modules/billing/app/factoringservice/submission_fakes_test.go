package factoringservice

import (
	"bytes"
	"context"
	"io"
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingfake"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
)

type fakeSubmissions struct {
	factoringsubmissionports.SubmissionRepository
	mu      sync.Mutex
	byID    map[pulid.ID]*factoring.Submission
	created int
	docs    *factoringsubmissionports.SetSubmissionDocumentsRequest
}

func newFakeSubmissions() *fakeSubmissions {
	return &fakeSubmissions{byID: map[pulid.ID]*factoring.Submission{}}
}

func (f *fakeSubmissions) Create(
	_ context.Context,
	e *factoring.Submission,
) (*factoring.Submission, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	e.ID = pulid.MustNew("facsub_")
	e.CreatedAt = testNow
	for idx, item := range e.Items {
		item.ID = pulid.MustNew("facsit_")
		item.SubmissionID = e.ID
		item.Position = idx + 1
	}
	f.byID[e.ID] = cloneSubmission(e)
	f.created++
	return e, nil
}

func (f *fakeSubmissions) Update(
	_ context.Context,
	e *factoring.Submission,
) (*factoring.Submission, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	current := f.byID[e.ID]
	if current == nil || current.Version != e.Version {
		return nil, errortypes.NewConflictError("version")
	}
	e.Version++
	f.byID[e.ID] = cloneSubmission(e)
	return e, nil
}

func (f *fakeSubmissions) GetByID(
	_ context.Context,
	req factoringsubmissionports.GetSubmissionRequest,
) (*factoring.Submission, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	found := f.byID[req.ID]
	if found == nil {
		return nil, errortypes.NewNotFoundError("Factoring submission not found")
	}
	return cloneSubmission(found), nil
}

func (f *fakeSubmissions) GetByIdempotencyKey(
	_ context.Context,
	req factoringsubmissionports.GetSubmissionByKeyRequest,
) (*factoring.Submission, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, sub := range f.byID {
		if sub.IdempotencyKey == req.IdempotencyKey {
			return cloneSubmission(sub), nil
		}
	}
	return nil, errortypes.NewNotFoundError("Factoring submission not found")
}

func (f *fakeSubmissions) SetDocuments(
	_ context.Context,
	req *factoringsubmissionports.SetSubmissionDocumentsRequest,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.docs = req
	sub := f.byID[req.SubmissionID]
	sub.SchedulePDFDocumentID = req.SchedulePDFDocumentID
	sub.ScheduleCSVDocumentID = req.ScheduleCSVDocumentID
	for _, item := range sub.Items {
		if id, ok := req.Packets[item.ID]; ok {
			item.PacketDocumentID = id
		}
	}
	return nil
}

func (f *fakeSubmissions) ListLiveItems(
	context.Context,
	*factoringsubmissionports.ListLiveItemsRequest,
) (map[pulid.ID]*factoringsubmissionports.LiveItem, error) {
	return map[pulid.ID]*factoringsubmissionports.LiveItem{}, nil
}

type fakeTransports struct {
	factoringsubmissionports.TransportRepository
	entity *factoring.Transport
	tested *factoring.Transport
}

func (f *fakeTransports) Get(
	context.Context,
	factoringsubmissionports.GetTransportRequest,
) (*factoring.Transport, error) {
	if f.entity == nil {
		return nil, errortypes.NewNotFoundError("Factoring transport not found")
	}
	copied := *f.entity
	return &copied, nil
}

func (f *fakeTransports) Create(
	_ context.Context,
	e *factoring.Transport,
) (*factoring.Transport, error) {
	copied := *e
	f.entity = &copied
	return e, nil
}

func (f *fakeTransports) Update(
	_ context.Context,
	e *factoring.Transport,
) (*factoring.Transport, error) {
	if f.entity == nil || f.entity.Version != e.Version {
		return nil, errortypes.NewConflictError("version")
	}
	e.Version++
	copied := *e
	f.entity = &copied
	return e, nil
}

func (f *fakeTransports) RecordTest(_ context.Context, e *factoring.Transport) error {
	copied := *e
	f.tested = &copied
	return nil
}

type fakeStates struct {
	factoringports.InvoiceStateRepository
	states map[pulid.ID]*factoring.InvoiceState
}

func (f *fakeStates) ListByInvoiceIDs(
	_ context.Context,
	req *factoringports.ListInvoiceStatesByInvoiceIDsRequest,
) (map[pulid.ID]*factoring.InvoiceState, error) {
	out := make(map[pulid.ID]*factoring.InvoiceState, len(req.InvoiceIDs))
	for _, id := range req.InvoiceIDs {
		if state, ok := f.states[id]; ok {
			out[id] = state
		}
	}
	return out, nil
}

type fakeAgreements struct {
	factoringports.AgreementRepository
	agreement *factoring.Agreement
}

func (f *fakeAgreements) GetInForce(
	context.Context,
	factoringports.GetAgreementInForceRequest,
) (*factoring.Agreement, error) {
	if f.agreement == nil {
		return nil, errortypes.NewNotFoundError("Factoring agreement not found")
	}
	return f.agreement, nil
}

func (f *fakeAgreements) GetByID(
	context.Context,
	factoringports.GetAgreementRequest,
) (*factoring.Agreement, error) {
	return f.agreement, nil
}

type fakeInvoices struct {
	invoiceports.InvoiceRepository
	byID map[pulid.ID]*invoice.Invoice
}

func (f *fakeInvoices) GetByIDs(
	_ context.Context,
	req invoiceports.GetInvoicesByIDsRequest,
) ([]*invoice.Invoice, error) {
	out := make([]*invoice.Invoice, 0, len(req.InvoiceIDs))
	for _, id := range req.InvoiceIDs {
		if inv, ok := f.byID[id]; ok {
			out = append(out, inv)
		}
	}
	return out, nil
}

func (f *fakeInvoices) GetByID(
	_ context.Context,
	req invoiceports.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	if inv, ok := f.byID[req.ID]; ok {
		return inv, nil
	}
	return nil, errortypes.NewNotFoundError("Invoice not found")
}

type fakeFactors struct {
	factoringcompanyports.FactoringCompanyRepository
	factor *factoringcompany.FactoringCompany
}

func (f *fakeFactors) GetByID(
	context.Context,
	factoringcompanyports.GetFactoringCompanyByIDRequest,
) (*factoringcompany.FactoringCompany, error) {
	return f.factor, nil
}

type fakeCustomers struct {
	customerports.CustomerRepository
	customers []*customer.Customer
}

func (f *fakeCustomers) GetByIDs(
	context.Context,
	customerports.GetCustomersByIDsRequest,
) ([]*customer.Customer, error) {
	return f.customers, nil
}

type fakeOrganizations struct {
	tenantports.OrganizationRepository
}

func (fakeOrganizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Name: "Trenova Logistics", Timezone: "America/Chicago"}, nil
}

type fakeSequences struct {
	seqgen.Generator
	next int
}

func (f *fakeSequences) Generate(context.Context, *seqgen.GenerateRequest) (string, error) {
	f.next++
	return "FS" + string(rune('0'+f.next)), nil
}

type fakeTemplates struct {
	documenttemplateports.DocumentTemplateResolver
	messages []any
}

func (f *fakeTemplates) RenderDocument(
	context.Context,
	*documenttemplateports.RenderDocumentRequest,
) (*documenttemplateports.RenderedDocument, error) {
	return &documenttemplateports.RenderedDocument{PDF: []byte("%PDF-schedule")}, nil
}

func (f *fakeTemplates) RenderMessage(
	_ context.Context,
	req *documenttemplateports.RenderMessageRequest,
) (*documenttemplateports.RenderedMessage, error) {
	f.messages = append(f.messages, req.Data)
	return &documenttemplateports.RenderedMessage{Subject: "Schedule", Text: "Attached"}, nil
}

type fakeDocumentStore struct {
	mu     sync.Mutex
	stored map[pulid.ID]*documentports.StoreDocumentRequest
}

func (f *fakeDocumentStore) Store(
	_ context.Context,
	req *documentports.StoreDocumentRequest,
) (*document.Document, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := pulid.MustNew("doc_")
	f.stored[id] = req
	return &document.Document{ID: id}, nil
}

type fakeDocumentReader struct {
	documentports.InvoiceDocumentService
	store *fakeDocumentStore
}

func (f *fakeDocumentReader) GetDownloadContent(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*documentports.DocumentContent, error) {
	f.store.mu.Lock()
	defer f.store.mu.Unlock()
	stored, ok := f.store.stored[req.ID]
	if !ok {
		return &documentports.DocumentContent{
			Body: io.NopCloser(bytes.NewReader([]byte("%PDF-invoice"))),
		}, nil
	}
	return &documentports.DocumentContent{Body: io.NopCloser(bytes.NewReader(stored.Content))}, nil
}

type fakeDocuments struct {
	documentports.DocumentRepository
}

func (fakeDocuments) GetByResourceIDs(
	context.Context,
	*documentports.GetDocumentsByResourceIDsRequest,
) ([]*document.Document, error) {
	return nil, nil
}

func fakeImaging() *imagingfake.Fake {
	fake := imagingfake.New()
	fake.AssembleFunc = func(
		ctx context.Context,
		req *kernelports.ImagingAssembleRequest,
		yield func(*kernelports.ImagingArtifact) error,
	) ([]int, error) {
		out := make([]byte, 0, 64)
		for _, source := range req.Sources {
			data, err := imagingfake.Read(ctx, source)
			if err != nil {
				return nil, err
			}
			if source.ContentType != "application/pdf" {
				data = []byte("%PDF-image")
			}
			out = append(out, data...)
		}
		if err := yield(&kernelports.ImagingArtifact{
			ContentType: "application/pdf",
			Size:        int64(len(out)),
			Body:        bytes.NewReader(out),
		}); err != nil {
			return nil, err
		}
		return []int{len(req.Sources)}, nil
	}
	return fake
}

type fakeCipher struct{}

func (fakeCipher) Seal(
	_ context.Context,
	plaintext string,
	scope *kernelports.SecretScope,
) (string, error) {
	return "sealed:" + scope.ResourceID + ":" + plaintext, nil
}

func (fakeCipher) Open(
	_ context.Context,
	sealed string,
	scope *kernelports.SecretScope,
) (string, error) {
	prefix := "sealed:" + scope.ResourceID + ":"
	if len(sealed) < len(prefix) || sealed[:len(prefix)] != prefix {
		return "", errortypes.NewBusinessError("wrong scope")
	}
	return sealed[len(prefix):], nil
}

type fakeSender struct {
	method   factoring.TransportMethod
	requests []*factoringsubmissionports.TransportSendRequest
	err      error
	testErr  error
}

func (f *fakeSender) Method() factoring.TransportMethod { return f.method }

func (f *fakeSender) Send(
	_ context.Context,
	req *factoringsubmissionports.TransportSendRequest,
) (*factoringsubmissionports.TransportSendResult, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	if req.Render != nil {
		names := make([]string, 0, len(req.Files))
		for _, file := range req.Files {
			names = append(names, file.Name)
		}
		if _, err := req.Render(1, 1, names); err != nil {
			return nil, err
		}
	}
	return &factoringsubmissionports.TransportSendResult{Reference: "ref-1", Parts: 1}, nil
}

func (f *fakeSender) Test(context.Context, *factoringsubmissionports.TransportTestRequest) error {
	return f.testErr
}

type fakeStarter struct {
	started []*factoring.Submission
}

func (f *fakeStarter) StartSend(
	_ context.Context,
	_ pagination.TenantInfo,
	sub *factoring.Submission,
) {
	f.started = append(f.started, cloneSubmission(sub))
}

type fakeRequester struct {
	targets []*factoringports.EvaluationTargets
}

func (f *fakeRequester) RequestEvaluation(
	_ context.Context,
	_ pagination.TenantInfo,
	targets *factoringports.EvaluationTargets,
) {
	f.targets = append(f.targets, targets)
}

type fakeEvents struct {
	mu     sync.Mutex
	events []agentports.AgentEvent
}

func (f *fakeEvents) Publish(_ context.Context, event agentports.AgentEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}

func (f *fakeEvents) kinds() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.events))
	for _, event := range f.events {
		out = append(out, string(event.Kind))
	}
	slices.Sort(out)
	return out
}

func fakeOrganizationValue() *tenant.Organization {
	return &tenant.Organization{Name: "Trenova Logistics", Timezone: "America/Chicago"}
}
