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

type FieldComparison struct {
	Key                string
	CandidateScored    int
	CandidateCorrect   int
	CandidateAccuracy  float64
	ProductionScored   int
	ProductionCorrect  int
	ProductionAccuracy float64
}

func CompareFields(candidate, production []FieldAccuracy) []FieldComparison {
	byKey := make(map[string]*FieldComparison, len(candidate)+len(production))
	entry := func(key string) *FieldComparison {
		row, ok := byKey[key]
		if !ok {
			row = &FieldComparison{Key: key}
			byKey[key] = row
		}
		return row
	}
	for i := range candidate {
		row := entry(candidate[i].Key)
		row.CandidateScored = candidate[i].Scored
		row.CandidateCorrect = candidate[i].Correct
		row.CandidateAccuracy = candidate[i].Accuracy
	}
	for i := range production {
		row := entry(production[i].Key)
		row.ProductionScored = production[i].Scored
		row.ProductionCorrect = production[i].Correct
		row.ProductionAccuracy = production[i].Accuracy
	}

	out := make([]FieldComparison, 0, len(byKey))
	for _, row := range byKey {
		if row.CandidateScored == 0 && row.ProductionScored == 0 {
			continue
		}
		out = append(out, *row)
	}
	slices.SortFunc(out, func(a, b FieldComparison) int {
		if c := cmp.Compare(
			a.CandidateAccuracy-a.ProductionAccuracy,
			b.CandidateAccuracy-b.ProductionAccuracy,
		); c != 0 {
			return c
		}
		return cmp.Compare(a.Key, b.Key)
	})

	return out
}
