package ratequerytools

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/filtercatalog"
)

const (
	toolRateListAgreements       = "rate_list_agreements"
	toolRateGetAgreement         = "rate_get_agreement"
	toolRateListFuelIndexPrices  = "rate_list_fuel_index_prices"
	toolRateListFuelSurcharge    = "rate_list_fuel_surcharge_programs"
	toolRateListFuelProgramsWord = "fuel surcharge programs"

	rateAgreementsSummary = "List rate agreements, the contracts and tariffs that price freight " +
		"for a customer or pay a carrier. Each has its code, name, party, type, status and " +
		"effective dates. rate_get_agreement opens one with its lane rules."
	rateAgreementQueryNote = "Words to look for in the agreement's code or name."

	fuelIndexPricesSummary = "List the prices on file for one fuel index, newest first, with " +
		"the day each applies to and whether a person entered it or it was fetched. Only a " +
		"manual price on a custom index can be corrected, with rate_correct_fuel_index_price; " +
		"rate_record_fuel_index_price adds the next one."
	fuelSurchargeProgramsSummary = "Read this week's fuel surcharge rate for every active fuel " +
		"program. It gives the fuel index price and its date, and the surcharge as a rate per " +
		"mile, a percent of linehaul or a flat amount. It says when a fallback price was used " +
		"because this week's was missing."
	fuelProgramQueryNote = "Words to look for in the program's code or name. Omit for every " +
		"active program."
	noteNoFuelPrice = "No index price is on file for this week or the fallback window, so no " +
		"surcharge can be computed."
	noteOutsideFuelTable = "The index price falls outside every row of this program's table."

	fieldPrice       = "price"
	fieldRatePerMile = "ratePerMile"
	fieldFlatAmount  = "flatAmount"
)

// fuelSurchargeRates are the fields a program's current rate is given in;
// a row with none of them was priced outside its table.
var fuelSurchargeRates = []string{fieldRatePerMile, querytool.FieldPercent, fieldFlatAmount}

// ReadSpecs are the rate agreement register and the fuel surcharge reads,
// each through the operation its screen or the Desk sends.
func ReadSpecs() []readspec.List {
	return []readspec.List{
		{
			Name:         toolRateListAgreements,
			EntityPlural: "rate agreements",
			Summary:      rateAgreementsSummary,
			QueryNote:    rateAgreementQueryNote,
			Resource:     permission.ResourceRateAgreement,
			References: map[string]readspec.Reference{
				querytool.ParamCustomerID: {
					Resource: permission.ResourceCustomer,
					What:     "The customer",
					Source:   querytool.ToolCustomerList,
				},
				querytool.ParamCarrierID: {
					Resource: permission.ResourceCarrier,
					What:     "The carrier",
					Source:   querytool.ToolCarrierList,
				},
			},
		},
		{
			Name:         toolRateListFuelIndexPrices,
			EntityPlural: "fuel index prices",
			Summary:      fuelIndexPricesSummary,
			Resource:     permission.ResourceFuelSurchargeProgram,
			References: map[string]readspec.Reference{querytool.ParamFuelIndexID: {
				Kinds:  []permission.RecordKind{permission.KindFuelIndex},
				What:   "The fuel index",
				Source: toolRateListFuelSurcharge,
			}},
			Defaults: querytool.ListDefaults,
		},
		{
			Name:         toolRateListFuelSurcharge,
			EntityPlural: toolRateListFuelProgramsWord,
			Summary:      fuelSurchargeProgramsSummary,
			SearchTerms:  []string{"fuel program", "diesel price in use", "surcharge schedule"},
			QueryNote:    fuelProgramQueryNote,
			Matches:      []string{"program.code", "program.name"},
			Resource:     permission.ResourceFuelSurchargeProgram,
			Shape:        noteFuelSurchargeRate,
		},
	}
}

// noteFuelSurchargeRate says why a program has no surcharge this week, as
// the repository tool did, unless what it would be judged by was withheld.
func noteFuelSurchargeRate(
	row map[string]any,
	withheld map[string]struct{},
	_ filtercatalog.Clock,
) {
	hidden := func(field string) bool {
		for path := range withheld {
			if path == field || strings.HasPrefix(path, field+".") {
				return true
			}
		}
		return false
	}
	if _, priced := row[fieldPrice]; !priced {
		if !hidden(fieldPrice) {
			row[graphqlread.OutcomeNote] = noteNoFuelPrice
		}
		return
	}
	for _, field := range fuelSurchargeRates {
		if _, ok := row[field]; ok || hidden(field) {
			return
		}
	}
	row[graphqlread.OutcomeNote] = noteOutsideFuelTable
}

// RateAgreementGetSpec reads one rate agreement through the Desk's card.
func RateAgreementGetSpec() readspec.Get {
	return readspec.Get{
		Name:   toolRateGetAgreement,
		Entity: "rate agreement",
		SearchTerms: []string{
			"contract terms",
			"lane rules",
			"lane rates",
			"minimums",
			"rule inputs",
		},
		Resource: permission.ResourceRateAgreement,
		Summary: "Retrieve one rate agreement, a contract or tariff, by id with its lane rules, " +
			"their tariffs and inputs (the first 50, and laneCount). It also gives the " +
			"agreement's own inputs, the current version, the accessorial schedule and the " +
			"fuel surcharge binding. rate_explain says how it priced one shipment. Take the " +
			"id from rate_list_agreements.",
		ParamName: "rateAgreementId",
		IDSource:  "from rate_list_agreements, rate_explain or the page you are on",
	}
}
