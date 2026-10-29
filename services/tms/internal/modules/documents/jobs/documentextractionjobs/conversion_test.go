package documentextractionjobs

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/documentconversionconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/documentmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/documents/app/storedobject"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentanalysis"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionjobs/documentextractioncontract"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionsupport"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractiontest"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/objectstoretest"
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
	documentextractionsupport.NoopAIDocumentService
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
	activities  *Activities
	controlRepo *tenantportsmock.MockDocumentControlRepository
	docs        *documentextractiontest.MemoryDocumentRepository
	contents    *documentextractiontest.MemoryContentRepository
	drafts      *documentextractiontest.MemoryDraftRepository
	converter   *kernelportsmock.MockDocumentConverter
	storage     *objectstoretest.Store
	payload     *documentextractioncontract.ProcessDocumentExtractionPayload
}

const testDocumentText = "%PDF-1.7 rate confirmation"

func newDoclingHarness(t *testing.T, providerAvailable bool) *doclingHarness {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	docID := pulid.MustNew("doc_")

	control := documentcontrol.NewDefaultDocumentControl(orgID, buID)
	control.EnableAutoDocumentTypeAssociate = false
	controlRepo := tenantportsmock.NewMockDocumentControlRepository(t)
	controlRepo.EXPECT().GetOrCreate(mock.Anything, orgID, buID).Return(control, nil)

	original := []byte(testDocumentText)
	docs := &documentextractiontest.MemoryDocumentRepository{Doc: &document.Document{
		ID:                docID,
		ProcessingProfile: document.ProcessingProfileRateConfirmationImport,
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		OriginalName:      "rate-confirmation.pdf",
		FileType:          "application/pdf",
		StoragePath:       "documents/rate-confirmation.pdf",
		FileSize:          int64(len(original)),
		ResourceType:      "shipment",
		ResourceID:        pulid.MustNew("shp_").String(),
		ContentStatus:     document.ContentStatusPending,
		UploadedByID:      userID,
	}}
	contents := &documentextractiontest.MemoryContentRepository{}
	drafts := &documentextractiontest.MemoryDraftRepository{}
	converter := kernelportsmock.NewMockDocumentConverter(t)
	storage := objectstoretest.New()
	encryption := encryptionservice.NewWithKeyManager(
		encryptionservice.NewLocalKeyManager("docling-harness-key-material"),
	)
	key := docs.Doc.StoragePath
	sealed, err := encryption.SealBlob(
		t.Context(),
		bytes.NewReader(original),
		int64(len(original)),
		storedobject.AAD(pagination.TenantInfo{OrgID: orgID, BuID: buID}, key),
	)
	require.NoError(t, err)
	body, err := io.ReadAll(sealed)
	require.NoError(t, err)
	storage.Put(key, body)

	activities := &Activities{
		logger: zap.NewNop(),
		cfg:    &aiconfig.Config{},
		metrics: &metrics.Registry{
			Document: documentmetrics.NewDocument(prometheus.NewRegistry(), zap.NewNop(), false),
		},
		documentRepo:      docs,
		contentRepo:       contents,
		draftRepo:         drafts,
		aiDocumentService: extractionAvailability{available: providerAvailable},
		storage:           storage,
		encryption:        encryption,
		workflowStarter:   noopWorkflowStarter{},
		converter:         converter,
		conversionCfg:     &documentconversionconfig.Config{DoclingURL: "http://docling:5001"},
		support: documentextractionsupport.New(documentextractionsupport.Params{
			Logger: zap.NewNop(),
			Metrics: &metrics.Registry{
				Document: documentmetrics.NewDocument(
					prometheus.NewRegistry(),
					zap.NewNop(),
					false,
				),
			},
			DocumentControlRepo: controlRepo,
			DraftRepo:           drafts,
		}),
	}

	return &doclingHarness{
		activities:  activities,
		controlRepo: controlRepo,
		docs:        docs,
		contents:    contents,
		drafts:      drafts,
		converter:   converter,
		storage:     storage,
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

func (h *doclingHarness) run(t *testing.T) *documentanalysis.ProcessDocumentIntelligenceResult {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(h.activities)

	env.ExecuteWorkflow(ProcessDocumentExtractionWorkflow, h.payload)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result *documentanalysis.ProcessDocumentIntelligenceResult
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

	last := h.docs.Last()
	assert.Equal(t, document.ContentStatusIndexed, last.ContentStatus)
	assert.Equal(t, document.ShipmentDraftStatusUnavailable, last.ShipmentDraftStatus)
	assert.False(t, last.HasExtractedText)

	require.NotNil(t, h.contents.Content)
	assert.Equal(t, documentcontent.StatusIndexed, h.contents.Content.Status)
	assert.Equal(
		t,
		documentupload.FailureExtractionUnavailable.String(),
		h.contents.Content.FailureCode,
	)
	assert.Equal(t, extractionNoProviderMessage, h.contents.Content.FailureMessage)
	assert.Empty(t, h.contents.Pages)
	assert.Empty(t, h.drafts.Drafts)
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

			require.NotNil(t, h.contents.Content)
			assert.Equal(
				t,
				documentupload.FailureExtractionUnavailable.String(),
				h.contents.Content.FailureCode,
			)
			assert.Equal(t, tt.message, h.contents.Content.FailureMessage)
			assert.Empty(t, h.contents.Pages)
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

	h.converter.EXPECT().Enabled().Return(true)
	h.converter.EXPECT().
		Submit(mock.Anything, mock.MatchedBy(func(req *kernelports.ConvertRequest) bool {
			file, err := req.Open(context.Background())
			if err != nil {
				return false
			}
			defer file.Close()
			body, err := io.ReadAll(file)
			return err == nil &&
				req.Size == int64(len(body)) &&
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

	content := h.contents.Content
	require.NotNil(t, content)
	assert.Equal(t, documentcontent.StatusIndexed, content.Status)
	assert.Equal(t, "1.36.0", content.ConverterVersion)
	assert.Equal(t, len(golden), content.PageCount)

	require.Len(t, h.contents.Pages, len(golden))
	for i, page := range h.contents.Pages {
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

	require.NotEmpty(t, h.drafts.Drafts)
	last := h.docs.Last()
	assert.Equal(t, document.ContentStatusIndexed, last.ContentStatus)
	assert.True(t, last.HasExtractedText)
	assert.Equal(t, h.drafts.Drafts[len(h.drafts.Drafts)-1].DocumentID, h.payload.DocumentID)
}

/*
Legacy Word, Excel, PowerPoint and RTF files are refused by type before the
sidecar sees them; Docling cannot read them and a person can save them as a
format it can.
*/
func TestProcessDocumentExtractionWorkflow_RefusesLegacyOfficeFilesBeforeConverting(t *testing.T) {
	t.Parallel()

	h := newDoclingHarness(t, true)
	h.docs.Doc.OriginalName = "rate-confirmation.doc"
	h.docs.Doc.FileType = "application/msword"

	result := h.run(t)

	require.NotNil(t, result)
	require.NotNil(t, h.contents.Content)
	assert.Equal(t, documentcontent.StatusFailed, h.contents.Content.Status)
	assert.Equal(
		t,
		documentupload.FailureUnsupportedFormat.String(),
		h.contents.Content.FailureCode,
	)
	assert.Empty(t, h.contents.Pages)
}

/*
A sidecar that restarted has forgotten its tasks. The attempt submits the file
again rather than failing the document.
*/
func TestProcessDocumentExtractionWorkflow_ResubmitsWhenTheConverterLosesTheTask(t *testing.T) {
	t.Parallel()

	h := newDoclingHarness(t, true)
	golden := doclingGoldenPages(t, "rate-confirmation.pdf.golden.json")

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
			file, err := req.Open(context.Background())
			if err != nil {
				return false
			}
			defer file.Close()
			body, err := io.ReadAll(file)
			return err == nil && string(body) == testDocumentText
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
	assert.Equal(t, documentcontent.StatusIndexed, h.contents.Content.Status)
	assert.Len(t, h.contents.Pages, len(golden))
}
