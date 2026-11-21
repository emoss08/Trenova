package documentextractionjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/documentconversionconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/documentmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentaiextractionjobs"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentanalysis"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionsupport"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const capturedPDF = "%PDF-1.7 captured scan"

type captureHarness struct {
	analyzer  captureports.CaptureAnalyzer
	converter *kernelportsmock.MockDocumentConverter
	control   *documentcontrol.DocumentControl
	tenant    pagination.TenantInfo
	assembled int
}

func newCaptureHarness(
	t *testing.T,
	availability documentaiextractionports.AIDocumentService,
) *captureHarness {
	t.Helper()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	control := documentcontrol.NewDefaultDocumentControl(tenant.OrgID, tenant.BuID)
	controlRepo := tenantportsmock.NewMockDocumentControlRepository(t)
	controlRepo.EXPECT().GetOrCreate(mock.Anything, tenant.OrgID, tenant.BuID).Return(control, nil)
	converter := kernelportsmock.NewMockDocumentConverter(t)
	registry := &metrics.Registry{
		Document: documentmetrics.NewDocument(prometheus.NewRegistry(), zap.NewNop(), false),
	}

	activities := &Activities{
		logger:            zap.NewNop(),
		cfg:               &aiconfig.Config{},
		metrics:           registry,
		aiDocumentService: availability,
		converter:         converter,
		conversionCfg:     &documentconversionconfig.Config{},
		support: documentextractionsupport.New(documentextractionsupport.Params{
			Logger:              zap.NewNop(),
			Metrics:             registry,
			DocumentControlRepo: controlRepo,
		}),
	}

	return &captureHarness{
		analyzer:  NewCaptureAnalyzer(activities),
		converter: converter,
		control:   control,
		tenant:    tenant,
	}
}

func (h *captureHarness) analyze(t *testing.T) *captureports.CaptureAnalysis {
	t.Helper()

	analysis, err := h.analyzer.Analyze(t.Context(), &captureports.CaptureAnalysisRequest{
		TenantInfo: h.tenant,
		FileName:   "item.pdf",
		Assemble: func(context.Context) ([]byte, error) {
			h.assembled++
			return []byte(capturedPDF), nil
		},
	})
	require.NoError(t, err)
	require.NotNil(t, analysis)

	return analysis
}

func (h *captureHarness) expectConversion(t *testing.T, wantOCR bool) {
	t.Helper()

	h.converter.EXPECT().Enabled().Return(true)
	h.converter.EXPECT().
		Submit(mock.Anything, mock.MatchedBy(func(req *kernelports.ConvertRequest) bool {
			return req.OCR == wantOCR && req.Size == int64(len(capturedPDF))
		})).
		Return(kernelports.ConversionTaskID("task-capture"), nil).
		Once()
	h.converter.EXPECT().
		Status(mock.Anything, kernelports.ConversionTaskID("task-capture")).
		Return(kernelports.ConversionStatus{State: kernelports.ConversionStateSucceeded}, nil).
		Once()
	h.converter.EXPECT().
		Result(mock.Anything, kernelports.ConversionTaskID("task-capture")).
		Return(&kernelports.ConvertedDocument{
			Pages: []kernelports.ConvertedPage{{Number: 1, Text: rateConfirmationText}},
		}, nil).
		Once()
}

/*
With document intelligence off, a captured scan is not read at all: no PDF is
assembled for it and the converter is never reached (the converter mock carries
no expectations, so any call fails the test). The item is left for a person to
route.
*/
func TestCaptureAnalyzer_ReadsNothingWhenDocumentIntelligenceIsOff(t *testing.T) {
	t.Parallel()

	h := newCaptureHarness(t, extractionAvailability{available: true})
	h.control.EnableDocumentIntelligence = false

	analysis := h.analyze(t)

	assert.Equal(t, &captureports.CaptureAnalysis{}, analysis)
	assert.Zero(t, h.assembled)
}

func TestCaptureAnalyzer_ReadsNothingWithoutAModelProvider(t *testing.T) {
	t.Parallel()

	h := newCaptureHarness(t, extractionAvailability{
		blocker: documentaiextractionports.ExtractionNotInPlan,
	})

	analysis := h.analyze(t)

	assert.Equal(t, &captureports.CaptureAnalysis{}, analysis)
	assert.Zero(t, h.assembled)
}

func TestCaptureAnalyzer_ConvertsWithoutOCRWhenOCRIsOff(t *testing.T) {
	t.Parallel()

	h := newCaptureHarness(t, extractionAvailability{available: true})
	h.control.EnableOCR = false
	h.expectConversion(t, false)

	analysis := h.analyze(t)

	assert.Equal(t, 1, h.assembled)
	assert.Equal(t, documentanalysis.KindRateConfirmation, analysis.Kind)
}

func TestCaptureAnalyzer_ConvertsWithOCRWhenOCRIsOn(t *testing.T) {
	t.Parallel()

	h := newCaptureHarness(t, extractionAvailability{available: true})
	h.expectConversion(t, true)

	analysis := h.analyze(t)

	inferred, ok := documentaiextractionjobs.InferDocumentType(
		documentanalysis.KindRateConfirmation,
	)
	require.True(t, ok)
	assert.Equal(t, 1, h.assembled)
	assert.Equal(t, documentanalysis.KindRateConfirmation, analysis.Kind)
	assert.Equal(t, inferred.Code, analysis.DocumentTypeCode)
}

func TestCaptureAnalyzer_SuggestsNoDocumentTypeWhenTypeFilingIsOff(t *testing.T) {
	t.Parallel()

	h := newCaptureHarness(t, extractionAvailability{available: true})
	h.control.EnableAutoDocumentTypeAssociate = false
	h.expectConversion(t, true)

	analysis := h.analyze(t)

	assert.Equal(t, documentanalysis.KindRateConfirmation, analysis.Kind)
	assert.Empty(t, analysis.DocumentTypeCode)
}
