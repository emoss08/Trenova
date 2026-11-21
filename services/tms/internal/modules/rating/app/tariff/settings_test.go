package tariff_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/places/domain/distancecontrol"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/record"
	"github.com/emoss08/trenova/internal/testutil/mocks/distancecontrolportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSettingsReadTheRatingDistanceUnit(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	tests := []struct {
		name    string
		control *distancecontrol.DistanceControl
		err     error
		want    string
	}{
		{
			name:    "kilometers rate in kilometers",
			control: &distancecontrol.DistanceControl{RatingDistanceUnit: "Kilometers"},
			want:    record.DistanceKilometers,
		},
		{
			name:    "miles rate in miles",
			control: &distancecontrol.DistanceControl{RatingDistanceUnit: "Miles"},
			want:    record.DistanceMiles,
		},
		{
			name: "no distance control rates in miles",
			err:  errortypes.NewNotFoundError("DistanceControl"),
			want: record.DistanceMiles,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := distancecontrolportsmock.NewMockDistanceControlRepository(t)
			repo.EXPECT().Get(mock.Anything, tenant).Return(tt.control, tt.err)
			settings := tariff.NewSettings(tariff.SettingsParams{DistanceControls: repo})

			got, err := settings.For(t.Context(), tenant)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got.DistanceUnit)
		})
	}
}
