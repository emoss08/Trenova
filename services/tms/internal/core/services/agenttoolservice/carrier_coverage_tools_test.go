package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/carrierassignmentservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCarrierCoverage builds its plans the way the service does, from the
// request and a shipment it holds, and records what a write was asked.
type fakeCarrierCoverage struct {
	guard    writeGuard
	shipment *shipment.Shipment
	carrier  *carrier.Carrier
	existing *shipment.CarrierAssignment
	standing *rateconfirmation.RateConfirmation
	refusal  error

	assigned *repositories.AssignMoveToCarrierRequest
	canceled *repositories.CancelCarrierAssignmentRequest
}

func newFakeCarrierCoverage() *fakeCarrierCoverage {
	moveID := pulid.MustNew("smv_")

	return &fakeCarrierCoverage{
		shipment: &shipment.Shipment{
			ID:        pulid.MustNew("shp_"),
			ProNumber: "S-3001",
			Status:    shipment.StatusNew,
			Version:   4,
			Moves: []*shipment.ShipmentMove{{
				ID:           moveID,
				Status:       shipment.MoveStatusNew,
				CoverageType: shipment.MoveCoverageTypeUnassigned,
				Version:      2,
			}},
		},
		carrier: &carrier.Carrier{ID: pulid.MustNew("car_"), Name: "Ridgeline Freight"},
	}
}

func (f *fakeCarrierCoverage) moveID() pulid.ID { return f.shipment.Moves[0].ID }

func (f *fakeCarrierCoverage) retire(plan *carrierassignmentservice.CoveragePlan, reason string) {
	if f.existing == nil {
		return
	}
	canceled := *f.existing
	canceled.Cancel(1790000000, reason)
	plan.CanceledBefore = f.existing
	plan.CanceledAfter = &canceled
	if f.standing != nil {
		voided := *f.standing
		voided.Void(1790000000, "Carrier assignment canceled")
		plan.VoidedBefore = f.standing
		plan.VoidedAfter = &voided
	}
}

func (f *fakeCarrierCoverage) PreviewAssignToMove(
	_ context.Context,
	req *repositories.AssignMoveToCarrierRequest,
) (*carrierassignmentservice.CoveragePlan, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	created := &shipment.CarrierAssignment{
		ShipmentMoveID: req.ShipmentMoveID,
		CarrierID:      req.CarrierID,
		Status:         shipment.CarrierAssignmentStatusPending,
		RateMethod:     req.RateMethod,
		BaseRate:       req.BaseRate,
		FuelSurcharge:  req.FuelSurcharge,
		CurrencyCode:   "USD",
	}
	for _, acc := range req.Accessorials {
		created.Accessorials = append(created.Accessorials, &shipment.CarrierAssignmentAccessorial{
			Description: acc.Description,
			Amount:      acc.Amount,
		})
	}
	created.SyncTotals(nil)

	after := shipment.CloneForUpdate(f.shipment)
	after.Status = shipment.StatusAssigned
	after.Moves[0].Status = shipment.MoveStatusAssigned
	after.Moves[0].CoverageType = shipment.MoveCoverageTypeCarrier
	plan := &carrierassignmentservice.CoveragePlan{
		Created:        created,
		Carrier:        f.carrier,
		ShipmentBefore: f.shipment,
		ShipmentAfter:  after,
	}
	if req.OverrideInsuranceWarning {
		plan.Warnings = []string{"Cargo liability expires in 10 days"}
	}
	f.retire(plan, "Replaced by a new carrier assignment")

	return plan, nil
}

func (f *fakeCarrierCoverage) AssignToMove(
	_ context.Context,
	req *repositories.AssignMoveToCarrierRequest,
) (*shipment.CarrierAssignment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.assigned = req

	return &shipment.CarrierAssignment{ID: pulid.MustNew("casn_")}, nil
}

