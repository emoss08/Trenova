package carrierassignmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingAuditService struct {
	portservices.AuditService

	entries []*portservices.LogActionParams
}

func (r *recordingAuditService) LogAction(
	params *portservices.LogActionParams,
	_ ...portservices.LogOption,
) error {
	r.entries = append(r.entries, params)
	return nil
}

func TestConfirmAssignmentLogsMoveAudit(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	entity := &shipment.CarrierAssignment{
		ID:             pulid.MustNew("cas_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ShipmentMoveID: pulid.MustNew("smv_"),
		CarrierID:      pulid.MustNew("car_"),
		Status:         shipment.CarrierAssignmentStatusPending,
	}

	repo := mocks.NewMockCarrierAssignmentRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(entity, nil).Once()
	repo.On("Update", mock.Anything, entity).Return(entity, nil).Once()

	audit := &recordingAuditService{}
	svc := &Service{l: zap.NewNop(), repo: repo, auditService: audit}

	require.NoError(t, svc.ConfirmAssignment(t.Context(), tenantInfo, entity.ID))
	assert.Equal(t, shipment.CarrierAssignmentStatusConfirmed, entity.Status)

	require.Len(t, audit.entries, 1)
	entry := audit.entries[0]
	assert.Equal(t, permission.ResourceShipmentMove, entry.Resource)
	assert.Equal(t, permission.OpUpdate, entry.Operation)
	assert.Equal(t, entity.ShipmentMoveID.String(), entry.ResourceID)
	assert.Equal(t, tenantInfo.UserID, entry.UserID)
	assert.Equal(t, tenantInfo.OrgID, entry.OrganizationID)
	assert.Equal(t, tenantInfo.BuID, entry.BusinessUnitID)
	assert.NotEmpty(t, entry.PreviousState)
	assert.NotEmpty(t, entry.CurrentState)
	assert.Empty(t, entry.PrincipalType)
}

func TestConfirmAssignmentAlreadyConfirmedLogsNoAudit(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	entity := &shipment.CarrierAssignment{
		ID:     pulid.MustNew("cas_"),
		Status: shipment.CarrierAssignmentStatusConfirmed,
	}

	repo := mocks.NewMockCarrierAssignmentRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(entity, nil).Once()

	audit := &recordingAuditService{}
	svc := &Service{l: zap.NewNop(), repo: repo, auditService: audit}

	require.NoError(t, svc.ConfirmAssignment(t.Context(), tenantInfo, entity.ID))
	assert.Empty(t, audit.entries)
}
