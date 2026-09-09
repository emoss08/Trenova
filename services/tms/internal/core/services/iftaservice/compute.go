package iftaservice

import (
	"sort"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	MPGScale     = 2
	GallonsScale = 3
	milesScale   = ifta.MilesScale
	wholeScale   = 0
	moneyScale   = 2
)

type TractorProfile struct {
	FuelType      domaintypes.IFTAFuelType
	IFTAQualified bool
}

func (p TractorProfile) MilesEligible() bool {
	return p.IFTAQualified && p.FuelType.CountsForIFTA()
}

type ComputeInput struct {
	Period        ifta.Period
	Loc           *time.Location
	Jurisdictions map[pulid.ID]*ifta.Jurisdiction
	Tractors      map[pulid.ID]TractorProfile
	Miles         *repositories.MileAccumulation
	Fuel          []*repositories.FuelAccumulationRow
	Rates         map[ifta.RateKey]*ifta.TaxRate
}

type ReturnTotals struct {
	TotalMiles          decimal.Decimal
	TotalTaxableMiles   decimal.Decimal
	TotalGallons        decimal.Decimal
	TotalTaxPaidGallons decimal.Decimal
	NetTaxableGallons   decimal.Decimal
	TaxDueMinor         int64
	SurchargeDueMinor   int64
	NetDueMinor         int64
}

type ComputeResult struct {
	Lines               []*ifta.ReturnLine
	Totals              ReturnTotals
	FleetMPG            []ifta.FleetMPG
	Problems            []ifta.Problem
	Unattributed        repositories.MoveAggregate
	NoTractor           repositories.MoveAggregate
	MismatchCount       int
	NonQualifiedMiles   decimal.Decimal
	NonQualifiedGallons decimal.Decimal
}

type lineKey struct {
	JurisdictionID pulid.ID
	FuelType       domaintypes.IFTAFuelType
}

type lineAccumulator struct {
	jurisdiction   *ifta.Jurisdiction
	fuelType       domaintypes.IFTAFuelType
	routeMiles     decimal.Decimal
	manualMiles    decimal.Decimal
	loadedMiles    decimal.Decimal
	emptyMiles     decimal.Decimal
	taxPaidGallons decimal.Decimal
	purchaseCount  int
}

func (a *lineAccumulator) totalMiles() decimal.Decimal {
	return a.routeMiles.Add(a.manualMiles)
}

type fuelTotals struct {
	miles   decimal.Decimal
	gallons decimal.Decimal
}

type computation struct {
	in                   ComputeInput
	lines                map[lineKey]*lineAccumulator
	byFuelType           map[domaintypes.IFTAFuelType]*fuelTotals
	unknownJurisdictions map[string]decimal.Decimal
	nonQualifiedMiles    decimal.Decimal
	nonQualifiedGallons  decimal.Decimal
}

func Compute(in ComputeInput) ComputeResult {
	c := &computation{
		in:                   in,
		lines:                make(map[lineKey]*lineAccumulator, 32),
		byFuelType:           make(map[domaintypes.IFTAFuelType]*fuelTotals, 2),
		unknownJurisdictions: make(map[string]decimal.Decimal),
	}
	if in.Miles != nil {
		c.addMiles(in.Miles.RouteRows, true)
		c.addMiles(in.Miles.ManualRows, false)
	}
	c.addFuel(in.Fuel)
	return c.finish()
}

func (c *computation) line(j *ifta.Jurisdiction, fuelType domaintypes.IFTAFuelType) *lineAccumulator {
	key := lineKey{JurisdictionID: j.ID, FuelType: fuelType}
	acc, ok := c.lines[key]
	if !ok {
		acc = &lineAccumulator{jurisdiction: j, fuelType: fuelType}
		c.lines[key] = acc
	}
	return acc
}

func (c *computation) totals(fuelType domaintypes.IFTAFuelType) *fuelTotals {
	t, ok := c.byFuelType[fuelType]
	if !ok {
		t = &fuelTotals{}
		c.byFuelType[fuelType] = t
	}
	return t
}