func (f *fakeCarrierCoverage) PreviewCancel(
	_ context.Context,
	req *repositories.CancelCarrierAssignmentRequest,
) (*carrierassignmentservice.CoveragePlan, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	if f.existing == nil {
		return nil, errortypes.NewNotFoundError(
			"Carrier assignment not found within your organization",
		)
	}
	after := shipment.CloneForUpdate(f.shipment)
	after.Status = shipment.StatusNew
	after.Moves[0].Status = shipment.MoveStatusNew
	after.Moves[0].CoverageType = shipment.MoveCoverageTypeUnassigned
	plan := &carrierassignmentservice.CoveragePlan{
		Carrier:        f.existing.Carrier,
		ShipmentBefore: f.shipment,
		ShipmentAfter:  after,
	}
	f.retire(plan, req.Reason)

	return plan, nil
}

func (f *fakeCarrierCoverage) Cancel(
	_ context.Context,
	req *repositories.CancelCarrierAssignmentRequest,
) error {
	if err := f.guard.write(); err != nil {
		return err
	}
	f.canceled = req

	return nil
}

func (f *fakeCarrierCoverage) coverWith(status shipment.CarrierAssignmentStatus) {
	f.shipment.Status = shipment.StatusAssigned
	f.shipment.Moves[0].Status = shipment.MoveStatusAssigned
	f.shipment.Moves[0].CoverageType = shipment.MoveCoverageTypeCarrier
	f.existing = &shipment.CarrierAssignment{
		ID:        pulid.MustNew("casn_"),
		CarrierID: f.carrier.ID,
		Carrier:   f.carrier,
		Status:    status,
	}
	f.standing = &rateconfirmation.RateConfirmation{
		ID:                  pulid.MustNew("rc_"),
		CarrierAssignmentID: f.existing.ID,
		Revision:            2,
		Status:              rateconfirmation.StatusSent,
		Version:             5,
	}
}

func (f *fakeCarrierCoverage) assignParams(extra map[string]any) map[string]any {
	params := map[string]any{
		previewFieldShipmentMoveID: f.moveID().String(),
		fieldCarrierID:             f.carrier.ID.String(),
		"rateMethod":               "Flat",
		"baseRate":                 "1800.00",
		"fuelSurcharge":            "120",
	}
	for key, value := range extra {
		params[key] = value
	}

	return params
}

func TestAssignMoveToCarrier_SendsTheConsolesRequest(t *testing.T) {
	t.Parallel()

	carriers := newFakeCarrierCoverage()
	chargeID := pulid.MustNew("acc_")
	params := executeParams(carriers.assignParams(map[string]any{
		"accessorials": []any{
			map[string]any{
				"description":         "Detention at shipper",
				"amount":              "75",
				"accessorialChargeId": chargeID.String(),
			},
		},
		"proNumber":          " RF-88 ",
		"externalDriverName": "Sam Ortiz",
	}))

	require.NoError(t, newAssignMoveToCarrierTool(carriers).Execute(t.Context(), params))

	req := carriers.assigned
	require.NotNil(t, req)
	assert.Equal(t, carriers.moveID(), req.ShipmentMoveID)
	assert.Equal(t, carriers.carrier.ID, req.CarrierID)
	assert.Equal(t, shipment.CarrierRateMethodFlat, req.RateMethod)
	assert.True(t, decimal.NewFromInt(1800).Equal(req.BaseRate))
	assert.True(t, decimal.NewFromInt(120).Equal(req.FuelSurcharge))
	require.Len(t, req.Accessorials, 1)
	assert.Equal(t, "Detention at shipper", req.Accessorials[0].Description)
	assert.Equal(t, chargeID, *req.Accessorials[0].AccessorialChargeID)
	assert.Equal(t, "RF-88", req.ProNumber)
	assert.Equal(t, "Sam Ortiz", req.ExternalDriverName)
	assert.False(t, req.Replace)
	assert.False(t, req.OverrideInsuranceWarning)
	assert.False(t, req.AutoRate, "the tool never leaves the rate to the contract")
	assert.Equal(t, params.OrganizationID, req.TenantInfo.OrgID)
	assert.Equal(t, params.Actor.UserID, req.TenantInfo.UserID)
}

