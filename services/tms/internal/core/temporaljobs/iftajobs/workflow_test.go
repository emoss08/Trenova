package iftajobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type BackfillWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *BackfillWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *BackfillWorkflowTestSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func (s *BackfillWorkflowTestSuite) runBackfill(
	unattributed []pulid.ID,
	maxMoves int,
) (*BackfillResult, [][]pulid.ID) {
	var a *Activities
	batches := make([][]pulid.ID, 0, 4)
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	s.env.OnActivity(a.ListMovesMissingBreakdownActivity, mock.Anything, mock.Anything).
		Return(func(
			_ context.Context,
			input ListMovesMissingBreakdownInput,
		) (*ListMovesMissingBreakdownResult, error) {
			s.Equal(tenantInfo, input.TenantInfo)
			s.Equal(ListMovesPageSize, input.Limit)
			return &ListMovesMissingBreakdownResult{
				MoveIDs:    unattributed,
				TotalMoves: len(unattributed),
				TotalMiles: decimal.NewFromInt(int64(len(unattributed)) * 10),
			}, nil
		})
	if len(unattributed) > 0 {
		s.env.OnActivity(a.AttributeMovesActivity, mock.Anything, mock.Anything).
			Return(func(_ context.Context, input AttributeMovesInput) (*AttributeMovesResult, error) {
				batches = append(batches, input.MoveIDs)
				return &AttributeMovesResult{
					Processed: len(input.MoveIDs),
					Miles:     decimal.NewFromInt(int64(len(input.MoveIDs)) * 10),
				}, nil
			})
	}

	s.env.ExecuteWorkflow(BackfillJurisdictionMilesWorkflow, BackfillInput{
		TenantInfo: tenantInfo,
		Start:      100,
		End:        200,
		MaxMoves:   maxMoves,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result *BackfillResult
	s.NoError(s.env.GetWorkflowResult(&result))
	return result, batches
}

func (s *BackfillWorkflowTestSuite) TestBatchesOfTwenty() {
	result, batches := s.runBackfill(moveIDs(45), 0)

	s.Require().Len(batches, 3)
	s.Len(batches[0], 20)
	s.Len(batches[1], 20)
	s.Len(batches[2], 5)
	s.Equal(45, result.Processed)
	s.Equal(0, result.Failed)
	s.True(result.Miles.Equal(decimal.NewFromInt(450)), result.Miles.String())
}

func (s *BackfillWorkflowTestSuite) TestStopsAtMaxMoves() {
	result, batches := s.runBackfill(moveIDs(45), 30)

	s.Require().Len(batches, 2)
	s.Len(batches[0], 20)
	s.Len(batches[1], 10)
	s.Equal(30, result.Processed)
}

func (s *BackfillWorkflowTestSuite) TestNothingToDo() {
	result, batches := s.runBackfill(nil, 0)

	s.Empty(batches)
	s.Equal(0, result.Processed)
	s.Equal(0, result.Failed)
}

func TestBackfillWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(BackfillWorkflowTestSuite))
}
