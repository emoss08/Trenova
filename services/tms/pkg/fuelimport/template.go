package fuelimport

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
)

var templateColumns = [...]Field{
	FieldPurchasedAt,
	FieldTractorCode,
	FieldDriverName,
	FieldCardLastFour,
	FieldVendor,
	FieldCity,
	FieldJurisdiction,
	FieldFuelType,
	FieldQuantity,
	FieldUnit,
	FieldUnitPrice,
	FieldTotalAmount,
	FieldCurrency,
	FieldOdometer,
	FieldTransactionReference,
}

var templateExample = map[Field]string{
	FieldPurchasedAt:          "2026-07-14 08:32",
	FieldTractorCode:          "TRC-001",
	FieldDriverName:           "J. Driver",
	FieldCardLastFour:         "1234",
	FieldVendor:               "Pilot Travel Center",
	FieldCity:                 "Amarillo",
	FieldJurisdiction:         "TX",
	FieldFuelType:             "Diesel",
	FieldQuantity:             "118.420",
	FieldUnit:                 "Gallon",
	FieldUnitPrice:            "3.899",
	FieldTotalAmount:          "461.72",
	FieldCurrency:             "USD",
	FieldOdometer:             "412885",
	FieldTransactionReference: "TX-20260714-000123",
}

func TemplateFileName(provider fuelpurchase.CardProvider) string {
	name := strings.ToLower(strings.TrimSpace(string(provider)))
	if name == "" || !provider.IsValid() {
		name = "generic"
	}
	return "fuel-purchases-" + name + "-template.csv"
}

func Template(provider fuelpurchase.CardProvider) string {
	preset := Preset(provider)
	headers := make([]string, 0, len(templateColumns))
	example := make([]string, 0, len(templateColumns))
	for _, field := range templateColumns {
		headers = append(headers, titleCase(preset[field][0]))
		example = append(example, templateExample[field])
	}
	return strings.Join(headers, ",") + "\n" + strings.Join(example, ",") + "\n"
}

func titleCase(value string) string {
	words := strings.Fields(value)
	for i, word := range words {
		if len(word) <= 3 && word != "st" && word != "ref" {
			words[i] = strings.ToUpper(word)
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}
