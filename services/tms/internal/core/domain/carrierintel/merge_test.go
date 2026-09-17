package carrierintel_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fullFetchedAt = testNow - 3*timeutils.SecondsPerDay

func fullProfileBelowRequired() *carrierintel.Profile {
	p := &carrierintel.Profile{
		Identity: &carrierintel.Identity{
			DOTNumber:    "265752",
			DocketNumber: "144457",
			DocketPrefix: "MC",
			LegalName:    "FEDEX GROUND PACKAGE SYSTEM INC",
			DBAName:      "FEDEX GROUND",
			USDOTStatus:  "ACTIVE",
			PhysicalAddress: &carrierintel.Address{
				Line1: "1000 FEDEX DR",
				City:  "MOON TOWNSHIP",
				State: "PA",
			},
		},
		Authority: &carrierintel.Authority{
			Common: &carrierintel.AuthorityGrant{
				Status:    carrierintel.AuthorityStatusActive,
				GrantedAt: new(int64(631152000)),
				AgeDays:   new(12000),
			},
			History: []carrierintel.AuthorityHistoryEntry{
				{AuthorityType: "COMMON", Action: "GRANTED", EffectiveAt: new(int64(631152000))},
			},
		},
		Insurance: &carrierintel.Insurance{
			BIPDOnFile:   dec(1_000_000),
			BIPDRequired: dec(5_000_000),
			BondRequired: dec(75_000),
			Filings: []carrierintel.InsuranceFiling{
				{Type: carrierintel.InsuranceFilingTypeBIPD, InsurerName: "ACME MUTUAL"},
			},
		},
		Safety: &carrierintel.Safety{
			Rating:     carrierintel.SafetyRatingSatisfactory,
			RatingDate: new(int64(843609600)),
			RiskScore:  carrierintel.RiskLevelLow,
		},
		Network: &carrierintel.Network{
			SharedPhones: new(2),
			Links: []carrierintel.NetworkLink{
				{Kind: carrierintel.NetworkKindPhone, DOTNumber: "111111"},
			},
		},
		Lanes: &carrierintel.Lanes{TotalLoads: new(40)},
	}
	p.NormalizeCoverage()
	return p
}

func liteProfile() *carrierintel.Profile {
	p := &carrierintel.Profile{
		Identity: &carrierintel.Identity{
			DOTNumber:   "265752",
			LegalName:   "FEDEX GROUND PACKAGE SYSTEM INC",
			USDOTStatus: "ACTIVE",
		},
		Authority: &carrierintel.Authority{
			Common: &carrierintel.AuthorityGrant{Status: carrierintel.AuthorityStatusUnknown},
		},
		Insurance: &carrierintel.Insurance{
			BIPDOnFile: dec(1_000_000),
		},
		Safety: &carrierintel.Safety{
			Rating:    carrierintel.SafetyRatingSatisfactory,
			RiskScore: carrierintel.RiskLevelUnknown,
		},
		Fleet: &carrierintel.Fleet{PowerUnits: new(1500)},
	}
	p.NormalizeCoverage()
	return p
}

func snapshotAt(
	profile *carrierintel.Profile,
	depth carrierintel.LookupDepth,
	fetchedAt int64,
) *carrierintel.CarrierIntelSnapshot {
	return &carrierintel.CarrierIntelSnapshot{
		Depth:          depth,
		FetchedDepth:   depth,
		FetchedAt:      fetchedAt,
		DepthFetchedAt: fetchedAt,
		Profile:        profile,
	}
}

func TestResolveIncomingProfile(t *testing.T) {
	t.Parallel()

	control := carrierintel.NewDefaultControl(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	expiredFullAt := testNow - control.FullProfileTTLSeconds() - timeutils.SecondsPerDay

	tests := []struct {
		name           string
		current        *carrierintel.CarrierIntelSnapshot
		incoming       *carrierintel.Profile
		incomingDepth  carrierintel.LookupDepth
		wantMerged     bool
		wantDepth      carrierintel.LookupDepth
		wantDepthAsOf  int64
		wantBIPDReq    bool
		wantBlocker    bool
		wantPowerUnits bool
	}{
		{
			name: "lite after fresh full keeps full-only data and the blocker",
			current: snapshotAt(
				fullProfileBelowRequired(),
				carrierintel.LookupDepthFull,
				fullFetchedAt,
			),
			incoming:       liteProfile(),
			incomingDepth:  carrierintel.LookupDepthLite,
			wantMerged:     true,
			wantDepth:      carrierintel.LookupDepthFull,
			wantDepthAsOf:  fullFetchedAt,
			wantBIPDReq:    true,
			wantBlocker:    true,
			wantPowerUnits: true,
		},
		{
			name: "lite after expired full drops full-only data",
			current: snapshotAt(
				fullProfileBelowRequired(),
				carrierintel.LookupDepthFull,
				expiredFullAt,
			),
			incoming:       liteProfile(),
			incomingDepth:  carrierintel.LookupDepthLite,
			wantDepth:      carrierintel.LookupDepthLite,
			wantDepthAsOf:  testNow,
			wantPowerUnits: true,
		},
		{
			name:          "full after lite replaces everything",
			current:       snapshotAt(liteProfile(), carrierintel.LookupDepthLite, fullFetchedAt),
			incoming:      fullProfileBelowRequired(),
			incomingDepth: carrierintel.LookupDepthFull,
			wantDepth:     carrierintel.LookupDepthFull,
			wantDepthAsOf: testNow,
			wantBIPDReq:   true,
			wantBlocker:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := carrierintel.ResolveIncomingProfile(&carrierintel.ProfileMergeInput{
				Current:       tt.current,
				Incoming:      tt.incoming,
				IncomingDepth: tt.incomingDepth,
				Control:       control,
				Now:           testNow,
			})

			assert.Equal(t, tt.wantMerged, result.Merged)
			assert.Equal(t, tt.wantDepth, result.Depth)
			assert.Equal(t, tt.wantDepthAsOf, result.DepthFetchedAt)
			require.NotNil(t, result.Profile.Insurance)
			assert.Equal(t, tt.wantBIPDReq, result.Profile.Insurance.BIPDRequired != nil)
			assert.Equal(t, tt.wantPowerUnits, result.Profile.Fleet != nil)

			findings := evaluate(result.Profile, nil)
			finding := findingByCode(findings, carrierintel.RuleInsuranceBIPDBelow)
			if tt.wantBlocker {
				require.NotNil(t, finding)
				assert.True(t, finding.IsBlocking())
				return
			}
			assert.Nil(t, finding)
		})
	}
}

