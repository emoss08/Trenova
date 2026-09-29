package shipmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/shipmentjobs"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	startWorkflowFunc func(
		ctx context.Context,
		options client.StartWorkflowOptions,
		workflow any,
		args ...any,
	) (client.WorkflowRun, error)
}

func (f *fakeShipmentTemporalClient) StartWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow any,
	args ...any,
) (client.WorkflowRun, error) {
	return f.startWorkflowFunc(ctx, options, workflow, args...)
}

func (*fakeShipmentTemporalClient) Enabled() bool { return true }

func (*fakeShipmentTemporalClient) CancelWorkflow(context.Context, string, string) error {
	return nil
}

func (*fakeShipmentTemporalClient) SignalWorkflow(
	context.Context,
	string, string, string,
	any,
) error {
	return nil
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

	repo.EXPECT().PlanDuplicate(mock.Anything, req).Return(&repositories.ShipmentDuplicatePlan{
		Source: &shipment.Shipment{
			ID:                req.ShipmentID,
			FormulaTemplateID: pulid.MustNew("fmt_"),
		},
		Copies: []*shipment.Shipment{{FormulaTemplateID: pulid.MustNew("fmt_")}},
	}, nil)

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		validator:    NewTestValidator(t),
		auditService: audit,
		workflowStarter: &fakeShipmentTemporalClient{
			startWorkflowFunc: func(
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
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}

	resp, err := svc.Duplicate(t.Context(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "run-1", resp.RunID)
	assert.Equal(t, temporaltype.TaskQueueSystem.String(), resp.TaskQueue)
}

func TestServiceDuplicate_RefusesToCopyAnUnratedShipment(t *testing.T) {
	t.Parallel()

	req := &repositories.BulkDuplicateShipmentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		ShipmentID: pulid.MustNew("shp_"),
		Count:      1,
	}
	source := &shipment.Shipment{
		ID:             req.ShipmentID,
		OrganizationID: req.TenantInfo.OrgID,
		ProNumber:      "SEED-DET-009",
		RatingDetail:   &shipment.RatingDetail{Source: string(ratequote.OutcomeNoRateFound)},
	}
	plan := &repositories.ShipmentDuplicatePlan{
		Source: source,
		Copies: []*shipment.Shipment{{}},
	}

	for _, tt := range []struct {
		name        string
		disposition tenant.UnratedShipmentDisposition
		refused     bool
	}{
		{name: "the default disposition", refused: true},
		{name: "fall back to a template", disposition: tenant.UnratedShipmentDispositionFallbackFormulaTemplate, refused: true},
		{name: "zero and flag", disposition: tenant.UnratedShipmentDispositionZeroAndFlag},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := mocks.NewMockShipmentRepository(t)
			repo.EXPECT().PlanDuplicate(mock.Anything, req).Return(plan, nil)
			started := false
			svc := &service{
				l:            zap.NewNop(),
				repo:         repo,
				billingRepo:  billingControlWith(t, tt.disposition),
				validator:    NewTestValidator(t),
				auditService: mocks.NewMockAuditService(t),
				workflowStarter: &fakeShipmentTemporalClient{
					startWorkflowFunc: func(
						_ context.Context,
						options client.StartWorkflowOptions,
						_ any,
						_ ...any,
					) (client.WorkflowRun, error) {
						started = true
						return &fakeShipmentWorkflowRun{workflowID: options.ID, runID: "run-1"}, nil
					},
				},
				eventService: noopShipmentEventService{},
				coordinator:  newStateCoordinator(),
			}

			_, err := svc.Duplicate(t.Context(), req)
			preview, previewErr := svc.PreviewDuplicate(t.Context(), req)

			if !tt.refused {
				require.NoError(t, err)
				require.NoError(t, previewErr)
				assert.NotNil(t, preview)
				assert.True(t, started)
				return
			}
			require.Error(t, err)
			require.Error(t, previewErr)
			assert.False(t, started, "nothing is copied once the copy is refused")
			var validationErr *errortypes.Error
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, "shipmentId", validationErr.Field)
			assert.Contains(t, validationErr.Error(), "SEED-DET-009")
		})
	}
}

func TestServiceDuplicate_RejectsInvalidRequest(t *testing.T) {
	t.Parallel()

	svc := &service{
		l:            zap.NewNop(),
		repo:         mocks.NewMockShipmentRepository(t),
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		eventService: noopShipmentEventService{},
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
		l:               zap.NewNop(),
		repo:            mocks.NewMockShipmentRepository(t),
		validator:       NewTestValidator(t),
		auditService:    mocks.NewMockAuditService(t),
		eventService:    noopShipmentEventService{},
		coordinator:     newStateCoordinator(),
		workflowStarter: disabledWorkflowStarter{},
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

type disabledWorkflowStarter struct{}

func (disabledWorkflowStarter) StartWorkflow(
	context.Context,
	client.StartWorkflowOptions,
	any,
	...any,
) (client.WorkflowRun, error) {
	return nil, services.ErrWorkflowStarterDisabled
}

func (disabledWorkflowStarter) Enabled() bool { return false }

func (disabledWorkflowStarter) CancelWorkflow(context.Context, string, string) error {
	return services.ErrWorkflowStarterDisabled
}

func (disabledWorkflowStarter) SignalWorkflow(
	context.Context,
	string, string, string,
	any,
) error {
	return services.ErrWorkflowStarterDisabled
}