// A carrier covered for nothing would be settled for nothing, and a rate the
// model could not parse is not a rate: both are refused before anything is
// proposed.
func TestAssignMoveToCarrier_RefusesARateThatIsNotARate(t *testing.T) {
	t.Parallel()

	carriers := newFakeCarrierCoverage()
	tool := newAssignMoveToCarrierTool(carriers).(*assignMoveToCarrierTool)
	for name, extra := range map[string]map[string]any{
		"zero":            {"baseRate": "0"},
		"negative":        {"baseRate": "-10"},
		"words":           {"baseRate": "about eighteen hundred"},
		"missing":         {"baseRate": ""},
		"method":          {"rateMethod": "PerLoad"},
		"negative fuel":   {"fuelSurcharge": "-1"},
		"bad accessorial": {"accessorials": []any{map[string]any{"description": "x", "amount": "-5"}}},
	} {
		err := tool.Validate(t.Context(), executeParams(carriers.assignParams(extra)))
		require.Error(t, err, name)
	}
	assert.Nil(t, carriers.assigned)
}

func TestAssignMoveToCarrier_PolicyCommitsMoneyAndHoldsAReplacement(t *testing.T) {
	t.Parallel()

	tool := newAssignMoveToCarrierTool(newFakeCarrierCoverage())
	policy := tool.Policy()

	assert.Equal(t, permission.ResourceShipmentMove, policy.Resource)
	assert.Equal(t, permission.OpAssign, policy.Operation)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)

	limit := func(extra map[string]any) agent.AutonomyTier {
		return policy.Condition.Limit(t.Context(), executeParams(extra))
	}
	assert.Equal(t, agent.TierActWithApproval, limit(map[string]any{}))
	assert.Equal(t, agent.TierPropose, limit(map[string]any{"replace": true}))
	assert.Equal(t, agent.TierPropose, limit(map[string]any{"overrideInsuranceWarning": true}))
}

func TestAssignMoveToCarrier_PreviewShowsTheAssignmentThePayAndTheCover(t *testing.T) {
	t.Parallel()

	carriers := newFakeCarrierCoverage()
	tool := newAssignMoveToCarrierTool(carriers).(*assignMoveToCarrierTool)

	preview := previewWithoutWrites(t, &carriers.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(carriers.assignParams(nil)))
	})

	created := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)
	assert.Equal(t, "Carrier on S-3001: Ridgeline Freight", created.Label)
	assert.Equal(t, "1800", fieldByPath(t, created, "baseRate").After)
	require.NotNil(t, created.Money)
	assert.True(t, decimal.NewFromInt(1920).Equal(created.Money.TotalAfter.Decimal))
	move := previewChange(t, preview, 1)
	assert.Equal(t, carriers.moveID(), move.EntityID)
	assert.Equal(t, "carrier", fieldByPath(t, move, "coverageType").After)
	assert.Contains(t, preview.Summary, "Ridgeline Freight")
	assert.Contains(t, preview.Summary, "1920.00 USD")
	assert.Contains(t, preview.Summary, "Nothing is sent to the carrier")
}

func TestAssignMoveToCarrier_PreviewShowsTheCarrierItReplaces(t *testing.T) {
	t.Parallel()

	carriers := newFakeCarrierCoverage()
	carriers.coverWith(shipment.CarrierAssignmentStatusConfirmed)
	tool := newAssignMoveToCarrierTool(carriers).(*assignMoveToCarrierTool)

	preview := previewWithoutWrites(t, &carriers.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(carriers.assignParams(map[string]any{
			"replace":                  true,
			"overrideInsuranceWarning": true,
		})))
	})

	canceled := previewChange(t, preview, 1)
	assert.Equal(t, agent.PreviewOperationArchive, canceled.Operation)
	assert.Equal(t, "Canceled", fieldByPath(t, canceled, fieldStatus).After)
	voided := previewChange(t, preview, 2)
	assert.Equal(t, permission.ResourceRateConfirmation, voided.Resource)
	assert.Equal(t, "Voided", fieldByPath(t, voided, fieldStatus).After)
	assert.Contains(t, preview.Summary, "rate confirmation voided")
	assert.Contains(t, preview.Summary, "Cargo liability expires in 10 days")
}

