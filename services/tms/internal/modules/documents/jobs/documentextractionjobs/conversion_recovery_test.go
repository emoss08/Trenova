package documentextractionjobs

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/documentconversionconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/searchconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/documentmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentextractionservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentanalysis"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionjobs/documentextractioncontract"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

type executionState int

const (
	executionRunning executionState = iota
	executionCompleted
	executionFailed
)

/*
policyStarter answers a start the way the Temporal server does for one
workflow ID: a running execution is refused unless the conflict policy says to
use it, and a closed one is reused or refused by the reuse policy.
*/
type policyStarter struct {
	mu         sync.Mutex
	executions map[string]executionState
	started    []*documentextractioncontract.ProcessDocumentExtractionPayload
	options    []client.StartWorkflowOptions
}

func newPolicyStarter() *policyStarter {
	return &policyStarter{executions: make(map[string]executionState)}
}

func (s *policyStarter) Enabled() bool { return true }

func (s *policyStarter) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	_ any,
	args ...any,
) (client.WorkflowRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if state, ok := s.executions[options.ID]; ok {
		alreadyStarted := serviceerror.NewWorkflowExecutionAlreadyStarted(
			"workflow execution already started",
			"",
			"",
		)
		switch state {
		case executionRunning:
			if options.WorkflowIDConflictPolicy != enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING {
				return nil, alreadyStarted
			}
			return nil, nil
		case executionCompleted, executionFailed:
			switch options.WorkflowIDReusePolicy {
			case enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE:
				return nil, alreadyStarted
			case enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY:
				if state != executionFailed {
					return nil, alreadyStarted
				}
			}
		}
	}

	s.executions[options.ID] = executionRunning
	s.options = append(s.options, options)
	if len(args) == 1 {
		if payload, ok := args[0].(*documentextractioncontract.ProcessDocumentExtractionPayload); ok {
			s.started = append(s.started, payload)
		}
	}

	return nil, nil
}

func (s *policyStarter) finish(id string, state executionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executions[id] = state
}

func (s *policyStarter) last() *documentextractioncontract.ProcessDocumentExtractionPayload {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.started) == 0 {
		return nil
	}

	return s.started[len(s.started)-1]
}

func (*policyStarter) CancelWorkflow(context.Context, string, string) error { return nil }

func (*policyStarter) SignalWorkflow(context.Context, string, string, string, any) error {
	return nil
}

func (h *doclingHarness) runWorkflow(
	t *testing.T,
	payload *documentextractioncontract.ProcessDocumentExtractionPayload,
) (*documentanalysis.ProcessDocumentIntelligenceResult, error) {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(h.activities)

	env.ExecuteWorkflow(ProcessDocumentExtractionWorkflow, payload)
	require.True(t, env.IsWorkflowCompleted())
	if err := env.GetWorkflowError(); err != nil {
		return nil, err
	}

	var result *documentanalysis.ProcessDocumentIntelligenceResult
	require.NoError(t, env.GetWorkflowResult(&result))

	return result, nil
}

func (h *doclingHarness) expectSucceedingConversion(t *testing.T) []kernelports.ConvertedPage {
	t.Helper()

	golden := doclingGoldenPages(t, "rate-confirmation.pdf.golden.json")
	h.converter.EXPECT().Enabled().Return(true)
	h.converter.EXPECT().
		Submit(mock.Anything, mock.Anything).
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

	return golden
}

func (h *doclingHarness) contentService(
	t *testing.T,
	starter kernelports.WorkflowStarter,
) *documentextractionservice.Service {
	t.Helper()

	return documentextractionservice.New(documentextractionservice.Params{
		Logger:                   zap.NewNop(),
		DocumentConversionConfig: &documentconversionconfig.Config{},
		SearchConfig:             &searchconfig.Config{},
		Metrics: &metrics.Registry{
			Document: documentmetrics.NewDocument(prometheus.NewRegistry(), zap.NewNop(), false),
		},
		DocumentControlRepo: h.controlRepo,
		DocumentRepo:        h.docs,
		ContentRepo:         h.contents,
		DraftRepo:           h.drafts,
		WorkflowStarter:     starter,
	})
}

