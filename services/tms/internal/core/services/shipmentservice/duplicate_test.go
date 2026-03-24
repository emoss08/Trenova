package shipmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/temporaljobs/shipmentjobs"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type fakeShipmentWorkflowRun struct {
	client.WorkflowRun
	workflowID string
	runID      string
}

func (f *fakeShipmentWorkflowRun) GetID() string    { return f.workflowID }
func (f *fakeShipmentWorkflowRun) GetRunID() string { return f.runID }

type fakeShipmentTemporalClient struct {
	client.Client
	executeWorkflowFunc func(
		ctx context.Context,
		options client.StartWorkflowOptions,
		workflow any,
		args ...any,
	) (client.WorkflowRun, error)
}

func (f *fakeShipmentTemporalClient) ExecuteWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow any,
	args ...any,
) (client.WorkflowRun, error) {
	return f.executeWorkflowFunc(ctx, options, workflow, args...)
}

func TestServiceDuplicate_StartsShipmentDuplicateWorkflow(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	audit := mocks.NewMockAuditService(t)
	req := &repositories.BulkDuplicateShipmentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		ShipmentID:    pulid.MustNew("shp_"),
		Count:         3,
		OverrideDates: true,
	}

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		validator:    NewTestValidator(t),
		auditService: audit,
		temporalClient: &fakeShipmentTemporalClient{
			executeWorkflowFunc: func(
				_ context.Context,
				options client.StartWorkflowOptions,
				workflow any,
				args ...any,
			) (client.WorkflowRun, error) {
				assert.Equal(t, temporaltype.TaskQueueSystem.String(), options.TaskQueue)
				assert.Equal(t, shipmentjobs.BulkDuplicateShipmentsWorkflowName, workflow)

				require.Len(t, args, 1)
				payload, ok := args[0].(*shipmentjobs.BulkDuplicateShipmentsPayload)
				require.True(t, ok)
				assert.Equal(t, req.ShipmentID, payload.ShipmentID)
				assert.Equal(t, req.Count, payload.Count)
				assert.True(t, payload.OverrideDates)
				assert.Equal(t, req.TenantInfo.UserID, payload.RequestedBy)

				return &fakeShipmentWorkflowRun{
					workflowID: options.ID,
					runID:      "run-1",
				}, nil
			},
		},
		coordinator: newStateCoordinator(),
	}

	resp, err := svc.Duplicate(t.Context(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "run-1", resp.RunID)
	assert.Equal(t, temporaltype.TaskQueueSystem.String(), resp.TaskQueue)
}

func TestServiceDuplicate_RejectsInvalidRequest(t *testing.T) {
	t.Parallel()

	svc := &service{
		l:            zap.NewNop(),
		repo:         mocks.NewMockShipmentRepository(t),
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		coordinator:  newStateCoordinator(),
	}

	resp, err := svc.Duplicate(t.Context(), &repositories.BulkDuplicateShipmentRequest{})
	require.Nil(t, resp)
	require.Error(t, err)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assertErrorField(t, multiErr, "shipmentId")
}

func TestServiceDuplicate_RejectsMissingTemporalClient(t *testing.T) {
	t.Parallel()

	svc := &service{
		l:            zap.NewNop(),
		repo:         mocks.NewMockShipmentRepository(t),
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		coordinator:  newStateCoordinator(),
	}

	resp, err := svc.Duplicate(t.Context(), &repositories.BulkDuplicateShipmentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		ShipmentID: pulid.MustNew("shp_"),
		Count:      1,
	})

	require.Nil(t, resp)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}