func (c *computation) addMiles(rows []*repositories.MileRow, route bool) {
	for _, row := range rows {
		if row == nil || !row.HasTractor() || !row.Miles.IsPositive() {
			continue
		}
		profile, ok := c.in.Tractors[row.TractorID]
		if !ok || !profile.MilesEligible() {
			c.nonQualifiedMiles = c.nonQualifiedMiles.Add(row.Miles)
			continue
		}
		j := c.in.Jurisdictions[row.JurisdictionID]
		if !row.HasJurisdiction() || j == nil {
			key := row.Key()
			c.unknownJurisdictions[key] = c.unknownJurisdictions[key].Add(row.Miles)
			continue
		}

		acc := c.line(j, profile.FuelType)
		if route {
			acc.routeMiles = acc.routeMiles.Add(row.Miles)
		} else {
			acc.manualMiles = acc.manualMiles.Add(row.Miles)
		}
		acc.loadedMiles = acc.loadedMiles.Add(row.LoadedMiles)
		acc.emptyMiles = acc.emptyMiles.Add(row.EmptyMiles)

		t := c.totals(profile.FuelType)
		t.miles = t.miles.Add(row.Miles)
	}
}

func (c *computation) addFuel(rows []*repositories.FuelAccumulationRow) {
	for _, row := range rows {
		if row == nil || !row.FuelType.CountsForIFTA() || !row.Gallons.IsPositive() {
			continue
		}
		profile, ok := c.in.Tractors[row.TractorID]
		if !ok || !profile.IFTAQualified {
			c.nonQualifiedGallons = c.nonQualifiedGallons.Add(row.Gallons)
			continue
		}

		t := c.totals(row.FuelType)
		t.gallons = t.gallons.Add(row.Gallons)

		j := c.in.Jurisdictions[row.JurisdictionID]
		if j == nil {
			continue
		}
		acc := c.line(j, row.FuelType)
		acc.purchaseCount += row.PurchaseCount
		acc.taxPaidGallons = acc.taxPaidGallons.Add(row.TaxPaidGallons)
	}
}