func TestAssignMoveToCarrier_PreviewWarnsWhatTheServiceRefuses(t *testing.T) {
	t.Parallel()

	carriers := newFakeCarrierCoverage()
	carriers.refusal = errortypes.NewBusinessError("Shipment move already has a carrier assignment")
	tool := newAssignMoveToCarrierTool(carriers).(*assignMoveToCarrierTool)

	preview := previewWithoutWrites(t, &carriers.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(carriers.assignParams(nil)))
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestCancelCarrierAssignment_SendsTheMoveAndTheReason(t *testing.T) {
	t.Parallel()

	carriers := newFakeCarrierCoverage()
	carriers.coverWith(shipment.CarrierAssignmentStatusPending)

	require.NoError(t, newCancelCarrierAssignmentTool(carriers).Execute(t.Context(), executeParams(
		map[string]any{
			previewFieldShipmentMoveID: carriers.moveID().String(),
			fieldReason:                "  Carrier's truck broke down ",
		},
	)))

	require.NotNil(t, carriers.canceled)
	assert.Equal(t, carriers.moveID(), carriers.canceled.ShipmentMoveID)
	assert.Equal(t, "Carrier's truck broke down", carriers.canceled.Reason)
}

func TestCancelCarrierAssignment_RequiresAReason(t *testing.T) {
	t.Parallel()

	carriers := newFakeCarrierCoverage()
	err := newCancelCarrierAssignmentTool(carriers).(*cancelCarrierAssignmentTool).Validate(
		t.Context(),
		executeParams(map[string]any{previewFieldShipmentMoveID: carriers.moveID().String()}),
	)
	require.Error(t, err)
}

// A carrier who confirmed the rate holds an executed agreement; taking them
// off it is a person's decision.
func TestCancelCarrierAssignment_AConfirmedCarrierIsAProposal(t *testing.T) {
	t.Parallel()

	params := func(carriers *fakeCarrierCoverage) map[string]any {
		return map[string]any{
			previewFieldShipmentMoveID: carriers.moveID().String(),
			fieldReason:                "Carrier fell off",
		}
	}

	pending := newFakeCarrierCoverage()
	pending.coverWith(shipment.CarrierAssignmentStatusPending)
	tool := newCancelCarrierAssignmentTool(pending)
	assert.Equal(t, agent.TierActWithApproval,
		tool.Policy().Condition.Limit(t.Context(), executeParams(params(pending))))

	confirmed := newFakeCarrierCoverage()
	confirmed.coverWith(shipment.CarrierAssignmentStatusConfirmed)
	tool = newCancelCarrierAssignmentTool(confirmed)
	assert.Equal(t, agent.TierPropose,
		tool.Policy().Condition.Limit(t.Context(), executeParams(params(confirmed))))

	uncovered := newFakeCarrierCoverage()
	tool = newCancelCarrierAssignmentTool(uncovered)
	assert.Equal(t, agent.TierPropose,
		tool.Policy().Condition.Limit(t.Context(), executeParams(params(uncovered))))

	policy := tool.Policy()
	assert.Equal(t, permission.OpUnassign, policy.Operation)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
	assert.False(t, policy.Reversible)
}

func TestCancelCarrierAssignment_PreviewShowsTheCarrierComingOff(t *testing.T) {
	t.Parallel()

	carriers := newFakeCarrierCoverage()
	carriers.coverWith(shipment.CarrierAssignmentStatusConfirmed)
	tool := newCancelCarrierAssignmentTool(carriers).(*cancelCarrierAssignmentTool)

	preview := previewWithoutWrites(t, &carriers.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{
			previewFieldShipmentMoveID: carriers.moveID().String(),
			fieldReason:                "Carrier fell off",
		}))
	})

	canceled := previewChange(t, preview, 0)
	assert.Equal(t, "Carrier on S-3001: Ridgeline Freight", canceled.Label)
	assert.Equal(t, "Carrier fell off", fieldByPath(t, canceled, "cancellationReason").After)
	voided := previewChange(t, preview, 1)
	assert.Equal(t, carriers.standing.ID, voided.EntityID)
	require.NotNil(t, voided.Version)
	assert.Equal(t, int64(5), *voided.Version)
	move := previewChange(t, preview, 2)
	assert.Equal(t, "unassigned", fieldByPath(t, move, "coverageType").After)
	assert.Contains(t, preview.Summary, "The carrier had confirmed the rate")
	assert.Contains(t, preview.Summary, "revision 2")
}
