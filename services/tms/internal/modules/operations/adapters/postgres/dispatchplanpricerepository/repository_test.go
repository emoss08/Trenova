package dispatchplanpricerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestReplace_RefusesPricesThatDoNotBelongToTheRequest(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	moveID := pulid.MustNew("smv_")
	generatedAt := int64(1_700_000_000)

	valid := func() *dispatchplanprice.DispatchPlanPrice {
		return &dispatchplanprice.DispatchPlanPrice{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			MoveID:         moveID,
			GeneratedAt:    generatedAt,
		}
	}

	tests := []struct {
		name   string
		mutate func(*dispatchplanprice.DispatchPlanPrice)
	}{
		{
			name:   "another organization",
			mutate: func(p *dispatchplanprice.DispatchPlanPrice) { p.OrganizationID = pulid.MustNew("org_") },
		},
		{
			name:   "another business unit",
			mutate: func(p *dispatchplanprice.DispatchPlanPrice) { p.BusinessUnitID = pulid.MustNew("bu_") },
		},
		{
			name:   "a move outside the plan",
			mutate: func(p *dispatchplanprice.DispatchPlanPrice) { p.MoveID = pulid.MustNew("smv_") },
		},
		{
			name:   "an older plan's price",
			mutate: func(p *dispatchplanprice.DispatchPlanPrice) { p.GeneratedAt = generatedAt - 1 },
		},
		{
			name:   "a newer plan's price",
			mutate: func(p *dispatchplanprice.DispatchPlanPrice) { p.GeneratedAt = generatedAt + 1 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stray := valid()
			tt.mutate(stray)
			repo := New(Params{Logger: zap.NewNop()})

			err := repo.Replace(t.Context(), &dispatchplanports.ReplacePricesRequest{
				TenantInfo:  tenant,
				GeneratedAt: generatedAt,
				MoveIDs:     []pulid.ID{moveID},
				Prices:      []*dispatchplanprice.DispatchPlanPrice{valid(), stray},
			})

			require.Error(t, err)
			assert.True(t, errortypes.IsBusinessError(err), "got %v", err)
		})
	}
}

func TestReplace_PlanWithNoMovesTouchesNothing(t *testing.T) {
	t.Parallel()

	repo := New(Params{Logger: zap.NewNop()})

	assert.NoError(t, repo.Replace(t.Context(), &dispatchplanports.ReplacePricesRequest{}))
}

func TestListByMoves_NoMovesReadsNothing(t *testing.T) {
	t.Parallel()

	repo := New(Params{Logger: zap.NewNop()})

	prices, err := repo.ListByMoves(t.Context(), &dispatchplanports.ListPricesRequest{})

	require.NoError(t, err)
	assert.NotNil(t, prices)
	assert.Empty(t, prices)
}
