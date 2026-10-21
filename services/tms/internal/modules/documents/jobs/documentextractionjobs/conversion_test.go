package documentextractionjobs

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionjobs/documentextractioncontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentcontentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/modules/operations/ports/documentshipmentdraftports"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

type extractionAvailability struct {
	noopAIDocumentService
	available bool
	blocker   documentaiextractionports.ExtractionAvailability
}

func (s extractionAvailability) ExtractionAvailability(
	context.Context,
	pagination.TenantInfo,
) (documentaiextractionports.ExtractionAvailability, error) {
	switch {
	case s.available:
		return documentaiextractionports.ExtractionAvailable, nil
	case s.blocker != "":
		return s.blocker, nil
	default:
		return documentaiextractionports.ExtractionNoProviderAssigned, nil
	}
}

type memoryDocumentRepository struct {
	documentports.DocumentRepository

	mu      sync.Mutex
	doc     *document.Document
	updates []documentports.UpdateDocumentIntelligenceRequest
}

func (r *memoryDocumentRepository) GetByID(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if req.ID != r.doc.ID {
		return nil, errortypes.NewNotFoundError("document not found")
	}
	clone := *r.doc

	return &clone, nil
}

func (r *memoryDocumentRepository) UpdateIntelligence(
	_ context.Context,
	req *documentports.UpdateDocumentIntelligenceRequest,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updates = append(r.updates, *req)
	r.doc.ContentStatus = req.ContentStatus
	r.doc.ContentError = req.ContentError
	r.doc.DetectedKind = req.DetectedKind
	r.doc.HasExtractedText = req.HasExtractedText
	r.doc.ShipmentDraftStatus = req.ShipmentDraftStatus
	r.doc.DocumentTypeID = req.DocumentTypeID

	return nil
}

func (r *memoryDocumentRepository) last() documentports.UpdateDocumentIntelligenceRequest {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.updates[len(r.updates)-1]
}

type memoryContentRepository struct {
	documentcontentports.DocumentContentRepository

	mu      sync.Mutex
	content *documentcontent.Content
	pages   []*documentcontent.Page
	writes  []documentcontent.Content
}

func (r *memoryContentRepository) GetByDocumentID(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentcontent.Content, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.content == nil {
		return nil, errortypes.NewNotFoundError("content not found")
	}
	clone := *r.content

	return &clone, nil
}

func (r *memoryContentRepository) ListPagesByDocumentID(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) ([]*documentcontent.Page, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*documentcontent.Page, 0, len(r.pages))
	for _, page := range r.pages {
		clone := *page
		out = append(out, &clone)
	}

	return out, nil
}

func (r *memoryContentRepository) ReplacePages(
	_ context.Context,
	_ *documentcontent.Content,
	pages []*documentcontent.Page,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pages = make([]*documentcontent.Page, 0, len(pages))
	for _, page := range pages {
		stored := *page
		stored.Metadata = jsonRoundTrip(page.Metadata)
		r.pages = append(r.pages, &stored)
	}

	return nil
}

func (r *memoryContentRepository) Upsert(
	_ context.Context,
	entity *documentcontent.Content,
) (*documentcontent.Content, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.content != nil {
		entity.ID = r.content.ID
	}
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("dc_")
	}
	stored := *entity
	r.content = &stored
	r.writes = append(r.writes, stored)

	return entity, nil
}

type memoryDraftRepository struct {
	documentshipmentdraftports.DocumentShipmentDraftRepository

	mu     sync.Mutex
	drafts []documentshipmentdraft.DocumentShipmentDraft
}

func (r *memoryDraftRepository) GetByDocumentID(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentshipmentdraft.DocumentShipmentDraft, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.drafts) == 0 {
		return nil, errortypes.NewNotFoundError("draft not found")
	}
	clone := r.drafts[len(r.drafts)-1]

	return &clone, nil
}

func (r *memoryDraftRepository) Upsert(
	_ context.Context,
	entity *documentshipmentdraft.DocumentShipmentDraft,
) (*documentshipmentdraft.DocumentShipmentDraft, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.drafts = append(r.drafts, *entity)

	return entity, nil
}

