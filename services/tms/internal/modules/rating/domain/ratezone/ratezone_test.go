package ratezone_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func zone(members ...*ratezone.RateZoneMember) *ratezone.RateZone {
	return &ratezone.RateZone{
		ID:             pulid.MustNew("rzn_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Code:           "SE",
		Name:           "Southeast",
		Members:        members,
	}
}

func TestStampMembers_KeysEveryMemberInOneCall(t *testing.T) {
	t.Parallel()

	entity := zone(
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeZip3, ScopeValue: "30301"},
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeState, ScopeValue: "ust_ga"},
		&ratezone.RateZoneMember{
			ScopeType:  rategeo.ScopeTypeCityState,
			ScopeValue: "ust_ga",
			City:       " atlanta ",
		},
	)

	require.NoError(t, entity.StampMembers(t.Context()))

	assert.Equal(t, "Z3:303", entity.Members[0].MatchKey)
	assert.Equal(t, "ST:ust_ga", entity.Members[1].MatchKey)
	assert.Equal(t, "CS:ust_ga|ATLANTA", entity.Members[2].MatchKey)
	for _, member := range entity.Members {
		assert.Equal(t, entity.ID, member.RateZoneID)
		assert.Equal(t, entity.OrganizationID, member.OrganizationID)
	}
}

func TestStampMembers_ValidationRefusesUnkeyableAndRepeatedPlaces(t *testing.T) {
	t.Parallel()

	entity := zone(
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeZip3, ScopeValue: "30"},
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeZip5, ScopeValue: "30301"},
		&ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeZip5, ScopeValue: "30301-1234"},
	)
	require.NoError(t, entity.StampMembers(t.Context()))

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	fields := make(map[string]string, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields[e.Field] = e.Message
	}
	assert.Equal(t, "Scope value is not valid for a {0} member", fields["members[0].scopeValue"])
	assert.Equal(t, "This place is already part of the zone", fields["members[2].scopeValue"])
	assert.NotContains(t, fields, "members[1].scopeValue")
}

func TestBeforeCreate_RefusesAMemberThatWasNeverKeyed(t *testing.T) {
	t.Parallel()

	member := &ratezone.RateZoneMember{ScopeType: rategeo.ScopeTypeState, ScopeValue: "ust_ga"}

	require.ErrorIs(t, member.BeforeCreate(t.Context()), ratezone.ErrMatchKeyNotApplied)
}
