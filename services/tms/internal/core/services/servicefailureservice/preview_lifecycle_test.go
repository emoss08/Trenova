package servicefailureservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// The repository has no Update expectation while the preview runs, and the
// EDI service no Generate one, so a write or a 214 sent during it fails the
// test.
func TestPreviewResolve_IsTheResolutionResolveSaves(t *testing.T) {
	t.Parallel()

	orgID, buID, userID := pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")
	reasonID := pulid.MustNew("sfrc_")
	stored := &servicefailure.ServiceFailure{
		ID:                 pulid.MustNew("sf_"),
		ShipmentID:         pulid.MustNew("sp_"),
		ShipmentMoveID:     pulid.MustNew("sm_"),
		StopID:             pulid.MustNew("stp_"),
		ReasonCodeID:       &reasonID,
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		Type:               servicefailure.TypeLateDelivery,
		Source:             servicefailure.SourceDetected,
		Status:             servicefailure.StatusReviewed,
		StopType:           shipment.StopTypeDelivery,
		ScheduledCutoff:    1_000,
		ActualArrival:      1_301,
		GracePeriodMinutes: 5,
		LateMinutes:        1,
	}
	repo := mocks.NewMockServiceFailureRepository(t)
	repo.EXPECT().GetByShipment(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, *repositories.GetServiceFailureByShipmentRequest) (*servicefailure.ServiceFailure, error) {
			copied := *stored
			return &copied, nil
		})
	ediSvc := mocks.NewMockEDIService(t)
	ediSvc.EXPECT().PreviewServiceFailure214ForLifecycle(mock.Anything, mock.Anything).
		Return(&serviceports.ServiceFailure214LifecycleResult{
			Trigger:       serviceports.ServiceFailureEDITriggerResolved,
			Action:        serviceports.ServiceFailureEDIActionSkipped,
			SkippedReason: "ready_for_generation",
			EDIPartnerID:  pulid.MustNew("edip_"),
		}, nil)
	svc := &service{l: zap.NewNop(), repo: repo, ediService: ediSvc}
	req := func() *serviceports.ServiceFailureLifecycleRequest {
		return &serviceports.ServiceFailureLifecycleRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
			ID:         stored.ID,
			ShipmentID: stored.ShipmentID,
			Version:    3,
			Notes:      "Receiver held the truck at the gate",
		}
	}
	actor := &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}

	preview, err := svc.PreviewResolve(t.Context(), req(), actor)
	require.NoError(t, err)
	require.NotNil(t, preview.EDI)
	assert.Equal(t, "ready_for_generation", preview.EDI.SkippedReason)

	var saved *servicefailure.ServiceFailure
	repo.EXPECT().Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *servicefailure.ServiceFailure) (*servicefailure.ServiceFailure, error) {
			saved = entity
			return entity, nil
		}).
		Once()
	ediSvc.EXPECT().GenerateServiceFailure214ForLifecycle(mock.Anything, mock.Anything).
		Return(&serviceports.ServiceFailure214LifecycleResult{
			Action: serviceports.ServiceFailureEDIActionSkipped,
		}, nil).
		Once()
	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Maybe()
	svc.auditService = audit
	svc.realtime = realtime

	_, err = svc.Resolve(t.Context(), req(), actor)
	require.NoError(t, err)
	require.NotNil(t, saved)

	assert.Equal(t, servicefailure.StatusReviewed, preview.Before.Status)
	assert.Equal(t, saved.Status, preview.After.Status)
	assert.Equal(t, saved.InternalNotes, preview.After.InternalNotes)
	assert.Equal(t, saved.ResolvedByID, preview.After.ResolvedByID)
	assert.Equal(t, saved.ReasonCodeID, preview.After.ReasonCodeID)
	assert.Equal(t, saved.Version, preview.After.Version)
}