func TestMergeProfilesFieldLevel(t *testing.T) {
	t.Parallel()

	base := fullProfileBelowRequired()
	overlay := liteProfile()
	overlay.Identity.LegalName = "FEDEX GROUND PACKAGE SYSTEM, INC."
	overlay.Insurance.BIPDOnFile = dec(6_000_000)

	merged := carrierintel.MergeProfiles(base, overlay)

	assert.Equal(t, "FEDEX GROUND PACKAGE SYSTEM, INC.", merged.Identity.LegalName)
	assert.Equal(t, "FEDEX GROUND", merged.Identity.DBAName)
	assert.Equal(t, "144457", merged.Identity.DocketNumber)
	assert.Equal(t, "MC", merged.Identity.DocketPrefix)
	assert.Equal(t, "1000 FEDEX DR", merged.Identity.PhysicalAddress.Line1)

	assert.Equal(t, carrierintel.AuthorityStatusActive, merged.Authority.Common.Status)
	assert.NotNil(t, merged.Authority.Common.GrantedAt)
	assert.Len(t, merged.Authority.History, 1)

	assert.True(t, merged.Insurance.BIPDOnFile.Equal(*dec(6_000_000)))
	assert.True(t, merged.Insurance.BIPDRequired.Equal(*dec(5_000_000)))
	assert.True(t, merged.Insurance.BondRequired.Equal(*dec(75_000)))
	assert.Len(t, merged.Insurance.Filings, 1)

	assert.Equal(t, int64(843609600), *merged.Safety.RatingDate)
	assert.Equal(t, carrierintel.RiskLevelLow, merged.Safety.RiskScore)
	assert.Len(t, merged.Network.Links, 1)
	assert.Equal(t, 1500, *merged.Fleet.PowerUnits)

	for _, section := range []carrierintel.Section{
		carrierintel.SectionIdentity,
		carrierintel.SectionAuthority,
		carrierintel.SectionInsurance,
		carrierintel.SectionSafety,
		carrierintel.SectionFleet,
		carrierintel.SectionNetwork,
		carrierintel.SectionLanes,
	} {
		assert.True(t, merged.Covers(section), section)
	}

	assert.True(t, base.Insurance.BIPDOnFile.Equal(*dec(1_000_000)))
	assert.Nil(t, base.Fleet)
}

func TestSnapshotFreshnessTracksDeeperFetch(t *testing.T) {
	t.Parallel()

	merged := &carrierintel.CarrierIntelSnapshot{
		Depth:          carrierintel.LookupDepthFull,
		FetchedDepth:   carrierintel.LookupDepthLite,
		FetchedAt:      testNow - timeutils.SecondsPerDay,
		DepthFetchedAt: testNow - 29*timeutils.SecondsPerDay,
	}
	fullTTL := int64(30 * timeutils.SecondsPerDay)
	liteTTL := int64(2 * timeutils.SecondsPerDay)

	assert.True(t, merged.IsFreshForDepth(testNow, liteTTL, carrierintel.LookupDepthLite))
	assert.True(t, merged.IsFreshForDepth(testNow, fullTTL, carrierintel.LookupDepthFull))
	later := testNow + 2*timeutils.SecondsPerDay
	assert.False(t, merged.IsFreshForDepth(later, fullTTL, carrierintel.LookupDepthFull))
	assert.Equal(t, merged.DepthFetchedAt, merged.AsOfForDepth(carrierintel.LookupDepthFull))

	confirmedAt := testNow
	full := &carrierintel.CarrierIntelSnapshot{
		Depth:          carrierintel.LookupDepthFull,
		FetchedDepth:   carrierintel.LookupDepthFull,
		FetchedAt:      testNow - 40*timeutils.SecondsPerDay,
		DepthFetchedAt: testNow - 40*timeutils.SecondsPerDay,
		ConfirmedAt:    &confirmedAt,
	}
	assert.Equal(t, confirmedAt, full.DepthAsOf())
	assert.True(t, full.IsFreshForDepth(testNow, fullTTL, carrierintel.LookupDepthFull))
}
