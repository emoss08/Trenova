package factoringservicetest

import (
	"bytes"
	"context"
	"io"
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingfake"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringaction"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringreadiness"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
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

const Now = int64(1_800_000_000)

type FakeSubmissions struct {
	factoringsubmissionports.SubmissionRepository
	Mu      sync.Mutex
	ByID    map[pulid.ID]*factoringsubmission.Submission
	Created int
	Docs    *factoringsubmissionports.SetSubmissionDocumentsRequest
}

func NewFakeSubmissions() *FakeSubmissions {
	return &FakeSubmissions{ByID: map[pulid.ID]*factoringsubmission.Submission{}}
}

func (f *FakeSubmissions) Create(
	_ context.Context,
	e *factoringsubmission.Submission,
) (*factoringsubmission.Submission, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	e.ID = pulid.MustNew("facsub_")
	e.CreatedAt = Now
	for idx, item := range e.Items {
		item.ID = pulid.MustNew("facsit_")
		item.SubmissionID = e.ID
		item.Position = idx + 1
	}
	f.ByID[e.ID] = factoringaction.CloneSubmission(e)
	f.Created++
	return e, nil
}

func (f *FakeSubmissions) Update(
	_ context.Context,
	e *factoringsubmission.Submission,
) (*factoringsubmission.Submission, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	current := f.ByID[e.ID]
	if current == nil || current.Version != e.Version {
		return nil, errortypes.NewConflictError("version")
	}
	e.Version++
	f.ByID[e.ID] = factoringaction.CloneSubmission(e)
	return e, nil
}

func (f *FakeSubmissions) GetByID(
	_ context.Context,
	req factoringsubmissionports.GetSubmissionRequest,
) (*factoringsubmission.Submission, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	found := f.ByID[req.ID]
	if found == nil {
		return nil, errortypes.NewNotFoundError("Factoring submission not found")
	}
	return factoringaction.CloneSubmission(found), nil
}

func (f *FakeSubmissions) GetByIdempotencyKey(
	_ context.Context,
	req factoringsubmissionports.GetSubmissionByKeyRequest,
) (*factoringsubmission.Submission, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	for _, sub := range f.ByID {
		if sub.IdempotencyKey == req.IdempotencyKey {
			return factoringaction.CloneSubmission(sub), nil
		}
	}
	return nil, errortypes.NewNotFoundError("Factoring submission not found")
}

func (f *FakeSubmissions) SetDocuments(
	_ context.Context,
	req *factoringsubmissionports.SetSubmissionDocumentsRequest,
) error {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.Docs = req
	sub := f.ByID[req.SubmissionID]
	sub.SchedulePDFDocumentID = req.SchedulePDFDocumentID
	sub.ScheduleCSVDocumentID = req.ScheduleCSVDocumentID
	for _, item := range sub.Items {
		if id, ok := req.Packets[item.ID]; ok {
			item.PacketDocumentID = id
		}
	}
	return nil
}

func (f *FakeSubmissions) ListLiveItems(
	context.Context,
	*factoringsubmissionports.ListLiveItemsRequest,
) (map[pulid.ID]*factoringsubmissionports.LiveItem, error) {
	return map[pulid.ID]*factoringsubmissionports.LiveItem{}, nil
}

type FakeTransports struct {
	factoringsubmissionports.TransportRepository
	Entity *factoringsubmission.Transport
	Tested *factoringsubmission.Transport
}

func (f *FakeTransports) Get(
	context.Context,
	factoringsubmissionports.GetTransportRequest,
) (*factoringsubmission.Transport, error) {
	if f.Entity == nil {
		return nil, errortypes.NewNotFoundError("Factoring transport not found")
	}
	copied := *f.Entity
	return &copied, nil
}

func (f *FakeTransports) Create(
	_ context.Context,
	e *factoringsubmission.Transport,
) (*factoringsubmission.Transport, error) {
	copied := *e
	f.Entity = &copied
	return e, nil
}

func (f *FakeTransports) Update(
	_ context.Context,
	e *factoringsubmission.Transport,
) (*factoringsubmission.Transport, error) {
	if f.Entity == nil || f.Entity.Version != e.Version {
		return nil, errortypes.NewConflictError("version")
	}
	e.Version++
	copied := *e
	f.Entity = &copied
	return e, nil
}

func (f *FakeTransports) RecordTest(_ context.Context, e *factoringsubmission.Transport) error {
	copied := *e
	f.Tested = &copied
	return nil
}

type FakeStates struct {
	factoringports.InvoiceStateRepository
	States map[pulid.ID]*factoringreadiness.InvoiceState
}

func (f *FakeStates) ListByInvoiceIDs(
	_ context.Context,
	req *factoringports.ListInvoiceStatesByInvoiceIDsRequest,
) (map[pulid.ID]*factoringreadiness.InvoiceState, error) {
	out := make(map[pulid.ID]*factoringreadiness.InvoiceState, len(req.InvoiceIDs))
	for _, id := range req.InvoiceIDs {
		if state, ok := f.States[id]; ok {
			out[id] = state
		}
	}
	return out, nil
}

type FakeAgreements struct {
	factoringports.AgreementRepository
	Agreement *factoring.Agreement
}

func (f *FakeAgreements) GetInForce(
	context.Context,
	factoringports.GetAgreementInForceRequest,
) (*factoring.Agreement, error) {
	if f.Agreement == nil {
		return nil, errortypes.NewNotFoundError("Factoring agreement not found")
	}
	return f.Agreement, nil
}

func (f *FakeAgreements) GetByID(
	context.Context,
	factoringports.GetAgreementRequest,
) (*factoring.Agreement, error) {
	return f.Agreement, nil
}

type FakeInvoices struct {
	invoiceports.InvoiceRepository
	ByID map[pulid.ID]*invoice.Invoice
}

func (f *FakeInvoices) GetByIDs(
	_ context.Context,
	req invoiceports.GetInvoicesByIDsRequest,
) ([]*invoice.Invoice, error) {
	out := make([]*invoice.Invoice, 0, len(req.InvoiceIDs))
	for _, id := range req.InvoiceIDs {
		if inv, ok := f.ByID[id]; ok {
			out = append(out, inv)
		}
	}
	return out, nil
}

func (f *FakeInvoices) GetByID(
	_ context.Context,
	req invoiceports.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	if inv, ok := f.ByID[req.ID]; ok {
		return inv, nil
	}
	return nil, errortypes.NewNotFoundError("Invoice not found")
}

type FakeFactors struct {
	factoringcompanyports.FactoringCompanyRepository
	Factor *factoringcompany.FactoringCompany
}

func (f *FakeFactors) GetByID(
	context.Context,
	factoringcompanyports.GetFactoringCompanyByIDRequest,
) (*factoringcompany.FactoringCompany, error) {
	return f.Factor, nil
}

type FakeCustomers struct {
	customerports.CustomerRepository
	Customers []*customer.Customer
}

func (f *FakeCustomers) GetByIDs(
	context.Context,
	customerports.GetCustomersByIDsRequest,
) ([]*customer.Customer, error) {
	return f.Customers, nil
}

type FakeOrganizations struct {
	tenantports.OrganizationRepository
}

func (FakeOrganizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Name: "Trenova Logistics", Timezone: "America/Chicago"}, nil
}

