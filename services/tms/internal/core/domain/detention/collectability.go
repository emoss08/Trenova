package detention

import "fmt"

// Score weights. They sum to 100 and are ordered by how often each factor
// decides a real dispute: timestamps and the written notice carry the claim,
// paperwork corroborates it, and an untouched record defends it.
const (
	ScoreWeightArrival     = 25
	ScoreWeightDeparture   = 20
	ScoreWeightNotice      = 25
	ScoreWeightDocument    = 15
	ScoreWeightAppointment = 10
	ScoreWeightUnedited    = 5

	ScoreMax = ScoreWeightArrival + ScoreWeightDeparture + ScoreWeightNotice +
		ScoreWeightDocument + ScoreWeightAppointment + ScoreWeightUnedited
)

// ScoreBand buckets a collectability score for display.
type ScoreBand string

const (
	ScoreBandStrong   = ScoreBand("Strong")
	ScoreBandAdequate = ScoreBand("Adequate")
	ScoreBandWeak     = ScoreBand("Weak")
	ScoreBandAtRisk   = ScoreBand("AtRisk")
)

func BandForScore(score int16) ScoreBand {
	switch {
	case score >= 85:
		return ScoreBandStrong
	case score >= 65:
		return ScoreBandAdequate
	case score >= 40:
		return ScoreBandWeak
	default:
		return ScoreBandAtRisk
	}
}

// ScoreFactor is one line of the defensibility rubric. Remedy is the concrete
// next action, so the UI can offer a fix rather than just a complaint.
type ScoreFactor struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Earned   int    `json:"earned"`
	Possible int    `json:"possible"`
	Detail   string `json:"detail"`
	Remedy   string `json:"remedy,omitempty"`
}

// CollectabilityAssessment answers "will this charge survive a dispute" before
// the invoice goes out, rather than after the customer rejects it.
type CollectabilityAssessment struct {
	Score      int16         `json:"score"`
	Band       ScoreBand     `json:"band"`
	Factors    []ScoreFactor `json:"factors"`
	ChainValid bool          `json:"chainValid"`
	Summary    string        `json:"summary"`
}

// AssessCollectability grades an occurrence against the evidence a carrier
// actually needs to win a detention dispute: a timestamped arrival, a
// timestamped departure, written notice inside the contractual window, a signed
// document, an appointment recorded up front, and a record nobody edited after
// the fact.
func AssessCollectability(
	occurrence *DetentionOccurrence,
	evidence []*DetentionEvidence,
) CollectabilityAssessment {
	if occurrence == nil {
		return CollectabilityAssessment{Band: ScoreBandAtRisk}
	}

	factors := make([]ScoreFactor, 0, 6)
	index := indexEvidence(evidence)

	factors = append(factors,
		scoreTimestamp(scoreTimestampInput{
			key:      "arrival",
			label:    "Arrival timestamp",
			weight:   ScoreWeightArrival,
			present:  occurrence.ArrivedAt != nil,
			source:   index[EvidenceKindArrival],
			missing:  "No arrival was recorded, so the clock has no defensible start.",
			remedy:   "Record the arrival from telematics or the driver app.",
			weakHint: "Confirm the arrival against a geofence crossing or gate log.",
		}),
		scoreTimestamp(scoreTimestampInput{
			key:      "departure",
			label:    "Departure timestamp",
			weight:   ScoreWeightDeparture,
			present:  occurrence.DepartedAt != nil,
			source:   index[EvidenceKindDeparture],
			missing:  "The driver has not departed, so the total is still projected.",
			remedy:   "Record the departure to close the clock.",
			weakHint: "Confirm the departure against a geofence crossing or POD.",
		}),
		scoreNotice(occurrence),
		scoreDocument(index),
		scoreAppointment(occurrence),
	)

	chainValid := VerifyChain(evidence) == -1
	factors = append(factors, scoreUnedited(chainValid))

	total := 0
	for _, factor := range factors {
		total += factor.Earned
	}

	score := int16(total)

	return CollectabilityAssessment{
		Score:      score,
		Band:       BandForScore(score),
		Factors:    factors,
		ChainValid: chainValid,
		Summary:    summarize(score, factors),
	}
}

type scoreTimestampInput struct {
	key      string
	label    string
	weight   int
	present  bool
	source   EvidenceSource
	missing  string
	remedy   string
	weakHint string
}

// scoreTimestamp grades a captured time by how the capture happened. A geofence
// crossing earns full credit; a dispatcher typing a number earns a fraction of
// it, because that is roughly how much weight each carries in a dispute.
func scoreTimestamp(in scoreTimestampInput) ScoreFactor {
	factor := ScoreFactor{
		Key:      in.key,
		Label:    in.label,
		Possible: in.weight,
	}

	if !in.present {
		factor.Detail = in.missing
		factor.Remedy = in.remedy
		return factor
	}

	if in.source == "" {
		factor.Earned = in.weight * EvidenceSourceManual.Weight() / 100
		factor.Detail = "Recorded, but with no evidence entry naming the source."
		factor.Remedy = in.weakHint
		return factor
	}

	factor.Earned = in.weight * in.source.Weight() / 100
	factor.Detail = fmt.Sprintf("Captured from %s.", in.source)

	if factor.Earned < in.weight {
		factor.Remedy = in.weakHint
	}

	return factor
}

