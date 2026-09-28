package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePermits struct {
	serviceports.PermitService

	guard  *writeGuard
	permit *permit.Permit

	created *permit.Permit
	updated *permit.Permit
}

func newFakePermits() *fakePermits {
	return &fakePermits{
		guard: &writeGuard{},
		permit: &permit.Permit{
			ID:           pulid.MustNew("pmt_"),
			ShipmentID:   pulid.MustNew("shp_"),
			StateID:      pulid.MustNew("us_"),
			PermitNumber: "IN-44120",
			Status:       permit.StatusPending,
			Version:      2,
		},
	}
}

func (f *fakePermits) ListPermits(
	_ context.Context,
	shipmentID pulid.ID,
	_ pagination.TenantInfo,
) ([]*permit.Permit, error) {
	if shipmentID != f.permit.ShipmentID {
		return nil, nil
	}
	copied := *f.permit
	return []*permit.Permit{&copied}, nil
}

func validPermit(entity *permit.Permit) error {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (f *fakePermits) PlanCreatePermit(
	_ context.Context,
	entity *permit.Permit,
) (*permit.Permit, error) {
	if err := validPermit(entity); err != nil {
		return nil, err
	}
	return entity, nil
}

func (f *fakePermits) CreatePermit(
	_ context.Context,
	entity *permit.Permit,
	_ *serviceports.RequestActor,
) (*permit.Permit, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.created = entity
	created := *entity
	created.ID = pulid.MustNew("pmt_")
	return &created, nil
}

func (f *fakePermits) PlanUpdatePermit(
	_ context.Context,
	entity *permit.Permit,
) (*serviceports.RecordChange[permit.Permit], error) {
	if err := validPermit(entity); err != nil {
		return nil, err
	}
	return &serviceports.RecordChange[permit.Permit]{Before: f.permit, After: entity}, nil
}

func (f *fakePermits) UpdatePermit(
	_ context.Context,
	entity *permit.Permit,
	_ *serviceports.RequestActor,
) (*permit.Permit, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = entity
	return entity, nil
}

func TestRecordShipmentPermit_ChecksWhatTheStateIssued(t *testing.T) {
	t.Parallel()

	permits := newFakePermits()
	tool := newRecordShipmentPermitTool(permits)
	params := executeParams(map[string]any{
		paramShipmentID:   permits.permit.ShipmentID.String(),
		paramStateID:      permits.permit.StateID.String(),
		paramPermitNumber: "OH-99812",
		fieldStatus:       "Active",
		paramPermitIssued: "2026-09-29T00:00:00-04:00",
		paramPermitExpiry: "2026-10-04T23:59:00-04:00",
		paramPermitCost:   "85.00",
		fieldNotes:        "Daylight travel only",
	})

	preview := previewWithoutWrites(t, permits.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Would record permit OH-99812 as Active.", preview.Summary)
	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, permits.created)
	assert.Equal(t, params.OrganizationID, permits.created.OrganizationID)
	assert.True(t, permits.created.Cost.Valid)
	require.NotNil(t, permits.created.ExpiresAt)
	assert.Equal(t, permits.permit.ShipmentID.String(), result.IDs[paramShipmentID])
	assert.Equal(t, shipmentRecordEntity, result.Record.EntityType)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourcePermit, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramShipmentID:   permits.permit.ShipmentID.String(),
			paramStateID:      permits.permit.StateID.String(),
			paramPermitNumber: "OH-99812",
			fieldStatus:       "Active",
		})), "an active permit records when it expires")
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramShipmentID:   permits.permit.ShipmentID.String(),
			paramStateID:      permits.permit.StateID.String(),
			paramPermitNumber: "OH-99812",
			paramPermitCost:   "-5",
		})))
}

func TestUpdateShipmentPermit_FindsThePermitOnItsShipment(t *testing.T) {
	t.Parallel()

	permits := newFakePermits()
	tool := newUpdateShipmentPermitTool(permits)
	params := executeParams(map[string]any{
		paramShipmentID: permits.permit.ShipmentID.String(),
		paramPermitID:   permits.permit.ID.String(),
		fieldStatus:     "Void",
	})

	preview := previewWithoutWrites(t, permits.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Would change permit IN-44120.", preview.Summary)
	assert.Equal(t, string(permit.StatusVoid),
		fieldByPath(t, previewChange(t, preview, 0), fieldStatus).After)
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, permits.updated)
	assert.Equal(t, permit.StatusVoid, permits.updated.Status)
	assert.Equal(t, "IN-44120", permits.updated.PermitNumber)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramShipmentID: pulid.MustNew("shp_").String(),
			paramPermitID:   permits.permit.ID.String(),
			fieldStatus:     "Void",
		})), "a permit is changed only on the shipment it belongs to")
}
