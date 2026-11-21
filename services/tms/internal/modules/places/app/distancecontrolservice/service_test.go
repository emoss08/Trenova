package distancecontrolservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/places/domain/distancecontrol"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/modules/places/ports/distanceprofileports"
	"github.com/emoss08/trenova/internal/testutil/mocks/distanceprofileportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestValidateReadsEveryAssignedProfileInOneQuery(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	practical := distanceprofile.NewDefault(orgID, buID)
	practical.ID = pulid.MustNew("dp_")
	inactive := distanceprofile.NewDefault(orgID, buID)
	inactive.ID = pulid.MustNew("dp_")
	inactive.Status = distanceprofile.StatusInactive
	missing := pulid.MustNew("dp_")

	control := distancecontrol.NewDefault(orgID, buID, practical.ID)
	control.BillingDistanceProfileID = inactive.ID
	control.FuelDistanceProfileID = missing

	profiles := distanceprofileportsmock.NewMockDistanceProfileRepository(t)
	profiles.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req distanceprofileports.GetDistanceProfilesByIDsRequest) bool {
			return len(req.DistanceProfileIDs) == 3 && req.TenantInfo.OrgID == orgID
		})).
		Return([]*distanceprofile.DistanceProfile{practical, inactive}, nil).
		Once()

	svc := &Service{l: zap.NewNop(), distanceProfileRepo: profiles}
	err := svc.validate(t.Context(), control)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.True(t, multiErr.HasErrorAt("billingDistanceProfileId"))
	assert.True(t, multiErr.HasErrorAt("fuelDistanceProfileId"))
	assert.False(t, multiErr.HasErrorAt("payDistanceProfileId"))
	assert.False(t, multiErr.HasErrorAt("ratingDistanceUnit"))
}

func TestValidateFailsWhenTheProfilesCannotBeRead(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	profileID := pulid.MustNew("dp_")
	control := distancecontrol.NewDefault(orgID, buID, profileID)

	profiles := distanceprofileportsmock.NewMockDistanceProfileRepository(t)
	readErr := errors.New("connection reset")
	profiles.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return(nil, readErr)

	svc := &Service{l: zap.NewNop(), distanceProfileRepo: profiles}

	require.ErrorIs(t, svc.validate(t.Context(), control), readErr)
}
