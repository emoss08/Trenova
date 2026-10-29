package ratingloaders

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateAgreementLanesBatchFunc_ReadsEveryAgreementsFirstLanesTogether(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	first := pulid.MustNew("ragr_")
	second := pulid.MustNew("ragr_")
	lane := &rateagreement.RateAgreementRule{ID: pulid.MustNew("rarl_"), RateAgreementID: first}
	reads := 0
	repo := &stubRateAgreementRepository{
		listFirstLanes: func(
			_ context.Context,
			req *rateagreementports.ListFirstLanesRequest,
		) (map[pulid.ID][]*rateagreement.RateAgreementRule, error) {
			reads++
			assert.Equal(t, tenantInfo, req.TenantInfo)
			assert.Equal(t, []pulid.ID{first, second}, req.RateAgreementIDs)
			assert.Equal(t, 50, req.First)
			return map[pulid.ID][]*rateagreement.RateAgreementRule{first: {lane}}, nil
		},
	}
	factory := &RateAgreementLanesLoaderFactory{agreements: repo}

	values, errs := factory.batchFunc(tenantInfo)(t.Context(), []string{
		loadercore.FirstKey(first.String(), 50),
		loadercore.FirstKey(second.String(), 50),
	})

	assert.Equal(t, 1, reads)
	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	assert.Equal(t, []*rateagreement.RateAgreementRule{lane}, values[0])
	assert.Empty(t, values[1])
}
