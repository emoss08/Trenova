package aitraining

import (
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
)

type ExamplePage struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

type ExampleStop struct {
	Role                string `json:"role"`
	Sequence            int    `json:"sequence"`
	Name                string `json:"name,omitempty"`
	AddressLine1        string `json:"addressLine1,omitempty"`
	AddressLine2        string `json:"addressLine2,omitempty"`
	City                string `json:"city,omitempty"`
	State               string `json:"state,omitempty"`
	PostalCode          string `json:"postalCode,omitempty"`
	Date                string `json:"date,omitempty"`
	TimeWindow          string `json:"timeWindow,omitempty"`
	AppointmentRequired bool   `json:"appointmentRequired"`
}

type ExampleSnapshot struct {
	Fields map[string]string `json:"fields"`
	Stops  []ExampleStop     `json:"stops"`
}

type ExampleInput struct {
	FileName string        `json:"fileName"`
	Pages    []ExamplePage `json:"pages"`
}

type ExampleQuality struct {
	Scored              int     `json:"scored"`
	Correct             int     `json:"correct"`
	Corrected           int     `json:"corrected"`
	Missed              int     `json:"missed"`
	Unconfirmed         int     `json:"unconfirmed"`
	Unscored            int     `json:"unscored"`
	PredictedConfidence float64 `json:"predictedConfidence"`
}

type Example struct {
	ID           string                          `json:"id"`
	Format       string                          `json:"format"`
	Task         aicorrection.Task               `json:"task"`
	Split        Split                           `json:"split"`
	DocumentKind string                          `json:"documentKind,omitempty"`
	Input        ExampleInput                    `json:"input"`
	Target       *ExampleSnapshot                `json:"target"`
	Prediction   *ExampleSnapshot                `json:"prediction"`
	Outcomes     map[string]aicorrection.Outcome `json:"outcomes"`
	Quality      ExampleQuality                  `json:"quality"`
}

func OutcomesOf(results []aicorrection.FieldResult) map[string]aicorrection.Outcome {
	outcomes := make(map[string]aicorrection.Outcome, len(results))
	for i := range results {
		outcomes[results[i].Key] = results[i].Outcome
	}

	return outcomes
}

func QualityOf(correction *aicorrection.Correction) ExampleQuality {
	return ExampleQuality{
		Scored:              correction.ScoredCount,
		Correct:             correction.CorrectCount,
		Corrected:           correction.CorrectedCount,
		Missed:              correction.MissedCount,
		Unconfirmed:         correction.UnconfirmedCount,
		Unscored:            correction.UnscoredCount,
		PredictedConfidence: correction.PredictedConfidence,
	}
}
