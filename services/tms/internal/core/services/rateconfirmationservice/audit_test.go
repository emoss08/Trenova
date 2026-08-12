package rateconfirmationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingAuditService struct {
	services.AuditService

	entries []*services.LogActionParams
}

func (r *recordingAuditService) LogAction(
	params *services.LogActionParams,
	_ ...services.LogOption,
) error {
	r.entries = append(r.entries, params)
	return nil
}

func (r *recordingAuditService) only(t *testing.T) *services.LogActionParams {
	t.Helper()
	require.Len(t, r.entries, 1)
	return r.entries[0]
}

func TestConfirmByTokenLogsSystemPrincipalAudit(t *testing.T) {
	deps := setupPublicTest(t)
	audit := &recordingAuditService{}
	deps.svc.auditService = audit

	entity := signableRateCon(deps)
	raw, _ := mintTestToken(deps, entity)

	assignment := &shipment.CarrierAssignment{
		ID:        entity.CarrierAssignmentID,
		CarrierID: entity.CarrierID,
		Status:    shipment.CarrierAssignmentStatusPending,
	}
	deps.carrierAssignmentRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(assignment, nil)
	deps.carrierAssignmentRepo.On("Update", mock.Anything, assignment).
		Return(assignment, nil)

	err := deps.svc.ConfirmByToken(t.Context(), raw, "J. Doe", "Operations Manager")

	require.NoError(t, err)

	entry := audit.only(t)
	assert.Equal(t, permission.ResourceRateConfirmation, entry.Resource)
	assert.Equal(t, permission.OpApprove, entry.Operation)
	assert.Equal(t, entity.ID.String(), entry.ResourceID)
	assert.True(t, entry.UserID.IsNil(), "a public signer is never a user principal")
	assert.Equal(t, services.PrincipalTypeSystem, entry.PrincipalType)
	assert.Equal(t, services.SystemPrincipalID, entry.PrincipalID)
	assert.Equal(t, entity.OrganizationID, entry.OrganizationID)
	assert.Equal(t, entity.BusinessUnitID, entry.BusinessUnitID)
	assert.NotEmpty(t, entry.PreviousState)
	assert.NotEmpty(t, entry.CurrentState)
}

func TestConfirmByTokenAlreadyConfirmedLogsNoAudit(t *testing.T) {
	deps := setupPublicTest(t)
	audit := &recordingAuditService{}
	deps.svc.auditService = audit

	entity := signableRateCon(deps)
	entity.Status = rateconfirmation.StatusConfirmed
	raw, _ := mintTestToken(deps, entity)

	err := deps.svc.ConfirmByToken(t.Context(), raw, "J. Doe", "")

	require.NoError(t, err)
	assert.Empty(t, audit.entries)
}
