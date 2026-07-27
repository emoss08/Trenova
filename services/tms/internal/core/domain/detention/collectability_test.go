package detention_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strongOccurrence() *detention.DetentionOccurrence {
	return &detention.DetentionOccurrence{
		ID:                 pulid.MustNew("dto_"),
		ArrivedAt:          ptr(int64(1000)),
		DepartedAt:         ptr(int64(20000)),
		AppointmentStart:   ptr(int64(900)),
		ScheduleType:       shipment.StopScheduleTypeAppointment,
		NotificationStatus: detention.NotificationStatusSent,
	}
}

func evidenceEntry(
	occurrenceID pulid.ID,
	sequence int32,
	kind detention.EvidenceKind,
	source detention.EvidenceSource,
	prev string,
) *detention.DetentionEvidence {
	entry := &detention.DetentionEvidence{
		ID:                    pulid.MustNew("dte_"),
		DetentionOccurrenceID: occurrenceID,
		Sequence:              sequence,
		Kind:                  kind,
		Source:                source,
		Summary:               string(kind) + " recorded",
		ObservedAt:            int64(1000 + sequence),
		RecordedAt:            int64(1000 + sequence),
		PrevHash:              prev,
	}
	entry.Seal()
	return entry
}

func strongEvidence(occurrenceID pulid.ID) []*detention.DetentionEvidence {
	kinds := []struct {
		kind   detention.EvidenceKind
		source detention.EvidenceSource
	}{
		{detention.EvidenceKindArrival, detention.EvidenceSourceGeofence},
		{detention.EvidenceKindDeparture, detention.EvidenceSourceGeofence},
		{detention.EvidenceKindDocument, detention.EvidenceSourceDocument},
	}

	entries := make([]*detention.DetentionEvidence, 0, len(kinds))
	prev := detention.GenesisHash

	for i, k := range kinds {
		entry := evidenceEntry(occurrenceID, int32(i), k.kind, k.source, prev)
		prev = entry.Hash
		entries = append(entries, entry)
	}

	return entries
}

func factorByKey(a detention.CollectabilityAssessment, key string) detention.ScoreFactor {
	for _, f := range a.Factors {
		if f.Key == key {
			return f
		}
	}
	return detention.ScoreFactor{}
}

func TestAssessCollectability_PerfectClaim(t *testing.T) {
	t.Parallel()

	occurrence := strongOccurrence()
	assessment := detention.AssessCollectability(
		occurrence, strongEvidence(occurrence.ID),
	)

	assert.Equal(t, int16(detention.ScoreMax), assessment.Score)
	assert.Equal(t, detention.ScoreBandStrong, assessment.Band)
	assert.True(t, assessment.ChainValid)
	assert.Contains(t, assessment.Summary, "Every evidence requirement is satisfied")
}

func TestAssessCollectability_NilOccurrence(t *testing.T) {
	t.Parallel()

	assessment := detention.AssessCollectability(nil, nil)

	assert.Equal(t, int16(0), assessment.Score)
	assert.Equal(t, detention.ScoreBandAtRisk, assessment.Band)
}

func TestAssessCollectability_ManualTimestampsScoreLower(t *testing.T) {
	t.Parallel()

	occurrence := strongOccurrence()

	prev := detention.GenesisHash
	arrival := evidenceEntry(
		occurrence.ID, 0, detention.EvidenceKindArrival, detention.EvidenceSourceManual, prev,
	)
	departure := evidenceEntry(
		occurrence.ID, 1, detention.EvidenceKindDeparture,
		detention.EvidenceSourceManual, arrival.Hash,
	)
	document := evidenceEntry(
		occurrence.ID, 2, detention.EvidenceKindDocument,
		detention.EvidenceSourceDocument, departure.Hash,
	)

	manual := detention.AssessCollectability(
		occurrence,
		[]*detention.DetentionEvidence{arrival, departure, document},
	)
	geofenced := detention.AssessCollectability(occurrence, strongEvidence(occurrence.ID))

	assert.Less(t, manual.Score, geofenced.Score,
		"a dispatcher keystroke must not score like a geofence crossing")

	arrivalFactor := factorByKey(manual, "arrival")
	assert.Less(t, arrivalFactor.Earned, arrivalFactor.Possible)
	assert.NotEmpty(t, arrivalFactor.Remedy)
}

func TestAssessCollectability_MissingNoticeIsHeavilyPenalized(t *testing.T) {
	t.Parallel()

	occurrence := strongOccurrence()
	occurrence.NotificationStatus = detention.NotificationStatusMissed

	assessment := detention.AssessCollectability(
		occurrence, strongEvidence(occurrence.ID),
	)

	notice := factorByKey(assessment, "notice")
	assert.Equal(t, 0, notice.Earned)
	assert.Equal(t, detention.ScoreWeightNotice, notice.Possible)
	assert.NotEmpty(t, notice.Remedy)
	assert.Equal(t, int16(detention.ScoreMax-detention.ScoreWeightNotice), assessment.Score)
}

