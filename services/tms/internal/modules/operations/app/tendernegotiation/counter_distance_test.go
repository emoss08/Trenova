package tendernegotiation

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/testutil/mocks/assignmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEvaluateCounterPricesAPerMileRateOnThePaidDistance(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	routed, paid := 400.0, 500.0
	move := &shipment.ShipmentMove{ID: pulid.MustNew("sm_"), Distance: &routed, PayDistance: &paid}
	sh := &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		TotalChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(1000)),
	}

	assignments := assignmentportsmock.NewMockAssignmentRepository(t)
	assignments.EXPECT().GetMoveByID(mock.Anything, tenantInfo, move.ID).Return(move, nil)
	shipments := shipmentportsmock.NewMockShipmentRepository(t)
	shipments.EXPECT().GetByID(mock.Anything, mock.Anything).Return(sh, nil)

	svc := &Service{l: zap.NewNop(), assignmentRepo: assignments, shipmentRepo: shipments}
	offer := &tender.TenderOffer{
		RateMethod: shipmentvocab.CarrierRateMethodPerMile,
		Tender:     &tender.Tender{ShipmentID: sh.ID, ShipmentMoveID: move.ID},
	}

	verdict := svc.evaluateCounter(
		t.Context(),
		tenantInfo,
		offer,
		decimal.RequireFromString("1.80"),
		&dispatchcontrol.TenderPolicy{CounterMarginFloor: decimal.NewFromInt(5)},
	)

	require.True(t, verdict.MarginPercent.Valid)
	assert.Equal(t, "10", verdict.MarginPercent.Decimal.Round(2).String())
}
