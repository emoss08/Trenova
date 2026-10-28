package servicefailureservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/ediportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/realtimeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/servicefailureportsmock"
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
	repo := servicefailureportsmock.NewMockServiceFailureRepository(t)
	repo.EXPECT().GetByShipment(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, *servicefailureports.GetServiceFailureByShipmentRequest) (*servicefailure.ServiceFailure, error) {
			copied := *stored
			return &copied, nil
		})
	ediSvc := ediportsmock.NewMockEDIService(t)
	ediSvc.EXPECT().PreviewServiceFailure214ForLifecycle(mock.Anything, mock.Anything).
		Return(&servicefailureports.ServiceFailure214LifecycleResult{
			Trigger:       servicefailureports.ServiceFailureEDITriggerResolved,
			Action:        ediports.ServiceFailureEDIActionSkipped,
			SkippedReason: "ready_for_generation",
			EDIPartnerID:  pulid.MustNew("edip_"),
		}, nil)
	svc := &service{l: zap.NewNop(), repo: repo, ediService: ediSvc}
	req := func() *servicefailureports.ServiceFailureLifecycleRequest {
		return &servicefailureports.ServiceFailureLifecycleRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
			ID:         stored.ID,
			ShipmentID: stored.ShipmentID,
			Version:    3,
			Notes:      "Receiver held the truck at the gate",
		}
	}
	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
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
		Return(&servicefailureports.ServiceFailure214LifecycleResult{
			Action: ediports.ServiceFailureEDIActionSkipped,
		}, nil).
		Once()
	audit := auditportsmock.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	realtime := realtimeportsmock.NewMockRealtimeService(t)
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
