package carrierinteldiff_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierinteldiff"
	"github.com/emoss08/trenova/internal/modules/parties/partiestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func changeByPath(
	changes []carrierinteldiff.FieldChange,
	path string,
) *carrierinteldiff.FieldChange {
	for idx := range changes {
		if changes[idx].Path == path {
			return &changes[idx]
		}
	}
	return nil
}

func TestDiffProfiles_DetectsMeaningfulChanges(t *testing.T) {
	t.Parallel()

	prior := partiestest.HealthyCarrierProfile()
	current := partiestest.HealthyCarrierProfile()
	current.Authority.Common.Status = carrierintel.AuthorityStatusRevoked
	current.Insurance.BIPDOnFile = partiestest.Decimal(0)
	current.Basics[0].Alert = true
	current.Authority.Common.AgeDays = new(901)

	changes, err := carrierinteldiff.DiffProfiles(prior, current, carrierinteldiff.DiffOptions{})
	require.NoError(t, err)

	authority := changeByPath(changes, "authority.common.status")
	require.NotNil(t, authority)
	assert.Equal(t, carrierintel.SeverityCritical, authority.Severity)
	assert.Equal(t, "Active", authority.Prior)
	assert.Equal(t, "Revoked", authority.Current)

	bipd := changeByPath(changes, "insurance.bipdOnFile")
	require.NotNil(t, bipd)
	assert.Equal(t, carrierintel.SectionInsurance, bipd.Section)

	basic := changeByPath(changes, "basics.UnsafeDriving.alert")
	require.NotNil(t, basic)
	assert.Equal(t, carrierintel.SeverityHigh, basic.Severity)

	assert.Nil(t, changeByPath(changes, "authority.common.ageDays"))
}

func TestDiffProfiles_IgnoresSectionsNotCoveredByBoth(t *testing.T) {
	t.Parallel()

	prior := partiestest.HealthyCarrierProfile()
	prior.Network = nil
	prior.Coverage = nil
	prior.NormalizeCoverage()

	current := partiestest.HealthyCarrierProfile()
	current.Network.SharedPhones = new(4)

	changes, err := carrierinteldiff.DiffProfiles(prior, current, carrierinteldiff.DiffOptions{})
	require.NoError(t, err)
	assert.Nil(t, changeByPath(changes, "network.sharedPhones"))
}

func TestDiffProfiles_Identical(t *testing.T) {
	t.Parallel()

	changes, err := carrierinteldiff.DiffProfiles(
		partiestest.HealthyCarrierProfile(),
		partiestest.HealthyCarrierProfile(),
		carrierinteldiff.DiffOptions{},
	)
	require.NoError(t, err)
	assert.Empty(t, changes)

	a, err := partiestest.HealthyCarrierProfile().ContentHash()
	require.NoError(t, err)
	b, err := partiestest.HealthyCarrierProfile().ContentHash()
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

func TestDiffOptionsForDepths(t *testing.T) {
	t.Parallel()

	assert.False(t, carrierinteldiff.DiffOptionsForDepths(
		carrierintel.LookupDepthFull, carrierintel.LookupDepthFull,
	).IgnoreMissing)
	assert.True(t, carrierinteldiff.DiffOptionsForDepths(
		carrierintel.LookupDepthLite, carrierintel.LookupDepthFull,
	).IgnoreMissing)
	assert.True(t, carrierinteldiff.DiffOptionsForDepths(
		carrierintel.LookupDepthFull, carrierintel.LookupDepthLite,
	).IgnoreMissing)
}

func TestDiffProfiles_DepthChangeSkipsFieldsTheShallowerEndpointOmits(t *testing.T) {
	t.Parallel()

	lite := partiestest.HealthyCarrierProfile()
	lite.Safety.Rating = ""
	lite.Safety.RatingDate = nil
	lite.Insurance.BIPDRequired = nil
	lite.Identity.DOTAgeDays = nil

	full := partiestest.HealthyCarrierProfile()
	full.Safety.RatingDate = new(int64(843609600))
	full.Insurance.BIPDRequired = partiestest.Decimal(5_000_000)
	full.Safety.ISSValue = new(69)

	for _, pair := range [][2]*carrierintel.Profile{{lite, full}, {full, lite}} {
		changes, err := carrierinteldiff.DiffProfiles(
			pair[0],
			pair[1],
			carrierinteldiff.DiffOptionsForDepths(
				carrierintel.LookupDepthLite,
				carrierintel.LookupDepthFull,
			),
		)
		require.NoError(t, err)

		assert.Nil(t, changeByPath(changes, "safety.rating"))
		assert.Nil(t, changeByPath(changes, "safety.ratingDate"))
		assert.Nil(t, changeByPath(changes, "insurance.bipdRequired"))

		iss := changeByPath(changes, "safety.issValue")
		require.NotNil(t, iss, "a value present at both depths still produces a change")
		assert.Len(t, changes, 1)
	}

	changes, err := carrierinteldiff.DiffProfiles(lite, full, carrierinteldiff.DiffOptions{})
	require.NoError(t, err)
	require.NotNil(t, changeByPath(changes, "safety.ratingDate"))
	assert.Nil(t, changeByPath(changes, "safety.ratingDate").Prior)
}

func TestDiffProfiles_IgnoresBasicMeasurementDates(t *testing.T) {
	t.Parallel()

	prior := partiestest.HealthyCarrierProfile()
	prior.Basics[0].MeasuredAt = new(int64(1_700_000_000))
	current := partiestest.HealthyCarrierProfile()
	current.Basics[0].MeasuredAt = new(int64(1_710_000_000))
	current.Basics[0].Violations = new(3)

	changes, err := carrierinteldiff.DiffProfiles(prior, current, carrierinteldiff.DiffOptions{})
	require.NoError(t, err)
	assert.Nil(t, changeByPath(changes, "basics.UnsafeDriving.measuredAt"))
	assert.NotNil(t, changeByPath(changes, "basics.UnsafeDriving.violations"))
}
