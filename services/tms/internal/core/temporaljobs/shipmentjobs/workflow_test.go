package shipmentjobs

import (
	"testing"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type ShipmentWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *ShipmentWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *ShipmentWorkflowTestSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func (s *ShipmentWorkflowTestSuite) TestBulkDuplicateShipmentsWorkflow() {
	sourceID := pulid.MustNew("shp_")
	copyID := pulid.MustNew("shp_")
	payload := &BulkDuplicateShipmentsPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			UserID:         pulid.MustNew("usr_"),
			Timestamp:      timeutils.NowUnix(),
		},
		ShipmentID:    sourceID,
		Count:         1,
		OverrideDates: true,
		RequestedBy:   pulid.MustNew("usr_"),
	}

	expected := &BulkDuplicateShipmentsResult{
		ShipmentIDs:      []pulid.ID{copyID},
		DuplicatedCount:  1,
		CompletedAt:      timeutils.NowUnix(),
		SourceShipmentID: sourceID,
	}

	var a *Activities
	s.env.OnActivity(a.BulkDuplicateShipmentsActivity, mock.Anything, payload).
		Return(expected, nil)

	s.env.ExecuteWorkflow(BulkDuplicateShipmentsWorkflow, payload)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result *BulkDuplicateShipmentsResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(1, result.DuplicatedCount)
	s.Equal(sourceID, result.SourceShipmentID)
}

func (s *ShipmentWorkflowTestSuite) TestAutoDelayShipmentsWorkflow() {
	expected := &AutoDelayShipmentsResult{
		ShipmentIDs:  []pulid.ID{pulid.MustNew("shp_")},
		DelayedCount: 1,
		CompletedAt:  timeutils.NowUnix(),
	}

	var a *Activities
	s.env.OnActivity(a.AutoDelayShipmentsActivity, mock.Anything).
		Return(expected, nil)

	s.env.ExecuteWorkflow(AutoDelayShipmentsWorkflow)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result *AutoDelayShipmentsResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(1, result.DelayedCount)
}

func (s *ShipmentWorkflowTestSuite) TestAutoCancelShipmentsWorkflow() {
	expected := &AutoCancelShipmentsResult{
		ShipmentIDs:   []pulid.ID{pulid.MustNew("shp_")},
		CanceledCount: 1,
		CompletedAt:   timeutils.NowUnix(),
	}

	var a *Activities
	s.env.OnActivity(a.AutoCancelShipmentsActivity, mock.Anything).
		Return(expected, nil)

	s.env.ExecuteWorkflow(AutoCancelShipmentsWorkflow)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result *AutoCancelShipmentsResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(1, result.CanceledCount)
}

func TestShipmentWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(ShipmentWorkflowTestSuite))
}
