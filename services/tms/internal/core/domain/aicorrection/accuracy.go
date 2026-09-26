package aicorrection

import (
	"cmp"
	"regexp"
	"slices"
)

var stopIndexPattern = regexp.MustCompile(`\[\d+\]`)

type FieldAccuracy struct {
	Key         string  `json:"key"`
	Scored      int     `json:"scored"`
	Correct     int     `json:"correct"`
	Corrected   int     `json:"corrected"`
	Missed      int     `json:"missed"`
	Unconfirmed int     `json:"unconfirmed"`
	Unscored    int     `json:"unscored"`
	Accuracy    float64 `json:"accuracy"`
}

func FieldGroup(key string) string {
	return stopIndexPattern.ReplaceAllString(key, "")
}

func Accuracy(correct, scored int) float64 {
	if scored <= 0 {
		return 0
	}

	return float64(correct) / float64(scored)
}

type AccuracyAggregator struct {
	fields map[string]*FieldAccuracy
}

func NewAccuracyAggregator() *AccuracyAggregator {
	return &AccuracyAggregator{fields: map[string]*FieldAccuracy{}}
}

func (a *AccuracyAggregator) Add(results []FieldResult) {
	for i := range results {
		group := FieldGroup(results[i].Key)
		entry, ok := a.fields[group]
		if !ok {
			entry = &FieldAccuracy{Key: group}
			a.fields[group] = entry
		}
		switch results[i].Outcome {
		case OutcomeCorrect:
			entry.Correct++
		case OutcomeCorrected:
			entry.Corrected++
		case OutcomeMissed:
			entry.Missed++
		case OutcomeUnconfirmed:
			entry.Unconfirmed++
		case OutcomeUnscored:
			entry.Unscored++
		}
	}
}

func (a *AccuracyAggregator) Fields() []FieldAccuracy {
	out := make([]FieldAccuracy, 0, len(a.fields))
	for _, entry := range a.fields {
		field := *entry
		field.Scored = field.Correct + field.Corrected + field.Missed
		field.Accuracy = Accuracy(field.Correct, field.Scored)
		out = append(out, field)
	}
	slices.SortFunc(out, func(x, y FieldAccuracy) int {
		if c := cmp.Compare(x.Accuracy, y.Accuracy); c != 0 {
			return c
		}
		return cmp.Compare(x.Key, y.Key)
	})

	return out
}