/*
A run stopped for want of a provider completes successfully, so the workflow ID
it used is closed and not failed. After an administrator adds a provider,
Reextract must be able to start that ID again; under a failed-only reuse policy
the start was refused, swallowed, and the document sat Pending for good.
*/
func TestReextractAfterAStoppedRunStartsANewConversion(t *testing.T) {
	t.Parallel()

	h := newDoclingHarness(t, false)
	starter := newPolicyStarter()
	service := h.contentService(t, starter)
	tenantInfo := pagination.TenantInfo{
		OrgID:  h.payload.OrganizationID,
		BuID:   h.payload.BusinessUnitID,
		UserID: h.payload.UserID,
	}
	workflowID := documentextractioncontract.WorkflowID(h.payload.DocumentID)

	require.NoError(t, service.EnqueueExtraction(t.Context(), h.docs.Doc, h.payload.UserID))
	first := starter.last()
	require.NotNil(t, first)

	stopped, err := h.runWorkflow(t, first)
	require.NoError(t, err)
	assert.Equal(t, string(document.ContentStatusIndexed), stopped.Status)
	assert.Equal(
		t,
		documentupload.FailureExtractionUnavailable.String(),
		h.contents.Content.FailureCode,
	)
	starter.finish(workflowID, executionCompleted)

	h.activities.aiDocumentService = extractionAvailability{available: true}
	golden := h.expectSucceedingConversion(t)

	require.NoError(t, service.Reextract(t.Context(), h.payload.DocumentID, tenantInfo))
	require.Len(t, starter.started, 2, "Reextract must start a new run on the closed ID")
	assert.Equal(
		t,
		enums.WORKFLOW_ID_CONFLICT_POLICY_FAIL,
		starter.options[1].WorkflowIDConflictPolicy,
	)

	converted, err := h.runWorkflow(t, starter.last())
	require.NoError(t, err)
	assert.Equal(t, string(document.ContentStatusIndexed), converted.Status)
	assert.Equal(t, documentcontent.StatusIndexed, h.contents.Content.Status)
	assert.Empty(t, h.contents.Content.FailureCode)
	assert.Len(t, h.contents.Pages, len(golden))
	assert.True(t, h.docs.Last().HasExtractedText)
}

/*
A running extraction is still never duplicated: the conflict policy refuses a
second start while the first is open, and the service treats that as done.
*/
func TestEnqueueExtractionDoesNotDuplicateARunningConversion(t *testing.T) {
	t.Parallel()

	h := newDoclingHarness(t, true)
	starter := newPolicyStarter()
	service := h.contentService(t, starter)

	require.NoError(t, service.EnqueueExtraction(t.Context(), h.docs.Doc, h.payload.UserID))
	require.NoError(t, service.EnqueueExtraction(t.Context(), h.docs.Doc, h.payload.UserID))
	assert.Len(t, starter.started, 1)
}

/*
A sidecar that stays down exhausts the convert activity's retries. planDocling
already cleared the text and set Extracting, so the workflow records the
failure itself before it returns; otherwise the document reads Extracting
forever and the reconcile converts it again every two hours.
*/
func TestProcessDocumentExtractionWorkflow_ExhaustedConversionEndsFailed(t *testing.T) {
	t.Parallel()

	h := newDoclingHarness(t, true)
	h.converter.EXPECT().Enabled().Return(true)
	h.converter.EXPECT().
		Submit(mock.Anything, mock.Anything).
		Return(kernelports.ConversionTaskID(""), kernelports.ErrDocumentConverterUnavailable).
		Times(int(convertRetryPolicy.MaximumAttempts))

	_, err := h.runWorkflow(t, h.payload)
	require.Error(t, err)

	content := h.contents.Content
	require.NotNil(t, content)
	assert.Equal(t, documentcontent.StatusFailed, content.Status)
	assert.Equal(t, documentupload.FailureConverterUnavailable.String(), content.FailureCode)
	assert.Equal(t, converterUnreachableMessage, content.FailureMessage)
	last := h.docs.Last()
	assert.Equal(t, document.ContentStatusFailed, last.ContentStatus)
	assert.Equal(t, document.ShipmentDraftStatusFailed, last.ShipmentDraftStatus)
}

/*
A converter that refuses the API key will refuse every retry too. It fails the
document at once, with a configuration code distinct from a file Docling could
not read, so an administrator knows the fix is the key and not the upload.
*/
func TestProcessDocumentExtractionWorkflow_RefusedAPIKeyFailsAtOnceWithTheConfigurationCode(
	t *testing.T,
) {
	t.Parallel()

	h := newDoclingHarness(t, true)
	h.converter.EXPECT().Enabled().Return(true)
	h.converter.EXPECT().
		Submit(mock.Anything, mock.Anything).
		Return(kernelports.ConversionTaskID(""), kernelports.ErrDocumentConverterMisconfigured).
		Once()

	result, err := h.runWorkflow(t, h.payload)
	require.NoError(t, err)
	assert.Equal(t, string(documentcontent.StatusFailed), result.Status)

	content := h.contents.Content
	require.NotNil(t, content)
	assert.Equal(t, documentcontent.StatusFailed, content.Status)
	assert.Equal(t, documentupload.FailureConverterMisconfigured.String(), content.FailureCode)
	assert.Equal(t, converterMisconfiguredMessage, content.FailureMessage)
	assert.Equal(t, document.ContentStatusFailed, h.docs.Last().ContentStatus)
}