func jsonRoundTrip(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}
	encoded, err := sonic.Marshal(metadata)
	if err != nil {
		panic(err)
	}
	var decoded map[string]any
	if err = sonic.Unmarshal(encoded, &decoded); err != nil {
		panic(err)
	}

	return decoded
}

func doclingGoldenPages(t *testing.T, name string) []kernelports.ConvertedPage {
	t.Helper()

	data, err := os.ReadFile(
		filepath.Join("..", "..", "..", "..", "infrastructure", "docling", "testdata", name),
	)
	require.NoError(t, err)

	var pages []kernelports.ConvertedPage
	require.NoError(t, sonic.Unmarshal(data, &pages))
	require.NotEmpty(t, pages)

	return pages
}

type doclingHarness struct {
	activities *Activities
	docs       *memoryDocumentRepository
	contents   *memoryContentRepository
	drafts     *memoryDraftRepository
	converter  *kernelportsmock.MockDocumentConverter
	storage    *kernelportsmock.MockClient
	payload    *documentextractioncontract.ProcessDocumentExtractionPayload
}

func newDoclingHarness(t *testing.T, providerAvailable bool) *doclingHarness {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	docID := pulid.MustNew("doc_")

	control := tenant.NewDefaultDocumentControl(orgID, buID)
	control.EnableAutoDocumentTypeAssociate = false
	controlRepo := tenantportsmock.NewMockDocumentControlRepository(t)
	controlRepo.EXPECT().GetOrCreate(mock.Anything, orgID, buID).Return(control, nil)

	docs := &memoryDocumentRepository{doc: &document.Document{
		ID:                docID,
		ProcessingProfile: document.ProcessingProfileRateConfirmationImport,
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		OriginalName:      "rate-confirmation.pdf",
		FileType:          "application/pdf",
		StoragePath:       "documents/rate-confirmation.pdf",
		ResourceType:      "shipment",
		ResourceID:        pulid.MustNew("shp_").String(),
		ContentStatus:     document.ContentStatusPending,
		UploadedByID:      userID,
	}}
	contents := &memoryContentRepository{}
	drafts := &memoryDraftRepository{}
	converter := kernelportsmock.NewMockDocumentConverter(t)
	storage := kernelportsmock.NewMockClient(t)

	activities := &Activities{
		logger: zap.NewNop(),
		cfg:    &config.AIConfig{},
		metrics: &metrics.Registry{
			Document: metrics.NewDocument(prometheus.NewRegistry(), zap.NewNop(), false),
		},
		documentRepo:        docs,
		documentControlRepo: controlRepo,
		contentRepo:         contents,
		draftRepo:           drafts,
		aiDocumentService:   extractionAvailability{available: providerAvailable},
		searchProjection:    noopDocumentSearchProjectionService{},
		storage:             storage,
		workflowStarter:     noopWorkflowStarter{},
		converter:           converter,
		conversionCfg:       &config.DocumentConversionConfig{DoclingURL: "http://docling:5001"},
	}

	return &doclingHarness{
		activities: activities,
		docs:       docs,
		contents:   contents,
		drafts:     drafts,
		converter:  converter,
		storage:    storage,
		payload: &documentextractioncontract.ProcessDocumentExtractionPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: orgID,
				BusinessUnitID: buID,
				UserID:         userID,
			},
			DocumentID: docID,
		},
	}
}

func (h *doclingHarness) run(t *testing.T) *ProcessDocumentIntelligenceResult {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(h.activities)

	env.ExecuteWorkflow(ProcessDocumentExtractionWorkflow, h.payload)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result *ProcessDocumentIntelligenceResult
	require.NoError(t, env.GetWorkflowResult(&result))

	return result
}

