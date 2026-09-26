package aicorrection

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	minContainedNameRunes = 4
	postalCodeDigits      = 5
	moneyPlaces           = 2
)

type comparison int

const (
	comparisonEqual comparison = iota
	comparisonDifferent
	comparisonUnreadable
)

type comparator func(predicted, confirmed string, stop *StopSnapshot) comparison

type fieldSpec struct {
	key     string
	aliases []string
	compare comparator
	stop    func(expected *Snapshot) *StopSnapshot
}

var fieldSpecs = []fieldSpec{
	{
		key:     FieldReference,
		aliases: []string{"referenceNumber", "bol", "reference", "loadNumber"},
		compare: compareIdentifier,
	},
	{key: FieldRate, aliases: []string{"rate"}, compare: compareMoney},
	{key: FieldWeight, aliases: []string{"weight"}, compare: compareInteger},
	{key: FieldPieces, aliases: []string{"pieceCount", "pieces"}, compare: compareInteger},
	{key: FieldCommodity, aliases: []string{"commodity"}, compare: compareName},
	{key: FieldShipper, aliases: []string{"shipper"}, compare: compareName},
	{key: FieldConsignee, aliases: []string{"consignee"}, compare: compareName},
	{
		key:     FieldPickupWindow,
		aliases: []string{"pickupWindow"},
		compare: compareDate,
		stop:    func(e *Snapshot) *StopSnapshot { return e.FirstStop(RolePickup) },
	},
	{
		key:     FieldDeliveryWindow,
		aliases: []string{"deliveryWindow"},
		compare: compareDate,
		stop:    func(e *Snapshot) *StopSnapshot { return e.LastStop(RoleDelivery) },
	},
}

type stopFieldSpec struct {
	name    string
	value   func(s *StopSnapshot) string
	compare comparator
}

var stopFieldSpecs = []stopFieldSpec{
	{name: "name", value: func(s *StopSnapshot) string { return s.Name }, compare: compareName},
	{
		name:    "addressLine1",
		value:   func(s *StopSnapshot) string { return s.AddressLine1 },
		compare: compareIdentifier,
	},
	{name: "city", value: func(s *StopSnapshot) string { return s.City }, compare: compareIdentifier},
	{name: "state", value: func(s *StopSnapshot) string { return s.State }, compare: compareIdentifier},
	{
		name:    "postalCode",
		value:   func(s *StopSnapshot) string { return s.PostalCode },
		compare: comparePostalCode,
	},
	{name: "date", value: func(s *StopSnapshot) string { return s.Date }, compare: compareDate},
	{
		name:    "appointmentRequired",
		value:   func(s *StopSnapshot) string { return strconv.FormatBool(s.AppointmentRequired) },
		compare: compareIdentifier,
	},
}

func Score(p *Prediction, expected *Snapshot) []FieldResult {
	results := make([]FieldResult, 0, len(fieldSpecs)+len(stopFieldSpecs)*len(expected.Stops))

	for i := range fieldSpecs {
		spec := &fieldSpecs[i]
		predicted, alias := firstPredicted(p, spec.aliases)
		var stop *StopSnapshot
		if spec.stop != nil {
			stop = spec.stop(expected)
		}
		if result, ok := score(&scoreInput{
			key:       spec.key,
			predicted: predicted,
			confirmed: expected.Fields[spec.key],
			stop:      stop,
			compare:   spec.compare,
			meta:      p.FieldMeta[alias],
		}); ok {
			results = append(results, result)
		}
	}

	for _, role := range []string{RolePickup, RoleDelivery} {
		results = append(results, scoreStops(p, expected, role)...)
	}

	return results
}

func scoreStops(p *Prediction, expected *Snapshot, role string) []FieldResult {
	predictedIdx := stopIndexes(p.Snapshot.Stops, role)
	expectedIdx := stopIndexes(expected.Stops, role)

	pairs := max(len(predictedIdx), len(expectedIdx))
	results := make([]FieldResult, 0, pairs*len(stopFieldSpecs))
	for n := range pairs {
		var predicted *StopSnapshot
		var meta FieldMeta
		if n < len(predictedIdx) {
			predicted = &p.Snapshot.Stops[predictedIdx[n]]
			if predictedIdx[n] < len(p.StopMeta) {
				meta = p.StopMeta[predictedIdx[n]]
			}
		}
		var confirmed *StopSnapshot
		if n < len(expectedIdx) {
			confirmed = &expected.Stops[expectedIdx[n]]
		}

		for i := range stopFieldSpecs {
			spec := &stopFieldSpecs[i]
			predictedValue := ""
			if predicted != nil {
				predictedValue = spec.value(predicted)
			}
			confirmedValue := ""
			if confirmed != nil {
				confirmedValue = spec.value(confirmed)
			}
			if result, ok := score(&scoreInput{
				key:       fmt.Sprintf("stops.%s[%d].%s", role, n, spec.name),
				predicted: predictedValue,
				confirmed: confirmedValue,
				stop:      confirmed,
				compare:   spec.compare,
				meta:      meta,
			}); ok {
				results = append(results, result)
			}
		}
	}

	return results
}