func (c *computation) sortedFuelTypes() []domaintypes.IFTAFuelType {
	out := make([]domaintypes.IFTAFuelType, 0, len(c.byFuelType))
	for fuelType := range c.byFuelType {
		out = append(out, fuelType)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (c *computation) sortedLines() []*lineAccumulator {
	out := make([]*lineAccumulator, 0, len(c.lines))
	for _, acc := range c.lines {
		out = append(out, acc)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.fuelType != b.fuelType {
			return a.fuelType < b.fuelType
		}
		if a.jurisdiction.CountryCode != b.jurisdiction.CountryCode {
			return a.jurisdiction.CountryCode < b.jurisdiction.CountryCode
		}
		return a.jurisdiction.Code < b.jurisdiction.Code
	})
	return out
}

func (c *computation) fleetMPG() (map[domaintypes.IFTAFuelType]decimal.Decimal, []ifta.FleetMPG, []ifta.Problem) {
	fuelTypes := c.sortedFuelTypes()
	mpgs := make(map[domaintypes.IFTAFuelType]decimal.Decimal, len(fuelTypes))
	entries := make([]ifta.FleetMPG, 0, len(fuelTypes))
	problems := make([]ifta.Problem, 0)

	for _, fuelType := range fuelTypes {
		t := c.byFuelType[fuelType]
		entry := ifta.FleetMPG{
			FuelType:     fuelType,
			TotalMiles:   t.miles.StringFixed(milesScale),
			TotalGallons: t.gallons.StringFixed(GallonsScale),
		}
		switch {
		case t.gallons.IsPositive() && t.miles.IsPositive():
			mpg := t.miles.Div(t.gallons).Round(MPGScale)
			mpgs[fuelType] = mpg
			entry.MPG = mpg.StringFixed(MPGScale)
		case t.miles.IsPositive():
			problems = append(problems, ifta.Problem{
				Code: ifta.ProblemNoFuelForType,
				Message: "No " + fuelType.Label() + " was purchased this quarter, so " +
					"fleet MPG cannot be computed and taxable gallons are zero",
				FuelType: fuelType,
				Amount:   t.miles.StringFixed(milesScale),
			})
		default:
			problems = append(problems, ifta.Problem{
				Code: ifta.ProblemNoFuelForType,
				Message: fuelType.Label() + " was purchased but no miles were driven on " +
					fuelType.Label() + " tractors this quarter",
				FuelType: fuelType,
				Amount:   t.gallons.StringFixed(GallonsScale),
			})
		}
		entries = append(entries, entry)
	}

	return mpgs, entries, problems
}

func (c *computation) buildLine(
	acc *lineAccumulator,
	mpg decimal.Decimal,
	hasMPG bool,
) (*ifta.ReturnLine, []ifta.Problem) {
	j := acc.jurisdiction
	problems := make([]ifta.Problem, 0, 2)
	totalMiles := acc.totalMiles().Round(wholeScale)

	line := &ifta.ReturnLine{
		JurisdictionID:    j.ID,
		FuelType:          acc.fuelType,
		IsIftaMember:      j.IsIftaMember,
		TotalMiles:        totalMiles,
		RouteMiles:        acc.routeMiles.Round(milesScale),
		ManualMiles:       acc.manualMiles.Round(milesScale),
		LoadedMiles:       acc.loadedMiles.Round(milesScale),
		EmptyMiles:        acc.emptyMiles.Round(milesScale),
		TaxPaidGallonsRaw: acc.taxPaidGallons.Round(GallonsScale),
		PurchaseCount:     acc.purchaseCount,
		TaxableMiles:      decimal.Zero,
		TaxPaidGallons:    decimal.Zero,
		TaxableGallons:    decimal.Zero,
		NetTaxableGallons: decimal.Zero,
	}

	if !j.IsIftaMember {
		line.RatePerGallon = decimal.NewNullDecimal(decimal.Zero)
		problems = append(problems, ifta.Problem{
			Code: ifta.ProblemNonMemberActivity,
			Message: j.Label() + " is not an IFTA member; its " +
				totalMiles.String() + " miles count toward fleet MPG but carry no tax",
			JurisdictionCode: j.Key(),
			FuelType:         acc.fuelType,
			Amount:           totalMiles.StringFixed(milesScale),
		})
		return line, problems
	}

	line.TaxableMiles = totalMiles
	line.TaxPaidGallons = acc.taxPaidGallons.Round(wholeScale)
	if hasMPG && mpg.IsPositive() {
		line.TaxableGallons = line.TaxableMiles.Div(mpg).Round(wholeScale)
	}
	line.NetTaxableGallons = line.TaxableGallons.Sub(line.TaxPaidGallons)

	rate := c.in.Rates[ifta.RateKey{JurisdictionID: j.ID, FuelType: acc.fuelType}]
	if rate == nil {
		line.RateMissing = true
		problems = append(problems, ifta.Problem{
			Code: ifta.ProblemMissingRate,
			Message: "No " + acc.fuelType.Label() + " tax rate is on file for " + j.Label() +
				" in " + c.in.Period.Label(),
			JurisdictionCode: j.Key(),
			FuelType:         acc.fuelType,
		})
		return line, problems
	}

	line.RatePerGallon = decimal.NewNullDecimal(rate.RatePerGallon)
	line.TaxDueMinor = money.MinorUnits(line.NetTaxableGallons.Mul(rate.RatePerGallon).Round(moneyScale))

	switch {
	case rate.HasSurcharge():
		line.SurchargeRatePerGallon = rate.SurchargeRatePerGallon
		surcharge := line.TaxableGallons.Mul(rate.SurchargeRate()).Round(moneyScale)
		if surcharge.IsNegative() {
			surcharge = decimal.Zero
		}
		line.SurchargeDueMinor = money.MinorUnits(surcharge)
	case j.HasSurcharge:
		problems = append(problems, ifta.Problem{
			Code: ifta.ProblemMissingRate,
			Message: j.Label() + " charges a surcharge but no " + acc.fuelType.Label() +
				" surcharge rate is on file for " + c.in.Period.Label(),
			JurisdictionCode: j.Key(),
			FuelType:         acc.fuelType,
		})
	}

	line.LineTotalMinor = line.TaxDueMinor + line.SurchargeDueMinor

	return line, problems
}

func (c *computation) finish() ComputeResult {
	mpgs, fleet, fuelProblems := c.fleetMPG()
	sorted := c.sortedLines()

	result := ComputeResult{
		Lines:               make([]*ifta.ReturnLine, 0, len(sorted)),
		FleetMPG:            fleet,
		NonQualifiedMiles:   c.nonQualifiedMiles,
		NonQualifiedGallons: c.nonQualifiedGallons,
	}
	if c.in.Miles != nil {
		result.Unattributed = c.in.Miles.Unattributed
		result.NoTractor = c.in.Miles.NoTractor
		result.MismatchCount = len(c.in.Miles.Mismatches)
	}

	missingRate := make([]ifta.Problem, 0)
	nonMember := make([]ifta.Problem, 0)
	for i, acc := range sorted {
		mpg, hasMPG := mpgs[acc.fuelType]
		line, lineProblems := c.buildLine(acc, mpg, hasMPG)
		line.SortOrder = i
		result.Lines = append(result.Lines, line)
		for _, p := range lineProblems {
			if p.Code == ifta.ProblemMissingRate {
				missingRate = append(missingRate, p)
			} else {
				nonMember = append(nonMember, p)
			}
		}

		result.Totals.TotalMiles = result.Totals.TotalMiles.Add(line.TotalMiles)
		if !line.IsIftaMember {
			continue
		}
		result.Totals.TotalTaxableMiles = result.Totals.TotalTaxableMiles.Add(line.TaxableMiles)
		result.Totals.TotalTaxPaidGallons = result.Totals.TotalTaxPaidGallons.Add(line.TaxPaidGallons)
		result.Totals.NetTaxableGallons = result.Totals.NetTaxableGallons.Add(line.NetTaxableGallons)
		result.Totals.TaxDueMinor += line.TaxDueMinor
		result.Totals.SurchargeDueMinor += line.SurchargeDueMinor
	}
	for _, t := range c.byFuelType {
		result.Totals.TotalGallons = result.Totals.TotalGallons.Add(t.gallons)
	}
	result.Totals.NetDueMinor = result.Totals.TaxDueMinor + result.Totals.SurchargeDueMinor

	problems := make([]ifta.Problem, 0, len(missingRate)+len(fuelProblems)+len(nonMember)+8)
	problems = append(problems, missingRate...)
	problems = append(problems, fuelProblems...)
	problems = append(problems, c.diagnosticProblems(result)...)
	problems = append(problems, nonMember...)
	problems = append(problems, c.unknownJurisdictionProblems()...)
	if c.nonQualifiedMiles.IsPositive() || c.nonQualifiedGallons.IsPositive() {
		problems = append(problems, ifta.Problem{
			Code: ifta.ProblemNonQualifiedActivity,
			Message: c.nonQualifiedMiles.StringFixed(milesScale) + " miles and " +
				c.nonQualifiedGallons.StringFixed(GallonsScale) +
				" gallons on tractors that are not IFTA qualified were left off the return",
			Amount: c.nonQualifiedMiles.StringFixed(milesScale),
		})
	}
	result.Problems = problems

	return result
}

func (c *computation) diagnosticProblems(result ComputeResult) []ifta.Problem {
	problems := make([]ifta.Problem, 0, 3)
	if result.Unattributed.MoveCount > 0 {
		problems = append(problems, ifta.Problem{
			Code: ifta.ProblemUnattributedMiles,
			Message: strconv.Itoa(result.Unattributed.MoveCount) + " completed moves totalling " +
				result.Unattributed.Miles.StringFixed(milesScale) +
				" miles have no jurisdiction breakdown and are not on the return",
			Amount: result.Unattributed.Miles.StringFixed(milesScale),
		})
	}
	if result.NoTractor.MoveCount > 0 {
		problems = append(problems, ifta.Problem{
			Code: ifta.ProblemNoTractorMiles,
			Message: strconv.Itoa(result.NoTractor.MoveCount) + " completed moves totalling " +
				result.NoTractor.Miles.StringFixed(milesScale) +
				" miles have no tractor assigned and cannot be attributed to a fuel type",
			Amount: result.NoTractor.Miles.StringFixed(milesScale),
		})
	}
	if result.MismatchCount > 0 {
		problems = append(problems, ifta.Problem{
			Code: ifta.ProblemMileageMismatch,
			Message: strconv.Itoa(result.MismatchCount) +
				" moves have jurisdiction miles that do not add up to the move distance",
			Amount: strconv.Itoa(result.MismatchCount),
		})
	}
	return problems
}

func (c *computation) unknownJurisdictionProblems() []ifta.Problem {
	if len(c.unknownJurisdictions) == 0 {
		return nil
	}
	keys := make([]string, 0, len(c.unknownJurisdictions))
	for key := range c.unknownJurisdictions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	problems := make([]ifta.Problem, 0, len(keys))
	for _, key := range keys {
		miles := c.unknownJurisdictions[key]
		problems = append(problems, ifta.Problem{
			Code: ifta.ProblemNonMemberActivity,
			Message: miles.StringFixed(milesScale) + " miles were routed through " + key +
				", which is not a known IFTA jurisdiction, and are not on the return",
			JurisdictionCode: key,
			Amount:           miles.StringFixed(milesScale),
		})
	}
	return problems
}
