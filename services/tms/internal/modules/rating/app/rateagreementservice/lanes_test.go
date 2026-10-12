package rateagreementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/testutil/mocks/rateagreementportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var anyLaneRequest = mock.AnythingOfType("*rateagreementports.ListRateAgreementLanesRequest")

// A card asks for a handful of lanes; whatever page it sends, the read that
// reaches the database is bounded, so no request can pull an agreement's tens
// of thousands of lanes in one go.
func TestListLanes_BoundsThePageBeforeReading(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		limit      int
		offset     int
		wantLimit  int
		wantOffset int
	}{
		{"missing limit takes the default", 0, 0, rateagreementports.DefaultLaneLimit, 0},
		{"negative limit takes the default", -4, 0, rateagreementports.DefaultLaneLimit, 0},
		{"limit within bounds is kept", 25, 16, 25, 16},
		{"limit above the cap is clamped", 500, 0, rateagreementports.MaxLaneLimit, 0},
		{"negative offset starts at the first lane", 8, -3, 8, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tenant := pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			}
			agreementID := pulid.MustNew("ragr_")
			lane := &rateagreement.RateAgreementRule{ID: pulid.MustNew("rarl_")}

			repo := rateagreementportsmock.NewMockRateAgreementRepository(t)
			repo.EXPECT().
				ListLanes(mock.Anything, anyLaneRequest).
				RunAndReturn(func(
					_ context.Context,
					req *rateagreementports.ListRateAgreementLanesRequest,
				) (*rateagreementports.RateAgreementLanePage, error) {
					assert.Equal(t, tenant, req.TenantInfo)
					assert.Equal(t, agreementID, req.RateAgreementID)
					assert.Equal(t, tt.wantLimit, req.Limit)
					assert.Equal(t, tt.wantOffset, req.Offset)

					return &rateagreementports.RateAgreementLanePage{
						Lanes:   []*rateagreement.RateAgreementRule{lane},
						Count:   12,
						Retired: 3,
						Limit:   req.Limit,
						Offset:  req.Offset,
					}, nil
				}).
				Once()

			svc := New(Params{
				Logger:    zap.NewNop(),
				Repo:      repo,
				Validator: NewTestValidator(),
			})

			page, err := svc.ListLanes(
				t.Context(),
				&rateagreementports.ListRateAgreementLanesRequest{
					TenantInfo:      tenant,
					RateAgreementID: agreementID,
					Limit:           tt.limit,
					Offset:          tt.offset,
				},
			)

			require.NoError(t, err)
			require.Len(t, page.Lanes, 1)
			assert.Equal(t, lane.ID, page.Lanes[0].ID)
			assert.Equal(t, 12, page.Count)
			assert.Equal(t, 3, page.Retired)
			assert.Equal(t, tt.wantLimit, page.Limit)
			assert.Equal(t, tt.wantOffset, page.Offset)
		})
	}
}

// An agreement outside the caller's tenant surfaces as not found, unchanged,
// so the handler answers 404 exactly as it does for the agreement itself.
func TestListLanes_PassesNotFoundThrough(t *testing.T) {
	t.Parallel()

	notFound := errortypes.NewNotFoundError(
		"{0} not found within your organization",
		"RateAgreement",
	)

	repo := rateagreementportsmock.NewMockRateAgreementRepository(t)
	repo.EXPECT().
		ListLanes(mock.Anything, anyLaneRequest).
		Return(nil, notFound).
		Once()

	svc := New(Params{
		Logger:    zap.NewNop(),
		Repo:      repo,
		Validator: NewTestValidator(),
	})

	page, err := svc.ListLanes(t.Context(), &rateagreementports.ListRateAgreementLanesRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		RateAgreementID: pulid.MustNew("ragr_"),
	})

	require.Error(t, err)
	assert.Nil(t, page)
	assert.True(t, errortypes.IsNotFoundError(err))
}