func scoreNotice(occurrence *DetentionOccurrence) ScoreFactor {
	factor := ScoreFactor{
		Key:      "notice",
		Label:    "Customer notice",
		Possible: ScoreWeightNotice,
	}

	switch occurrence.NotificationStatus {
	case NotificationStatusNotRequired:
		factor.Earned = ScoreWeightNotice
		factor.Detail = "This contract does not require advance notice."
	case NotificationStatusSent:
		factor.Earned = ScoreWeightNotice
		factor.Detail = "Notice was delivered inside the contractual window."
	case NotificationStatusPending:
		factor.Detail = "The notice window is open but nothing has been sent."
		factor.Remedy = "Send the detention notice before the deadline passes."
	case NotificationStatusLate:
		factor.Earned = ScoreWeightNotice / 5
		factor.Detail = "Notice went out after the contractual deadline."
		factor.Remedy = "Expect a challenge; attach the delivery receipt to the claim."
	case NotificationStatusMissed:
		factor.Detail = "No notice was sent inside the contractual window."
		factor.Remedy = "Document why the notice was missed before billing."
	case NotificationStatusFailed:
		factor.Detail = "The notice bounced and never reached the customer."
		factor.Remedy = "Correct the customer contact and resend."
	}

	return factor
}

func scoreDocument(index map[EvidenceKind]EvidenceSource) ScoreFactor {
	factor := ScoreFactor{
		Key:      "document",
		Label:    "Supporting document",
		Possible: ScoreWeightDocument,
	}

	if _, ok := index[EvidenceKindDocument]; !ok {
		factor.Detail = "No signed BOL or POD is attached to this occurrence."
		factor.Remedy = "Attach the signed BOL or POD showing facility timestamps."
		return factor
	}

	factor.Earned = ScoreWeightDocument
	factor.Detail = "A signed document is attached."

	return factor
}

// scoreAppointment rewards an appointment that existed before the truck showed
// up. An appointment backfilled after arrival is the first thing a customer
// attacks.
func scoreAppointment(occurrence *DetentionOccurrence) ScoreFactor {
	factor := ScoreFactor{
		Key:      "appointment",
		Label:    "Scheduled appointment",
		Possible: ScoreWeightAppointment,
	}

	if occurrence.AppointmentStart == nil {
		factor.Detail = "No appointment was scheduled for this stop."
		factor.Remedy = "Schedule appointments so the clock has an agreed reference."
		return factor
	}

	if occurrence.ScheduleType != "Appointment" {
		factor.Earned = ScoreWeightAppointment / 2
		factor.Detail = "The stop carries a window but is not an appointment stop."
		factor.Remedy = "Mark the stop as an appointment when the facility agrees to one."
		return factor
	}

	factor.Earned = ScoreWeightAppointment
	factor.Detail = "An appointment was on the books before arrival."

	return factor
}

func scoreUnedited(chainValid bool) ScoreFactor {
	factor := ScoreFactor{
		Key:      "unedited",
		Label:    "Untampered record",
		Possible: ScoreWeightUnedited,
	}

	if !chainValid {
		factor.Detail = "The evidence chain does not verify; a record was altered or removed."
		factor.Remedy = "Investigate before billing; this occurrence cannot be defended as-is."
		return factor
	}

	factor.Earned = ScoreWeightUnedited
	factor.Detail = "The evidence chain verifies end to end."

	return factor
}

func indexEvidence(evidence []*DetentionEvidence) map[EvidenceKind]EvidenceSource {
	index := make(map[EvidenceKind]EvidenceSource, len(evidence))

	for _, entry := range evidence {
		if entry == nil {
			continue
		}

		if existing, ok := index[entry.Kind]; ok &&
			existing.Weight() >= entry.Source.Weight() {
			continue
		}

		index[entry.Kind] = entry.Source
	}

	return index
}

func summarize(score int16, factors []ScoreFactor) string {
	weakest := ""
	gap := 0

	for _, factor := range factors {
		shortfall := factor.Possible - factor.Earned
		if shortfall > gap && factor.Remedy != "" {
			gap = shortfall
			weakest = factor.Remedy
		}
	}

	band := BandForScore(score)

	if weakest == "" {
		return fmt.Sprintf("%d/100 (%s). Every evidence requirement is satisfied.",
			score, band)
	}

	return fmt.Sprintf("%d/100 (%s). Biggest gap: %s", score, band, weakest)
}