func TestAssessCollectability_NoticeStatusGradations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    detention.NotificationStatus
		wantScore int
	}{
		{"not required earns full credit", detention.NotificationStatusNotRequired, detention.ScoreWeightNotice},
		{"sent earns full credit", detention.NotificationStatusSent, detention.ScoreWeightNotice},
		{"late earns partial credit", detention.NotificationStatusLate, detention.ScoreWeightNotice / 5},
		{"pending earns nothing yet", detention.NotificationStatusPending, 0},
		{"missed earns nothing", detention.NotificationStatusMissed, 0},
		{"failed earns nothing", detention.NotificationStatusFailed, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			occurrence := strongOccurrence()
			occurrence.NotificationStatus = tt.status

			assessment := detention.AssessCollectability(
				occurrence, strongEvidence(occurrence.ID),
			)

			assert.Equal(t, tt.wantScore, factorByKey(assessment, "notice").Earned)
		})
	}
}

func TestAssessCollectability_OpenStopHasNoDeparture(t *testing.T) {
	t.Parallel()

	occurrence := strongOccurrence()
	occurrence.DepartedAt = nil

	assessment := detention.AssessCollectability(occurrence, nil)

	departure := factorByKey(assessment, "departure")
	assert.Equal(t, 0, departure.Earned)
	assert.Contains(t, departure.Detail, "still projected")
}

func TestAssessCollectability_MissingDocument(t *testing.T) {
	t.Parallel()

	occurrence := strongOccurrence()

	prev := detention.GenesisHash
	arrival := evidenceEntry(
		occurrence.ID, 0, detention.EvidenceKindArrival, detention.EvidenceSourceGeofence, prev,
	)
	departure := evidenceEntry(
		occurrence.ID, 1, detention.EvidenceKindDeparture,
		detention.EvidenceSourceGeofence, arrival.Hash,
	)

	assessment := detention.AssessCollectability(
		occurrence, []*detention.DetentionEvidence{arrival, departure},
	)

	document := factorByKey(assessment, "document")
	assert.Equal(t, 0, document.Earned)
	assert.Contains(t, document.Remedy, "BOL")
}

func TestAssessCollectability_BackfilledAppointmentScoresLower(t *testing.T) {
	t.Parallel()

	occurrence := strongOccurrence()
	occurrence.ScheduleType = shipment.StopScheduleTypeOpen

	assessment := detention.AssessCollectability(
		occurrence, strongEvidence(occurrence.ID),
	)

	appointment := factorByKey(assessment, "appointment")
	assert.Equal(t, detention.ScoreWeightAppointment/2, appointment.Earned)

	occurrence.AppointmentStart = nil
	noAppointment := detention.AssessCollectability(
		occurrence, strongEvidence(occurrence.ID),
	)
	assert.Equal(t, 0, factorByKey(noAppointment, "appointment").Earned)
}

func TestAssessCollectability_BrokenChainFlagsTampering(t *testing.T) {
	t.Parallel()

	occurrence := strongOccurrence()
	evidence := strongEvidence(occurrence.ID)
	evidence[1].Summary = "adjusted departure"

	assessment := detention.AssessCollectability(occurrence, evidence)

	assert.False(t, assessment.ChainValid)

	unedited := factorByKey(assessment, "unedited")
	assert.Equal(t, 0, unedited.Earned)
	assert.Contains(t, unedited.Detail, "altered or removed")
}

func TestAssessCollectability_TimestampWithoutEvidenceEntry(t *testing.T) {
	t.Parallel()

	occurrence := strongOccurrence()

	assessment := detention.AssessCollectability(occurrence, nil)

	arrival := factorByKey(assessment, "arrival")
	assert.Greater(t, arrival.Earned, 0,
		"a recorded arrival still counts for something")
	assert.Less(t, arrival.Earned, arrival.Possible,
		"but an unattributed timestamp cannot earn full credit")
	assert.Contains(t, arrival.Detail, "no evidence entry")
}

func TestAssessCollectability_StrongestSourceWinsPerKind(t *testing.T) {
	t.Parallel()

	occurrence := strongOccurrence()

	prev := detention.GenesisHash
	weak := evidenceEntry(
		occurrence.ID, 0, detention.EvidenceKindArrival, detention.EvidenceSourceManual, prev,
	)
	strong := evidenceEntry(
		occurrence.ID, 1, detention.EvidenceKindArrival,
		detention.EvidenceSourceGeofence, weak.Hash,
	)

	assessment := detention.AssessCollectability(
		occurrence, []*detention.DetentionEvidence{weak, strong},
	)

	assert.Equal(t, detention.ScoreWeightArrival, factorByKey(assessment, "arrival").Earned,
		"corroborating a manual entry with a geofence crossing should upgrade the claim")
}

func TestBandForScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		score int16
		want  detention.ScoreBand
	}{
		{100, detention.ScoreBandStrong},
		{85, detention.ScoreBandStrong},
		{84, detention.ScoreBandAdequate},
		{65, detention.ScoreBandAdequate},
		{64, detention.ScoreBandWeak},
		{40, detention.ScoreBandWeak},
		{39, detention.ScoreBandAtRisk},
		{0, detention.ScoreBandAtRisk},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, detention.BandForScore(tt.score),
			"score %d", tt.score)
	}
}

func TestAssessCollectability_SummaryNamesTheBiggestGap(t *testing.T) {
	t.Parallel()

	occurrence := strongOccurrence()
	occurrence.NotificationStatus = detention.NotificationStatusMissed

	assessment := detention.AssessCollectability(
		occurrence, strongEvidence(occurrence.ID),
	)

	require.NotEmpty(t, assessment.Summary)
	assert.Contains(t, assessment.Summary, "Biggest gap")
	assert.Contains(t, assessment.Summary, "notice")
}
