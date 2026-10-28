package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingHolds struct {
	shipmentholdports.ShipmentHoldService

	reason  *holdreason.HoldReason
	current *shipment.ShipmentHold
	saved   *shipment.ShipmentHold
	guard   agenttoolstest.WriteGuard
}

func (f *savingHolds) plan(
	req *shipmentholdports.CreateShipmentHoldRequest,
) *shipment.ShipmentHold {
	return &shipment.ShipmentHold{
		ShipmentID:     req.ShipmentID,
		HoldReasonID:   &f.reason.ID,
		Type:           f.reason.Type,
		Severity:       f.reason.DefaultSeverity,
		ReasonCode:     f.reason.Code,
		Notes:          req.Notes,
		BlocksDispatch: f.reason.DefaultBlocksDispatch,
		BlocksBilling:  f.reason.DefaultBlocksBilling,
		StartedAt:      timeutils.NowUnix(),
	}
}

func (f *savingHolds) PreviewCreate(
	_ context.Context,
	req *shipmentholdports.CreateShipmentHoldRequest,
	_ *permissionports.RequestActor,
) (*shipment.ShipmentHold, error) {
	if !f.reason.Active {
		return nil, errortypes.NewValidationError(
			"holdReasonId", errortypes.ErrInvalid, "Hold reason must be active",
		)
	}

	return f.plan(req), nil
}

func (f *savingHolds) Create(
	_ context.Context,
	req *shipmentholdports.CreateShipmentHoldRequest,
	_ *permissionports.RequestActor,
) (*shipment.ShipmentHold, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.saved = f.plan(req)
	f.saved.ID = pulid.MustNew("shld_")

	return f.saved, nil
}

func (f *savingHolds) GetByID(
	context.Context,
	*shipmentholdports.GetShipmentHoldByIDRequest,
) (*shipment.ShipmentHold, error) {
	copied := *f.current

	return &copied, nil
}

func (f *savingHolds) release(actor *permissionports.RequestActor) (*shipment.ShipmentHold, error) {
	if !f.current.IsActive() {
		return nil, errortypes.NewBusinessError("Shipment hold is already released")
	}
	released := *f.current
	at := timeutils.NowUnix()
	released.ReleasedAt = &at
	released.ReleasedByID = &actor.UserID

	return &released, nil
}

func (f *savingHolds) PreviewRelease(
	_ context.Context,
	_ *shipmentholdports.ReleaseShipmentHoldRequest,
	actor *permissionports.RequestActor,
) (*shipment.ShipmentHold, error) {
	return f.release(actor)
}

func (f *savingHolds) Release(
	_ context.Context,
	_ *shipmentholdports.ReleaseShipmentHoldRequest,
	actor *permissionports.RequestActor,
) (*shipment.ShipmentHold, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	released, err := f.release(actor)
	if err != nil {
		return nil, err
	}
	f.saved = released

	return released, nil
}

func documentsHoldReason(active bool) *holdreason.HoldReason {
	return &holdreason.HoldReason{
		ID:                    pulid.MustNew("hr_"),
		Code:                  "MISSING_DOCS",
		Type:                  holdreason.HoldTypeOperational,
		DefaultSeverity:       holdreason.HoldSeverityBlocking,
		DefaultBlocksDispatch: true,
		DefaultBlocksBilling:  true,
		Active:                active,
	}
}

func TestPlaceShipmentHold_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	holds := &savingHolds{reason: documentsHoldReason(true)}
	tool := shipmenttools.NewPlaceShipmentHoldTool(holds).(*shipmenttools.PlaceShipmentHoldTool)
	shipmentID := pulid.MustNew("shp_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId":   shipmentID.String(),
		"holdReasonId": holds.reason.ID.String(),
		"notes":        "Waiting on the signed BOL.",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&holds.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	assert.Equal(t, permission.ResourceShipmentHold, change.Resource)
	assert.Equal(t, true, agenttoolstest.FieldByPath(t, change, "blocksDispatch").After)
	shipmentField := agenttoolstest.FieldByPath(t, change, "shipmentId")
	require.NotNil(t, shipmentField.AfterRef)
	assert.Equal(t, shipmentID, shipmentField.AfterRef.ID)
	assert.Contains(t, preview.Summary, "blocks dispatch, billing")

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireCreateParity(t, change, holds.saved, shipmenttools.PlacedHoldOptions()...)
}

func TestPlaceShipmentHold_PreviewWarnsOnAnInactiveReason(t *testing.T) {
	t.Parallel()

	holds := &savingHolds{reason: documentsHoldReason(false)}
	tool := shipmenttools.NewPlaceShipmentHoldTool(holds).(*shipmenttools.PlaceShipmentHoldTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&holds.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"shipmentId":   pulid.MustNew("shp_").String(),
				"holdReasonId": holds.reason.ID.String(),
			}))
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestReleaseShipmentHold_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	reason := documentsHoldReason(true)
	current := &shipment.ShipmentHold{
		ID:             pulid.MustNew("shld_"),
		ShipmentID:     pulid.MustNew("shp_"),
		Type:           reason.Type,
		ReasonCode:     reason.Code,
		BlocksDelivery: true,
		Version:        4,
	}
	holds := &savingHolds{reason: reason, current: current}
	tool := shipmenttools.NewReleaseShipmentHoldTool(holds).(*shipmenttools.ReleaseShipmentHoldTool)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": current.ShipmentID.String(),
		"holdId":     current.ID.String(),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&holds.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationArchive, change.Operation)
	assert.Equal(t, current.ID, change.EntityID)
	assert.True(t, agenttoolstest.FieldByPath(t, change, "releasedAt").Volatile)
	assert.Contains(t, preview.Summary, "blocks delivery")

	require.NoError(t, tool.Execute(t.Context(), params))
	want, err := toolpreview.Changed(toolpreview.Record{Resource: change.Resource},
		current, holds.saved, toolpreview.Only(shipmenttools.ReleasedHoldFields...))
	require.NoError(t, err)
	assert.Equal(t, []string{"releasedAt"}, fieldPaths(want.Fields))
	assert.Equal(t, fieldPaths(want.Fields), fieldPaths(change.Fields))
}

func TestReleaseShipmentHold_PreviewWarnsOnAHoldAlreadyReleased(t *testing.T) {
	t.Parallel()

	released := int64(100)
	current := &shipment.ShipmentHold{ID: pulid.MustNew("shld_"), ReleasedAt: &released}
	holds := &savingHolds{reason: documentsHoldReason(true), current: current}
	tool := shipmenttools.NewReleaseShipmentHoldTool(holds).(*shipmenttools.ReleaseShipmentHoldTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&holds.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"shipmentId": pulid.MustNew("shp_").String(),
				"holdId":     current.ID.String(),
			}))
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func fieldPaths(fields []agent.PreviewFieldChange) []string {
	paths := make([]string, 0, len(fields))
	for i := range fields {
		paths = append(paths, fields[i].Path)
	}

	return paths
}