type FakeSequences struct {
	seqgen.Generator
	next int
}

func (f *FakeSequences) Generate(context.Context, *seqgen.GenerateRequest) (string, error) {
	f.next++
	return "FS" + string(rune('0'+f.next)), nil
}

type FakeTemplates struct {
	documenttemplateports.DocumentTemplateResolver
	Messages []any
}

func (f *FakeTemplates) RenderDocument(
	context.Context,
	*documenttemplateports.RenderDocumentRequest,
) (*documenttemplateports.RenderedDocument, error) {
	return &documenttemplateports.RenderedDocument{PDF: []byte("%PDF-schedule")}, nil
}

func (f *FakeTemplates) RenderMessage(
	_ context.Context,
	req *documenttemplateports.RenderMessageRequest,
) (*documenttemplateports.RenderedMessage, error) {
	f.Messages = append(f.Messages, req.Data)
	return &documenttemplateports.RenderedMessage{Subject: "Schedule", Text: "Attached"}, nil
}

type FakeDocumentStore struct {
	Mu     sync.Mutex
	Stored map[pulid.ID]*documentports.StoreDocumentRequest
}

func (f *FakeDocumentStore) Store(
	_ context.Context,
	req *documentports.StoreDocumentRequest,
) (*document.Document, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	id := pulid.MustNew("doc_")
	f.Stored[id] = req
	return &document.Document{ID: id}, nil
}