/*
Without a model provider nothing is read. Docling only feeds the model, so a
tenant with no provider that serves document work never reaches the sidecar:
the converter mock carries no expectations, so any call to it fails the test.
The document is left stored and indexed, with its draft marked unavailable and
the reason on the content.
*/
func TestProcessDocumentExtractionWorkflow_WithoutAProviderNeverCallsTheConverter(t *testing.T) {
	t.Parallel()

	h := newDoclingHarness(t, false)

	result := h.run(t)

	require.NotNil(t, result)
	assert.Equal(t, string(document.ContentStatusIndexed), result.Status)

	last := h.docs.last()
	assert.Equal(t, document.ContentStatusIndexed, last.ContentStatus)
	assert.Equal(t, document.ShipmentDraftStatusUnavailable, last.ShipmentDraftStatus)
	assert.False(t, last.HasExtractedText)

	require.NotNil(t, h.contents.content)
	assert.Equal(t, documentcontent.StatusIndexed, h.contents.content.Status)
	assert.Equal(
		t,
		documentupload.FailureExtractionUnavailable.String(),
		h.contents.content.FailureCode,
	)
	assert.Equal(t, extractionNoProviderMessage, h.contents.content.FailureMessage)
	assert.Empty(t, h.contents.pages)
	assert.Empty(t, h.drafts.drafts)
}

/*
Each thing that keeps a model from reading the document is named on the
content, so the person is sent to the setting that is actually missing: a
server with document extraction switched off is not fixed by adding a
provider, and a provider that serves chat does not serve document extraction.
*/
func TestProcessDocumentExtractionWorkflow_NamesWhatKeepsTheModelFromReading(t *testing.T) {
	t.Parallel()

	tests := []struct {
		blocker documentaiextractionports.ExtractionAvailability
		message string
	}{
		{documentaiextractionports.ExtractionOffOnServer, extractionOffOnServerMessage},
		{documentaiextractionports.ExtractionNotInPlan, extractionNotInPlanMessage},
		{documentaiextractionports.ExtractionNoProviderAssigned, extractionNoProviderMessage},
	}

	for _, tt := range tests {
		t.Run(string(tt.blocker), func(t *testing.T) {
			t.Parallel()

			h := newDoclingHarness(t, false)
			h.activities.aiDocumentService = extractionAvailability{blocker: tt.blocker}

			h.run(t)

			require.NotNil(t, h.contents.content)
			assert.Equal(
				t,
				documentupload.FailureExtractionUnavailable.String(),
				h.contents.content.FailureCode,
			)
			assert.Equal(t, tt.message, h.contents.content.FailureMessage)
			assert.Empty(t, h.contents.pages)
		})
	}
}

/*
A Docling tenant with a provider has its document converted, its pages stored
with Docling's layout lines, and a shipment draft built from them. The lines
are stored exactly as the converter placed them, so the Desk extraction card
can box a value on the page.
*/
func TestProcessDocumentExtractionWorkflow_ConvertsWithDoclingAndBuildsADraft(t *testing.T) {
	t.Parallel()

	h := newDoclingHarness(t, true)
	golden := doclingGoldenPages(t, "rate-confirmation.pdf.golden.json")

	h.storage.EXPECT().
		Download(mock.Anything, "documents/rate-confirmation.pdf").
		Return(&kernelports.DownloadResult{
			Body: io.NopCloser(strings.NewReader("%PDF-1.7 rate confirmation")),
		}, nil).
		Once()
	h.converter.EXPECT().Enabled().Return(true)
	h.converter.EXPECT().
		Submit(mock.Anything, mock.MatchedBy(func(req *kernelports.ConvertRequest) bool {
			body, err := io.ReadAll(req.Body)
			return err == nil &&
				string(body) == "%PDF-1.7 rate confirmation" &&
				req.FileName == "rate-confirmation.pdf" &&
				req.ContentType == "application/pdf" &&
				req.OCR
		})).
		Return(kernelports.ConversionTaskID("task-1"), nil).
		Once()
	h.converter.EXPECT().
		Status(mock.Anything, kernelports.ConversionTaskID("task-1")).
		Return(kernelports.ConversionStatus{State: kernelports.ConversionStateSucceeded}, nil).
		Once()
	h.converter.EXPECT().
		Result(mock.Anything, kernelports.ConversionTaskID("task-1")).
		Return(&kernelports.ConvertedDocument{
			ConverterVersion: "1.36.0",
			Pages:            golden,
		}, nil).
		Once()

	result := h.run(t)

	require.NotNil(t, result)
	assert.Equal(t, string(document.ContentStatusIndexed), result.Status)

	content := h.contents.content
	require.NotNil(t, content)
	assert.Equal(t, documentcontent.StatusIndexed, content.Status)
	assert.Equal(t, "1.36.0", content.ConverterVersion)
	assert.Equal(t, len(golden), content.PageCount)

	require.Len(t, h.contents.pages, len(golden))
	for i, page := range h.contents.pages {
		want := golden[i]
		assert.Equal(t, want.Number, page.PageNumber)
		assert.Equal(t, strings.TrimSpace(want.Markdown), strings.TrimSpace(page.Markdown))

		lines := documentcontent.LinesOf(page.Metadata)
		require.Len(t, lines, len(want.Lines), "page %d", want.Number)
		for j, line := range lines {
			assert.Equal(t, documentcontent.LayoutLine{
				Text: want.Lines[j].Text,
				X:    want.Lines[j].X,
				Y:    want.Lines[j].Y,
				W:    want.Lines[j].W,
				H:    want.Lines[j].H,
			}, line)
		}
	}

	require.NotEmpty(t, h.drafts.drafts)
	last := h.docs.last()
	assert.Equal(t, document.ContentStatusIndexed, last.ContentStatus)
	assert.True(t, last.HasExtractedText)
	assert.Equal(t, h.drafts.drafts[len(h.drafts.drafts)-1].DocumentID, h.payload.DocumentID)
}

