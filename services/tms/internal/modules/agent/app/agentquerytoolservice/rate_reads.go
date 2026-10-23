package agentquerytoolservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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

	fieldPrice = "price"
)

// fuelSurchargeRates are the fields a program's current rate is given in;
// a row with none of them was priced outside its table.
var fuelSurchargeRates = []string{"ratePerMile", "percent", "flatAmount"}

// rateReadSpecs are the rate agreement register and the fuel surcharge reads,
// each through the operation its screen or the Desk sends.
func rateReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{
		{
			name:         toolRateListAgreements,
			entityPlural: "rate agreements",
			summary:      rateAgreementsSummary,
			queryNote:    rateAgreementQueryNote,
			resource:     permission.ResourceRateAgreement,
			references: map[string]idReference{
				paramCustomerID: {
					resource: permission.ResourceCustomer,
					what:     "The customer",
					source:   toolCustomerList,
				},
				paramCarrierID: {
					resource: permission.ResourceCarrier,
					what:     "The carrier",
					source:   toolCarrierList,
				},
			},
		},
		{
			name:         toolRateListFuelIndexPrices,
			entityPlural: "fuel index prices",
			summary:      fuelIndexPricesSummary,
			resource:     permission.ResourceFuelSurchargeProgram,
			references: map[string]idReference{paramFuelIndexID: {
				kinds:  []permission.RecordKind{permission.KindFuelIndex},
				what:   "The fuel index",
				source: toolRateListFuelSurcharge,
			}},
			defaults: listDefaults,
		},
		{
			name:         toolRateListFuelSurcharge,
			entityPlural: toolRateListFuelProgramsWord,
			summary:      fuelSurchargeProgramsSummary,
			searchTerms:  []string{"fuel program", "diesel price in use", "surcharge schedule"},
			queryNote:    fuelProgramQueryNote,
			matches:      []string{"program.code", "program.name"},
			resource:     permission.ResourceFuelSurchargeProgram,
			shape:        noteFuelSurchargeRate,
		},
	}
}

// noteFuelSurchargeRate says why a program has no surcharge this week, as
// the repository tool did, unless what it would be judged by was withheld.
func noteFuelSurchargeRate(row map[string]any, withheld map[string]struct{}, _ clock) {
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
			row[outcomeNote] = noteNoFuelPrice
		}
		return
	}
	for _, field := range fuelSurchargeRates {
		if _, ok := row[field]; ok || hidden(field) {
			return
		}
	}
	row[outcomeNote] = noteOutsideFuelTable
}

// rateAgreementGetSpec reads one rate agreement through the Desk's card.
func rateAgreementGetSpec() getSpec {
	return getSpec{
		name:        toolRateGetAgreement,
		entity:      "rate agreement",
		searchTerms: []string{"contract terms", "lane rules", "lane rates", "minimums", "rule inputs"},
		resource:    permission.ResourceRateAgreement,
		summary: "Retrieve one rate agreement, a contract or tariff, by id with its lane rules, " +
			"their tariffs and inputs (the first 50, and laneCount). It also gives the " +
			"agreement's own inputs, the current version, the accessorial schedule and the " +
			"fuel surcharge binding. rate_explain says how it priced one shipment. Take the " +
			"id from rate_list_agreements.",
		paramName: "rateAgreementId",
		idSource:  "from rate_list_agreements, rate_explain or the page you are on",
	}
}
