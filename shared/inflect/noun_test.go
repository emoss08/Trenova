package inflect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPluralNoun(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"shipment":                "shipments",
		"formula template":        "formula templates",
		"Formula template":        "Formula templates",
		"clearinghouse query":     "clearinghouse queries",
		"fuel index":              "fuel indexes",
		"address":                 "addresses",
		"batch":                   "batches",
		"day":                     "days",
		"worker PTO":              "worker PTOs",
		"rate matrix":             "rate matrices",
		"person":                  "people",
		"accessorial_charge":      "accessorial_charges",
		"data":                    "data",
		"":                        "",
		"DOT random pool":         "DOT random pools",
		"billing queue item":      "billing queue items",
		"hazardous material":      "hazardous materials",
		"credit memo app":         "credit memo apps",
		"location category":       "location categories",
		"employment verification": "employment verifications",
	}
	for input, want := range cases {
		assert.Equal(t, want, PluralNoun(input), input)
	}
}

func TestSingularNoun(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"shipments":             "shipment",
		"formula_templates":     "formula_template",
		"clearinghouse queries": "clearinghouse query",
		"fuel indexes":          "fuel index",
		"addresses":             "address",
		"batches":               "batch",
		"worker PTOs":           "worker PTO",
		"rate matrices":         "rate matrix",
		"people":                "person",
		"status":                "status",
		"licenses":              "license",
		"fuel_index_prices":     "fuel_index_price",
		"analysis":              "analysis",
		"address":               "address",
		"":                      "",
	}
	for input, want := range cases {
		assert.Equal(t, want, SingularNoun(input), input)
	}
}