/*
Legacy Word, Excel, PowerPoint and RTF files are refused by type before the
sidecar sees them; Docling cannot read them and a person can save them as a
format it can.
*/
func TestProcessDocumentExtractionWorkflow_RefusesLegacyOfficeFilesBeforeConverting(t *testing.T) {
	t.Parallel()

	h := newDoclingHarness(t, true)
	h.docs.doc.OriginalName = "rate-confirmation.doc"
	h.docs.doc.FileType = "application/msword"

	result := h.run(t)

	require.NotNil(t, result)
	require.NotNil(t, h.contents.content)
	assert.Equal(t, documentcontent.StatusFailed, h.contents.content.Status)
	assert.Equal(
		t,
		documentupload.FailureUnsupportedFormat.String(),
		h.contents.content.FailureCode,
	)
	assert.Empty(t, h.contents.pages)
}

/*
A sidecar that restarted has forgotten its tasks. The attempt submits the file
again rather than failing the document.
*/
func TestProcessDocumentExtractionWorkflow_ResubmitsWhenTheConverterLosesTheTask(t *testing.T) {
	t.Parallel()

	h := newDoclingHarness(t, true)
	golden := doclingGoldenPages(t, "rate-confirmation.pdf.golden.json")

	h.storage.EXPECT().
		Download(mock.Anything, mock.Anything).
		Return(&kernelports.DownloadResult{
			Body: io.NopCloser(strings.NewReader("%PDF-1.7 rate confirmation")),
		}, nil).
		Once()
	h.converter.EXPECT().Enabled().Return(true)
	h.converter.EXPECT().
		Submit(mock.Anything, mock.Anything).
		Return(kernelports.ConversionTaskID("task-lost"), nil).
		Once()
	h.converter.EXPECT().
		Status(mock.Anything, kernelports.ConversionTaskID("task-lost")).
		Return(kernelports.ConversionStatus{}, kernelports.ErrConversionTaskNotFound).
		Once()
	h.converter.EXPECT().
		Submit(mock.Anything, mock.MatchedBy(func(req *kernelports.ConvertRequest) bool {
			body, err := io.ReadAll(req.Body)
			return err == nil && string(body) == "%PDF-1.7 rate confirmation"
		})).
		Return(kernelports.ConversionTaskID("task-2"), nil).
		Once()
	h.converter.EXPECT().
		Status(mock.Anything, kernelports.ConversionTaskID("task-2")).
		Return(kernelports.ConversionStatus{State: kernelports.ConversionStateSucceeded}, nil).
		Once()
	h.converter.EXPECT().
		Result(mock.Anything, kernelports.ConversionTaskID("task-2")).
		Return(&kernelports.ConvertedDocument{
			ConverterVersion: "1.36.0",
			Pages:            golden,
		}, nil).
		Once()

	result := h.run(t)

	require.NotNil(t, result)
	assert.Equal(t, documentcontent.StatusIndexed, h.contents.content.Status)
	assert.Len(t, h.contents.pages, len(golden))
}
