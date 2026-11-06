package dispatchplanservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPricesOf_WritesOneRowPerMoveTheyPlanned(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	generatedAt := int64(1_700_000_000)
	covered, priced, unpriced := pulid.MustNew("smv_"), pulid.MustNew("smv_"), pulid.MustNew("smv_")
	workerID := pulid.MustNew("wrk_")
	alternatives := []dispatchplanprice.Alternative{
		{WorkerID: pulid.MustNew("wrk_"), WorkerName: "Bo", ExtraCents: 650},
	}

	plan := &dispatchplanports.Plan{
		GeneratedAt: generatedAt,
		Pairings: []*dispatchplanports.Pairing{{
			MoveID: covered,
			Candidate: &dispatchplanports.CandidateScore{
				WorkerID:   workerID,
				WorkerName: "Ada",
				Cost:       dispatcheconomics.Cost{TotalCents: 1_300},
			},
			PlanCostCents:        1_234,
			PreferenceFloorCents: 66,
			CoveragePriceCents:   5_000,
			DriverValueCents:     800,
			Alternatives:         alternatives,
		}},
		Uncovered: []*dispatchplanports.Uncovered{
			{
				MoveID:             priced,
				Reason:             dispatchplanprice.UncoveredNotWorthIt,
				PlanCostCents:      900,
				CoveragePriceCents: 400,
				BestCandidate: &dispatchplanports.CandidateScore{
					WorkerID:   pulid.MustNew("wrk_"),
					WorkerName: "Cy",
					Cost:       dispatcheconomics.Cost{TotalCents: 960},
				},
			},
			{
				MoveID:             unpriced,
				Reason:             dispatchplanprice.UncoveredNoCandidate,
				CoveragePriceCents: 40_000,
			},
		},
	}

	prices := pricesOf(tenant, plan)

	require.Len(t, prices, 3)
	assert.Equal(t, &dispatchplanprice.DispatchPlanPrice{
		OrganizationID:       tenant.OrgID,
		BusinessUnitID:       tenant.BuID,
		MoveID:               covered,
		WorkerID:             workerID,
		WorkerName:           "Ada",
		CostCents:            1_234,
		PreferenceFloorCents: 66,
		CoveragePriceCents:   5_000,
		DriverValueCents:     800,
		Alternatives:         alternatives,
		GeneratedAt:          generatedAt,
	}, prices[0])
	assert.True(t, prices[0].Covered())

	assert.Equal(t, &dispatchplanprice.DispatchPlanPrice{
		OrganizationID:     tenant.OrgID,
		BusinessUnitID:     tenant.BuID,
		MoveID:             priced,
		WorkerName:         "Cy",
		CostCents:          900,
		CoveragePriceCents: 400,
		UncoveredReason:    dispatchplanprice.UncoveredNotWorthIt,
		Alternatives:       []dispatchplanprice.Alternative{},
		GeneratedAt:        generatedAt,
	}, prices[1], "an uncovered move names its best driver but never assigns one")
	assert.False(t, prices[1].Covered())

	assert.Equal(t, &dispatchplanprice.DispatchPlanPrice{
		OrganizationID:     tenant.OrgID,
		BusinessUnitID:     tenant.BuID,
		MoveID:             unpriced,
		CoveragePriceCents: 40_000,
		UncoveredReason:    dispatchplanprice.UncoveredNoCandidate,
		Alternatives:       []dispatchplanprice.Alternative{},
		GeneratedAt:        generatedAt,
	}, prices[2])
	assert.NotNil(t, prices[2].Alternatives, "alternatives store as an empty list, never null")
}

func TestPricesOf_EmptyPlanWritesNothing(t *testing.T) {
	t.Parallel()

	prices := pricesOf(pagination.TenantInfo{}, &dispatchplanports.Plan{})

	assert.NotNil(t, prices)
	assert.Empty(t, prices)
}
