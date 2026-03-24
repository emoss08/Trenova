package auditjobs

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type AuditWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *AuditWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *AuditWorkflowTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func (s *AuditWorkflowTestSuite) TestProcessAuditBatchWorkflow_Success() {
	batchID := pulid.MustNew("aeb_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	payload := &ProcessAuditBatchPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Timestamp:      time.Now().Unix(),
		},
		Entries: []*audit.Entry{
			{ID: pulid.MustNew("ael_")},
			{ID: pulid.MustNew("ael_")},
		},
		BatchID: batchID,
	}

	expectedResult := &ProcessAuditBatchResult{
		ProcessedCount: 2,
		FailedCount:    0,
		BatchID:        batchID,
		ProcessedAt:    time.Now().Unix(),
	}

	var a *Activities
	s.env.OnActivity(a.ProcessAuditBatchActivity, mock.Anything, payload).
		Return(expectedResult, nil)

	s.env.ExecuteWorkflow(ProcessAuditBatchWorkflow, payload)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result *ProcessAuditBatchResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(2, result.ProcessedCount)
	s.Equal(0, result.FailedCount)
}

func (s *AuditWorkflowTestSuite) TestProcessAuditBatchWorkflow_EmptyBatch() {
	batchID := pulid.MustNew("aeb_")

	payload := &ProcessAuditBatchPayload{
		BasePayload: temporaltype.BasePayload{
			Timestamp: time.Now().Unix(),
		},
		Entries: []*audit.Entry{},
		BatchID: batchID,
	}

	expectedResult := &ProcessAuditBatchResult{
		ProcessedCount: 0,
		FailedCount:    0,
		BatchID:        batchID,
		ProcessedAt:    time.Now().Unix(),
		Metadata: map[string]any{
			"message": "No entries to process",
		},
	}

	var a *Activities
	s.env.OnActivity(a.ProcessAuditBatchActivity, mock.Anything, payload).
		Return(expectedResult, nil)

	s.env.ExecuteWorkflow(ProcessAuditBatchWorkflow, payload)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result *ProcessAuditBatchResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(0, result.ProcessedCount)
}

func (s *AuditWorkflowTestSuite) TestScheduledAuditFlushWorkflow_NoEntries() {
	var a *Activities

	flushResult := &FlushFromRedisResult{
		Batches:    [][]*audit.Entry{},
		EntryCount: 0,
	}

	s.env.OnActivity(a.FlushFromRedisActivity, mock.Anything).Return(flushResult, nil)

	s.env.ExecuteWorkflow(ScheduledAuditFlushWorkflow)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *AuditWorkflowTestSuite) TestScheduledAuditFlushWorkflow_WithEntries() {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	entries := []*audit.Entry{
		{
			ID:             pulid.MustNew("ael_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		{
			ID:             pulid.MustNew("ael_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
	}

	flushResult := &FlushFromRedisResult{
		Batches:    [][]*audit.Entry{entries},
		EntryCount: 2,
	}

	var a *Activities
	s.env.OnActivity(a.FlushFromRedisActivity, mock.Anything).Return(flushResult, nil)

	s.env.RegisterWorkflow(ProcessAuditBatchWorkflow)
	s.env.OnActivity(a.ProcessAuditBatchActivity, mock.Anything, mock.Anything).Return(
		&ProcessAuditBatchResult{
			ProcessedCount: 2,
			FailedCount:    0,
		}, nil)

	s.env.ExecuteWorkflow(ScheduledAuditFlushWorkflow)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *AuditWorkflowTestSuite) TestDLQRetryWorkflow_NoEntries() {
	var a *Activities

	result := &DLQRetryResult{
		RetryCount:     0,
		SuccessCount:   0,
		FailedCount:    0,
		ExhaustedCount: 0,
	}

	s.env.OnActivity(a.RetryDLQEntriesActivity, mock.Anything, 100).Return(result, nil)

	s.env.ExecuteWorkflow(DLQRetryWorkflow)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func TestAuditWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(AuditWorkflowTestSuite))
}
