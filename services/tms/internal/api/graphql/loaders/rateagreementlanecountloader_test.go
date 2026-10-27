package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateAgreementLaneCountBatchFunc_CountsEveryAgreementInOneRead(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	small := pulid.MustNew("ragr_")
	large := pulid.MustNew("ragr_")
	reads := 0
	repo := &stubRateAgreementRepository{
		countLiveLanes: func(
			_ context.Context,
			req *rateagreementports.CountLiveLanesRequest,
		) (map[pulid.ID]rateagreementports.LaneCount, error) {
			reads++
			assert.Equal(t, tenantInfo, req.TenantInfo)
			assert.Equal(t, []pulid.ID{small, large}, req.RateAgreementIDs)
			return map[pulid.ID]rateagreementports.LaneCount{
				small: {Count: 12, Exact: true},
				large: {Count: 1_250_000},
			}, nil
		},
	}
	factory := &RateAgreementLaneCountLoaderFactory{agreements: repo}

	values, errs := factory.batchFunc(tenantInfo)(t.Context(), []string{
		small.String(),
		large.String(),
		small.String(),
	})

	assert.Equal(t, 1, reads, "the laneCount and laneCountExact of every agreement share one read")
	require.Len(t, values, 3)
	for _, err := range errs {
		require.NoError(t, err)
	}
	assert.Equal(t, rateagreementports.LaneCount{Count: 12, Exact: true}, values[0])
	assert.Equal(t, rateagreementports.LaneCount{Count: 1_250_000}, values[1])
	assert.Equal(t, values[0], values[2])
}

func TestRateAgreementLaneCountBatchFunc_RepositoryErrorFillsEveryKey(t *testing.T) {
	t.Parallel()

	agreementID := pulid.MustNew("ragr_")
	repoErr := errors.New("repository failed")
	repo := &stubRateAgreementRepository{
		countLiveLanes: func(
			context.Context,
			*rateagreementports.CountLiveLanesRequest,
		) (map[pulid.ID]rateagreementports.LaneCount, error) {
			return nil, repoErr
		},
	}
	factory := &RateAgreementLaneCountLoaderFactory{agreements: repo}

	_, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		"bad",
		agreementID.String(),
	})

	require.Error(t, errs[0])
	require.NotErrorIs(t, errs[0], repoErr)
	require.ErrorIs(t, errs[1], repoErr)
}

type stubRateAgreementRepository struct {
	rateagreementports.RateAgreementRepository

	countLiveLanes func(
		context.Context,
		*rateagreementports.CountLiveLanesRequest,
	) (map[pulid.ID]rateagreementports.LaneCount, error)
	listFirstLanes func(
		context.Context,
		*rateagreementports.ListFirstLanesRequest,
	) (map[pulid.ID][]*rateagreement.RateAgreementRule, error)
}

func (s *stubRateAgreementRepository) ListFirstLanes(
	ctx context.Context,
	req *rateagreementports.ListFirstLanesRequest,
) (map[pulid.ID][]*rateagreement.RateAgreementRule, error) {
	return s.listFirstLanes(ctx, req)
}

func (s *stubRateAgreementRepository) CountLiveLanes(
	ctx context.Context,
	req *rateagreementports.CountLiveLanesRequest,
) (map[pulid.ID]rateagreementports.LaneCount, error) {
	return s.countLiveLanes(ctx, req)
}