type FakeDocumentReader struct {
	documentports.InvoiceDocumentService
	Store *FakeDocumentStore
}

func (f *FakeDocumentReader) GetDownloadContent(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*documentports.DocumentContent, error) {
	f.Store.Mu.Lock()
	defer f.Store.Mu.Unlock()
	stored, ok := f.Store.Stored[req.ID]
	if !ok {
		return &documentports.DocumentContent{
			Body: io.NopCloser(bytes.NewReader([]byte("%PDF-invoice"))),
		}, nil
	}
	return &documentports.DocumentContent{Body: io.NopCloser(bytes.NewReader(stored.Content))}, nil
}

type FakeDocuments struct {
	documentports.DocumentRepository
}

func (FakeDocuments) GetByResourceIDs(
	context.Context,
	*documentports.GetDocumentsByResourceIDsRequest,
) ([]*document.Document, error) {
	return nil, nil
}

func FakeImaging() *imagingfake.Fake {
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

type FakeCipher struct{}

func (FakeCipher) Seal(
	_ context.Context,
	plaintext string,
	scope *kernelports.SecretScope,
) (string, error) {
	return "sealed:" + scope.ResourceID + ":" + plaintext, nil
}

func (FakeCipher) Open(
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

type FakeSender struct {
	TransportMethod factoringsubmission.TransportMethod
	Requests        []*factoringsubmissionports.TransportSendRequest
	err             error
	TestErr         error
}

func (f *FakeSender) Method() factoringsubmission.TransportMethod { return f.TransportMethod }

func (f *FakeSender) Send(
	_ context.Context,
	req *factoringsubmissionports.TransportSendRequest,
) (*factoringsubmissionports.TransportSendResult, error) {
	f.Requests = append(f.Requests, req)
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

func (f *FakeSender) Test(context.Context, *factoringsubmissionports.TransportTestRequest) error {
	return f.TestErr
}

type FakeStarter struct {
	Started []*factoringsubmission.Submission
}

func (f *FakeStarter) StartSend(
	_ context.Context,
	_ pagination.TenantInfo,
	sub *factoringsubmission.Submission,
) {
	f.Started = append(f.Started, factoringaction.CloneSubmission(sub))
}

type FakeRequester struct {
	Targets []*factoringports.EvaluationTargets
}

func (f *FakeRequester) RequestEvaluation(
	_ context.Context,
	_ pagination.TenantInfo,
	targets *factoringports.EvaluationTargets,
) {
	f.Targets = append(f.Targets, targets)
}

type FakeEvents struct {
	Mu     sync.Mutex
	Events []agenteventports.AgentEvent
}

func (f *FakeEvents) Publish(_ context.Context, event *agenteventports.AgentEvent) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.Events = append(f.Events, *event)
}

func (f *FakeEvents) Kinds() []string {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	out := make([]string, 0, len(f.Events))
	for _, event := range f.Events {
		out = append(out, string(event.Kind))
	}
	slices.Sort(out)
	return out
}

func FakeOrganizationValue() *tenant.Organization {
	return &tenant.Organization{Name: "Trenova Logistics", Timezone: "America/Chicago"}
}
