package aicorrectionservice

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
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

type comparator func(predicted, confirmed string, stop *confirmedStop) comparison

type fieldSpec struct {
	key     string
	aliases []string
	compare comparator
	stop    func(c *confirmation) *confirmedStop
}

var fieldSpecs = []fieldSpec{
	{
		key:     confirmedReference,
		aliases: []string{"referenceNumber", "bol", "reference", "loadNumber"},
		compare: compareIdentifier,
	},
	{key: confirmedRate, aliases: []string{"rate"}, compare: compareMoney},
	{key: confirmedWeight, aliases: []string{"weight"}, compare: compareInteger},
	{key: confirmedPieces, aliases: []string{"pieceCount", "pieces"}, compare: compareInteger},
	{key: confirmedCommodity, aliases: []string{"commodity"}, compare: compareName},
	{key: confirmedShipper, aliases: []string{"shipper"}, compare: compareName},
	{key: confirmedConsignee, aliases: []string{"consignee"}, compare: compareName},
	{
		key:     confirmedPickup,
		aliases: []string{"pickupWindow"},
		compare: compareDate,
		stop:    func(c *confirmation) *confirmedStop { return firstStop(c.stops, rolePickup) },
	},
	{
		key:     confirmedDelivery,
		aliases: []string{"deliveryWindow"},
		compare: compareDate,
		stop:    func(c *confirmation) *confirmedStop { return lastStop(c.stops, roleDelivery) },
	},
}

type stopFieldSpec struct {
	name    string
	value   func(s *aicorrection.StopSnapshot) string
	compare comparator
}

var stopFieldSpecs = []stopFieldSpec{
	{name: "name", value: func(s *aicorrection.StopSnapshot) string { return s.Name }, compare: compareName},
	{
		name:    "addressLine1",
		value:   func(s *aicorrection.StopSnapshot) string { return s.AddressLine1 },
		compare: compareIdentifier,
	},
	{name: "city", value: func(s *aicorrection.StopSnapshot) string { return s.City }, compare: compareIdentifier},
	{name: "state", value: func(s *aicorrection.StopSnapshot) string { return s.State }, compare: compareIdentifier},
	{
		name:    "postalCode",
		value:   func(s *aicorrection.StopSnapshot) string { return s.PostalCode },
		compare: comparePostalCode,
	},
	{name: "date", value: func(s *aicorrection.StopSnapshot) string { return s.Date }, compare: compareDate},
	{
		name:    "appointmentRequired",
		value:   func(s *aicorrection.StopSnapshot) string { return strconv.FormatBool(s.AppointmentRequired) },
		compare: compareIdentifier,
	},
}

func compareSnapshots(p *prediction, c *confirmation) []aicorrection.FieldResult {
	results := make([]aicorrection.FieldResult, 0, len(fieldSpecs)+len(stopFieldSpecs)*len(c.stops))

	for i := range fieldSpecs {
		spec := &fieldSpecs[i]
		predicted, alias := firstPredicted(p, spec.aliases)
		var stop *confirmedStop
		if spec.stop != nil {
			stop = spec.stop(c)
		}
		if result, ok := score(&scoreInput{
			key:       spec.key,
			predicted: predicted,
			confirmed: c.snapshot.Fields[spec.key],
			stop:      stop,
			compare:   spec.compare,
			meta:      p.fieldMeta[alias],
		}); ok {
			results = append(results, result)
		}
	}

	for _, role := range []string{rolePickup, roleDelivery} {
		results = append(results, compareStops(p, c, role)...)
	}

	return results
}

func compareStops(p *prediction, c *confirmation, role string) []aicorrection.FieldResult {
	predictedIdx := make([]int, 0, len(p.snapshot.Stops))
	for i := range p.snapshot.Stops {
		if p.snapshot.Stops[i].Role == role {
			predictedIdx = append(predictedIdx, i)
		}
	}
	confirmedIdx := make([]int, 0, len(c.stops))
	for i := range c.stops {
		if c.stops[i].snapshot.Role == role {
			confirmedIdx = append(confirmedIdx, i)
		}
	}

	pairs := max(len(predictedIdx), len(confirmedIdx))
	results := make([]aicorrection.FieldResult, 0, pairs*len(stopFieldSpecs))
	for n := range pairs {
		var predicted *aicorrection.StopSnapshot
		var meta fieldMeta
		if n < len(predictedIdx) {
			predicted = &p.snapshot.Stops[predictedIdx[n]]
			meta = p.stopMeta[predictedIdx[n]]
		}
		var confirmed *confirmedStop
		if n < len(confirmedIdx) {
			confirmed = &c.stops[confirmedIdx[n]]
		}

		for i := range stopFieldSpecs {
			spec := &stopFieldSpecs[i]
			predictedValue := ""
			if predicted != nil {
				predictedValue = spec.value(predicted)
			}
			confirmedValue := ""
			if confirmed != nil {
				confirmedValue = spec.value(&confirmed.snapshot)
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

type scoreInput struct {
	key       string
	predicted string
	confirmed string
	stop      *confirmedStop
	compare   comparator
	meta      fieldMeta
}

func score(in *scoreInput) (aicorrection.FieldResult, bool) {
	result := aicorrection.FieldResult{
		Key:        in.key,
		Predicted:  in.predicted,
		Confirmed:  in.confirmed,
		Source:     in.meta.source,
		Confidence: in.meta.confidence,
	}

	switch {
	case in.predicted == "" && in.confirmed == "":
		return result, false
	case in.predicted == "":
		result.Outcome = aicorrection.OutcomeMissed
	case in.confirmed == "":
		result.Outcome = aicorrection.OutcomeUnconfirmed
	default:
		switch in.compare(in.predicted, in.confirmed, in.stop) {
		case comparisonEqual:
			result.Outcome = aicorrection.OutcomeCorrect
		case comparisonDifferent:
			result.Outcome = aicorrection.OutcomeCorrected
		case comparisonUnreadable:
			result.Outcome = aicorrection.OutcomeUnscored
		}
	}

	return result, true
}

func firstPredicted(p *prediction, aliases []string) (value, alias string) {
	for _, key := range aliases {
		if v := p.snapshot.Fields[key]; v != "" {
			return v, key
		}
	}

	return "", ""
}

func compareIdentifier(predicted, confirmed string, _ *confirmedStop) comparison {
	if stringutils.NormalizeIdentifier(predicted) == stringutils.NormalizeIdentifier(confirmed) {
		return comparisonEqual
	}

	return comparisonDifferent
}

func compareName(predicted, confirmed string, _ *confirmedStop) comparison {
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

func compareMoney(predicted, confirmed string, _ *confirmedStop) comparison {
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

func compareInteger(predicted, confirmed string, _ *confirmedStop) comparison {
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

func comparePostalCode(predicted, confirmed string, _ *confirmedStop) comparison {
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

func compareDate(predicted, _ string, stop *confirmedStop) comparison {
	day, ok := timeutils.FindDocumentDate(predicted)
	if !ok || stop == nil || stop.snapshot.ScheduledWindowStart <= 0 {
		return comparisonUnreadable
	}

	at := time.Unix(stop.snapshot.ScheduledWindowStart, 0)
	if day.Matches(at.In(stop.location)) || day.Matches(at.UTC()) {
		return comparisonEqual
	}

	return comparisonDifferent
}