func stopIndexes(stops []StopSnapshot, role string) []int {
	indexes := make([]int, 0, len(stops))
	for i := range stops {
		if stops[i].Role == role {
			indexes = append(indexes, i)
		}
	}

	return indexes
}

type scoreInput struct {
	key       string
	predicted string
	confirmed string
	stop      *StopSnapshot
	compare   comparator
	meta      FieldMeta
}

func score(in *scoreInput) (FieldResult, bool) {
	result := FieldResult{
		Key:        in.key,
		Predicted:  in.predicted,
		Confirmed:  in.confirmed,
		Source:     in.meta.Source,
		Confidence: in.meta.Confidence,
	}

	switch {
	case in.predicted == "" && in.confirmed == "":
		return result, false
	case in.predicted == "":
		result.Outcome = OutcomeMissed
	case in.confirmed == "":
		result.Outcome = OutcomeUnconfirmed
	default:
		switch in.compare(in.predicted, in.confirmed, in.stop) {
		case comparisonEqual:
			result.Outcome = OutcomeCorrect
		case comparisonDifferent:
			result.Outcome = OutcomeCorrected
		case comparisonUnreadable:
			result.Outcome = OutcomeUnscored
		}
	}

	return result, true
}

func firstPredicted(p *Prediction, aliases []string) (value, alias string) {
	for _, key := range aliases {
		if v := p.Snapshot.Fields[key]; v != "" {
			return v, key
		}
	}

	return "", ""
}

func compareIdentifier(predicted, confirmed string, _ *StopSnapshot) comparison {
	if stringutils.NormalizeIdentifier(predicted) == stringutils.NormalizeIdentifier(confirmed) {
		return comparisonEqual
	}

	return comparisonDifferent
}

func compareName(predicted, confirmed string, _ *StopSnapshot) comparison {
	p := stringutils.NormalizeIdentifier(predicted)
	c := stringutils.NormalizeIdentifier(confirmed)
	if p == c {
		return comparisonEqual
	}

	shorter, longer := p, c
	if len(shorter) > len(longer) {
		shorter, longer = longer, shorter
	}
	if len([]rune(shorter)) >= minContainedNameRunes && strings.Contains(longer, shorter) {
		return comparisonEqual
	}

	return comparisonDifferent
}

func compareMoney(predicted, confirmed string, _ *StopSnapshot) comparison {
	p, err := decimalutils.ParseMoneyText(predicted)
	if err != nil || !p.Valid {
		return comparisonUnreadable
	}
	c, err := decimalutils.ParseMoneyText(confirmed)
	if err != nil || !c.Valid {
		return comparisonUnreadable
	}
	if p.Decimal.Round(moneyPlaces).Equal(c.Decimal.Round(moneyPlaces)) {
		return comparisonEqual
	}

	return comparisonDifferent
}

func compareInteger(predicted, confirmed string, _ *StopSnapshot) comparison {
	p, ok := intutils.FirstInteger(predicted)
	if !ok {
		return comparisonUnreadable
	}
	c, ok := intutils.FirstInteger(confirmed)
	if !ok {
		return comparisonUnreadable
	}
	if p == c {
		return comparisonEqual
	}

	return comparisonDifferent
}

func comparePostalCode(predicted, confirmed string, _ *StopSnapshot) comparison {
	p := stringutils.DigitsOnly(predicted)
	c := stringutils.DigitsOnly(confirmed)
	if len(p) < postalCodeDigits || len(c) < postalCodeDigits {
		return compareIdentifier(predicted, confirmed, nil)
	}
	if p[:postalCodeDigits] == c[:postalCodeDigits] {
		return comparisonEqual
	}

	return comparisonDifferent
}

func compareDate(predicted, confirmed string, stop *StopSnapshot) comparison {
	day, ok := timeutils.FindDocumentDate(predicted)
	if !ok {
		return comparisonUnreadable
	}
	if stop == nil || stop.ScheduledWindowStart <= 0 {
		return compareCalendarDays(day, confirmed)
	}

	at := time.Unix(stop.ScheduledWindowStart, 0)
	if day.Matches(at.In(stop.Location())) || day.Matches(at.UTC()) {
		return comparisonEqual
	}

	return comparisonDifferent
}

func compareCalendarDays(predicted timeutils.CalendarDay, confirmed string) comparison {
	expected, ok := timeutils.FindDocumentDate(confirmed)
	if !ok || !expected.HasYear() {
		return comparisonUnreadable
	}
	if predicted.Matches(time.Date(expected.Year, expected.Month, expected.Day, 0, 0, 0, 0, time.UTC)) {
		return comparisonEqual
	}

	return comparisonDifferent
}
