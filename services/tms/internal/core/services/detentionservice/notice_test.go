package detentionservice_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noticeOccurrence() *detention.DetentionOccurrence {
	snap := baseSnapshot()

	return &detention.DetentionOccurrence{
		ID:                 pulid.MustNew("dto_"),
		ShipmentID:         pulid.MustNew("shp_"),
		PolicySnapshot:     &snap,
		ArrivedAt:          ptr(baseTime),
		FreeTimeExpiresAt:  baseTime + 2*hour,
		FreeMinutesGranted: 120,
		RawDwellMinutes:    300,
		RoundedMinutes:     180,
		BillableAmount:     money("225.00"),
		Currency:           "USD",
	}
}

func TestBuildNotice_WarningCitesGoverningTerms(t *testing.T) {
	t.Parallel()

	content := detentionservice.BuildNotice(detentionservice.BuildNoticeParams{
		Occurrence:    noticeOccurrence(),
		Kind:          detention.NoticeKindWarning,
		FacilityName:  "Memphis DC",
		ShipmentRef:   "SHP-1001",
		ArrivalSource: detention.EvidenceSourceGeofence,
		Location:      time.UTC,
	})

	assert.Contains(t, content.Subject, "Free time expiring soon")
	assert.Contains(t, content.Subject, "SHP-1001")
	assert.Contains(t, content.Subject, "Memphis DC")

	assert.Contains(t, content.Text, "Arrived:")
	assert.Contains(t, content.Text, "Geofence",
		"naming the capture source is what makes the timestamp defensible")
	assert.Contains(t, content.Text, "Contracted free time: 2h")
	assert.Contains(t, content.Text, "Free time expires:")
	assert.Contains(t, content.Text, "75.00/hr")
	assert.NotEmpty(t, content.HTML)
}

func TestBuildNotice_FinalIncludesTheSettledNumbers(t *testing.T) {
	t.Parallel()

	occ := noticeOccurrence()
	occ.DepartedAt = ptr(baseTime + 5*hour)

	content := detentionservice.BuildNotice(detentionservice.BuildNoticeParams{
		Occurrence:   occ,
		Kind:         detention.NoticeKindFinal,
		FacilityName: "Memphis DC",
		ShipmentRef:  "SHP-1001",
		Location:     time.UTC,
	})

	assert.Contains(t, content.Subject, "Detention summary")
	assert.Contains(t, content.Text, "Departed:")
	assert.Contains(t, content.Text, "Total time on site: 5h")
	assert.Contains(t, content.Text, "Billable detention: 3h")
	assert.Contains(t, content.Text, "Detention charge: 225.00 USD")
	assert.Contains(t, content.Text, "will appear on the invoice")
}

func TestBuildNotice_TieredRatesAreQuotedInFull(t *testing.T) {
	t.Parallel()

	occ := noticeOccurrence()
	snap := *occ.PolicySnapshot
	snap.RateSource = detention.RateSourceTiers
	snap.Tiers = []detention.TierSnapshot{
		{FromMinute: 0, ToMinute: ptr(int32(120)), Rate: money("75.00"), RateUnit: detention.TierRateUnitHour},
		{FromMinute: 120, ToMinute: nil, Rate: money("100.00"), RateUnit: detention.TierRateUnitHour},
	}
	occ.PolicySnapshot = &snap

	content := detentionservice.BuildNotice(detentionservice.BuildNoticeParams{
		Occurrence: occ,
		Kind:       detention.NoticeKindStarted,
		Location:   time.UTC,
	})

	assert.Contains(t, content.Text, "75.00/hr")
	assert.Contains(t, content.Text, "100.00/hr")
	assert.Contains(t, content.Text, "and beyond",
		"the escalated band must be spelled out so the customer is not surprised")
}

func TestBuildNotice_FallsBackWhenContextIsMissing(t *testing.T) {
	t.Parallel()

	occ := noticeOccurrence()

	content := detentionservice.BuildNotice(detentionservice.BuildNoticeParams{
		Occurrence: occ,
		Kind:       detention.NoticeKindUpdate,
	})

	assert.Contains(t, content.Text, "the facility")
	assert.Contains(t, content.Subject, occ.ShipmentID.String())
}

func TestBuildNotice_KindsProduceDistinctOpenings(t *testing.T) {
	t.Parallel()

	kinds := []detention.NoticeKind{
		detention.NoticeKindWarning,
		detention.NoticeKindStarted,
		detention.NoticeKindUpdate,
		detention.NoticeKindFinal,
	}

	seen := map[string]bool{}

	for _, kind := range kinds {
		content := detentionservice.BuildNotice(detentionservice.BuildNoticeParams{
			Occurrence:   noticeOccurrence(),
			Kind:         kind,
			FacilityName: "Memphis DC",
			ShipmentRef:  "SHP-1001",
			Location:     time.UTC,
		})

		require.NotEmpty(t, content.Text)
		assert.False(t, seen[content.Subject], "subject for %s duplicates another kind", kind)
		seen[content.Subject] = true
	}
}

func TestBuildNotice_RespectsTimezone(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	central := detentionservice.BuildNotice(detentionservice.BuildNoticeParams{
		Occurrence: noticeOccurrence(),
		Kind:       detention.NoticeKindWarning,
		Location:   loc,
	})
	utc := detentionservice.BuildNotice(detentionservice.BuildNoticeParams{
		Occurrence: noticeOccurrence(),
		Kind:       detention.NoticeKindWarning,
		Location:   time.UTC,
	})

	assert.NotEqual(t, central.Text, utc.Text,
		"timestamps must render in the carrier's local time")
	assert.Contains(t, central.Text, "CST")
}
