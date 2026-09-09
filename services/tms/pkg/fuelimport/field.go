package fuelimport

import (
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
)

type Field string

const (
	FieldPurchasedAt          = Field("purchasedAt")
	FieldVendor               = Field("vendor")
	FieldCity                 = Field("city")
	FieldJurisdiction         = Field("jurisdiction")
	FieldFuelType             = Field("fuelType")
	FieldQuantity             = Field("quantity")
	FieldUnit                 = Field("unit")
	FieldUnitPrice            = Field("unitPrice")
	FieldTotalAmount          = Field("totalAmount")
	FieldCurrency             = Field("currency")
	FieldOdometer             = Field("odometer")
	FieldCardLastFour         = Field("cardLastFour")
	FieldTransactionReference = Field("transactionReference")
	FieldTractorCode          = Field("tractorCode")
	FieldLicensePlate         = Field("licensePlate")
	FieldDriverName           = Field("driverName")
)

func (f Field) String() string { return string(f) }

func (f Field) IsValid() bool {
	for _, known := range Fields() {
		if known == f {
			return true
		}
	}
	return false
}

func (f Field) Label() string {
	switch f {
	case FieldPurchasedAt:
		return "Transaction date"
	case FieldVendor:
		return "Vendor"
	case FieldCity:
		return "City"
	case FieldJurisdiction:
		return "State or province"
	case FieldFuelType:
		return "Fuel product"
	case FieldQuantity:
		return "Quantity"
	case FieldUnit:
		return "Quantity unit"
	case FieldUnitPrice:
		return "Unit price"
	case FieldTotalAmount:
		return "Total amount"
	case FieldCurrency:
		return "Currency"
	case FieldOdometer:
		return "Odometer"
	case FieldCardLastFour:
		return "Card"
	case FieldTransactionReference:
		return "Transaction reference"
	case FieldTractorCode:
		return "Unit number"
	case FieldLicensePlate:
		return "License plate"
	case FieldDriverName:
		return "Driver"
	default:
		return string(f)
	}
}

func Fields() []Field {
	return []Field{
		FieldPurchasedAt,
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
		FieldCardLastFour,
		FieldTransactionReference,
		FieldTractorCode,
		FieldLicensePlate,
		FieldDriverName,
	}
}

func RequiredFields() []Field {
	return []Field{FieldPurchasedAt, FieldQuantity, FieldTotalAmount, FieldJurisdiction}
}

type Mapping map[Field]int

func MappingFromStrings(raw map[string]int) (Mapping, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	mapping := make(Mapping, len(raw))
	for name, index := range raw {
		field := Field(name)
		if !field.IsValid() {
			return nil, errors.New("unknown import field " + name)
		}
		if index < 0 {
			return nil, errors.New("column index for " + name + " cannot be negative")
		}
		mapping[field] = index
	}
	return mapping, nil
}

func (m Mapping) Strings() map[string]int {
	out := make(map[string]int, len(m))
	for field, index := range m {
		out[string(field)] = index
	}
	return out
}

func (m Mapping) Has(field Field) bool {
	_, ok := m[field]
	return ok
}

type Problem struct {
	Field   Field
	Message string
}

func (p Problem) Error() string { return p.Message }

func Validate(mapping Mapping, hasDefaultFuelType bool) []Problem {
	problems := make([]Problem, 0, 5)
	for _, field := range RequiredFields() {
		if !mapping.Has(field) {
			problems = append(problems, Problem{
				Field:   field,
				Message: "No column was found for " + strings.ToLower(field.Label()),
			})
		}
	}
	if !mapping.Has(FieldFuelType) && !hasDefaultFuelType {
		problems = append(problems, Problem{
			Field: FieldFuelType,
			Message: "No column was found for fuel product and the import has no " +
				"default fuel type",
		})
	}

	seen := make(map[int]Field, len(mapping))
	for _, field := range Fields() {
		index, ok := mapping[field]
		if !ok {
			continue
		}
		if other, dup := seen[index]; dup {
			problems = append(problems, Problem{
				Field: field,
				Message: "Column " + strings.ToLower(field.Label()) + " and " +
					strings.ToLower(other.Label()) + " point at the same column",
			})
			continue
		}
		seen[index] = field
	}

	return problems
}

var genericSynonyms = map[Field][]string{
	FieldPurchasedAt: {
		"transaction date", "trans date", "tran date", "date", "purchase date",
		"fuel date", "transaction date time", "date time", "trans date time",
		"transaction datetime", "sale date", "activity date",
	},
	FieldVendor: {
		"merchant", "merchant name", "vendor", "vendor name", "location name",
		"truck stop", "site name", "station", "location", "merchant location",
		"site", "supplier",
	},
	FieldCity: {
		"city", "merchant city", "location city", "site city", "vendor city",
	},
	FieldJurisdiction: {
		"state", "st", "merchant state", "state province", "state prov",
		"prov", "province", "jurisdiction", "location state", "site state",
		"vendor state", "state or province", "merchant st",
	},
	FieldFuelType: {
		"product", "product description", "fuel type", "product code", "item",
		"commodity", "product type", "fuel product", "product name", "fuel",
		"item description",
	},
	FieldQuantity: {
		"quantity", "qty", "gallons", "gals", "units", "volume", "litres",
		"liters", "fuel quantity", "fuel gallons", "qty gallons", "gallons purchased",
		"net gallons", "fuel qty", "quantity purchased",
	},
	FieldUnit: {
		"uom", "unit of measure", "quantity unit", "qty unit", "measure",
		"unit measure", "volume unit",
	},
	FieldUnitPrice: {
		"unit price", "price per gallon", "ppg", "price", "unit cost", "pump price",
		"price gal", "retail ppg", "price per unit", "fuel price", "cost per gallon",
	},
	FieldTotalAmount: {
		"amount", "total", "total amount", "fuel amount", "net amount",
		"extended amount", "total cost", "gross amount", "invoice amount",
		"fuel total", "total fuel amount", "fuel cost", "transaction amount",
		"line total",
	},
	FieldCurrency: {
		"currency", "curr", "currency code", "ccy",
	},
	FieldOdometer: {
		"odometer", "odom", "hub reading", "hubometer", "mileage", "unit odometer",
		"odometer reading", "odo",
	},
	FieldCardLastFour: {
		"card", "card number", "card no", "card num", "card last 4", "last four",
		"last 4", "card last four", "masked card", "card id", "fuel card",
		"card ending", "account number",
	},
	FieldTransactionReference: {
		"transaction id", "transaction number", "trans id", "trans number",
		"transaction", "reference", "ref", "reference number", "invoice number",
		"invoice", "invoice no", "auth code", "authorization", "authorization code",
		"auth number", "receipt number", "receipt", "ticket number", "ticket",
		"transaction ref", "trans ref", "tran id", "confirmation number",
	},
	FieldTractorCode: {
		"unit", "unit number", "unit no", "unit num", "unit id", "truck", "truck number",
		"truck no", "truck id", "tractor", "tractor number", "tractor id", "vehicle",
		"vehicle number", "vehicle id", "vehicle no", "equipment", "equipment number",
		"equipment id", "asset", "asset id", "asset number", "power unit",
	},
	FieldLicensePlate: {
		"plate", "license plate", "licence plate", "plate number", "tag", "vehicle plate",
		"license", "licence", "registration",
	},
	FieldDriverName: {
		"driver", "driver name", "cardholder", "card holder", "cardholder name",
		"employee", "employee name", "driver id", "driver number",
	},
}

var providerSynonyms = map[fuelpurchase.CardProvider]map[Field][]string{
	fuelpurchase.CardProviderComdata: {
		FieldPurchasedAt:          {"trans date", "transaction date"},
		FieldVendor:               {"location name", "merchant name"},
		FieldCity:                 {"city", "location city"},
		FieldJurisdiction:         {"state", "location state"},
		FieldFuelType:             {"product", "product description"},
		FieldQuantity:             {"quantity", "gallons"},
		FieldUnitPrice:            {"unit price", "ppg"},
		FieldTotalAmount:          {"total amount", "amount"},
		FieldOdometer:             {"odometer", "hub reading"},
		FieldCardLastFour:         {"card number", "card"},
		FieldTransactionReference: {"trans id", "transaction id", "invoice number"},
		FieldTractorCode:          {"unit", "unit number"},
		FieldDriverName:           {"driver name", "driver"},
	},
	fuelpurchase.CardProviderEFS: {
		FieldPurchasedAt:          {"transaction date", "trans date"},
		FieldVendor:               {"merchant name", "location name"},
		FieldCity:                 {"merchant city", "city"},
		FieldJurisdiction:         {"merchant state", "state"},
		FieldFuelType:             {"product code", "product"},
		FieldQuantity:             {"gallons", "quantity"},
		FieldUnitPrice:            {"ppg", "unit price"},
		FieldTotalAmount:          {"amount", "total amount"},
		FieldOdometer:             {"odometer"},
		FieldCardLastFour:         {"card number", "card"},
		FieldTransactionReference: {"transaction id", "transaction number"},
		FieldTractorCode:          {"unit number", "unit"},
		FieldDriverName:           {"driver id", "driver name"},
	},
	fuelpurchase.CardProviderWEX: {
		FieldPurchasedAt:          {"transaction date", "date"},
		FieldVendor:               {"merchant name", "merchant"},
		FieldCity:                 {"merchant city", "city"},
		FieldJurisdiction:         {"merchant state", "state"},
		FieldFuelType:             {"product description", "product"},
		FieldQuantity:             {"units", "quantity"},
		FieldUnitPrice:            {"unit cost", "price per gallon"},
		FieldTotalAmount:          {"gross amount", "net amount", "amount"},
		FieldOdometer:             {"odometer"},
		FieldCardLastFour:         {"card number", "card"},
		FieldTransactionReference: {"transaction number", "transaction id"},
		FieldTractorCode:          {"vehicle number", "vehicle id", "unit"},
		FieldDriverName:           {"driver name", "driver"},
	},
}

func Preset(provider fuelpurchase.CardProvider) map[Field][]string {
	specific := providerSynonyms[provider]
	preset := make(map[Field][]string, len(genericSynonyms))
	for _, field := range Fields() {
		names := make([]string, 0, len(specific[field])+len(genericSynonyms[field]))
		seen := make(map[string]struct{}, cap(names))
		for _, name := range specific[field] {
			key := normalizeHeader(name)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			names = append(names, name)
		}
		for _, name := range genericSynonyms[field] {
			key := normalizeHeader(name)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			names = append(names, name)
		}
		preset[field] = names
	}
	return preset
}

func normalizeHeader(header string) string {
	var b strings.Builder
	b.Grow(len(header))
	lastSpace := true
	for _, r := range strings.ToLower(strings.TrimSpace(header)) {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlnum {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func GuessMapping(headers []string, provider fuelpurchase.CardProvider) (Mapping, []string) {
	preset := Preset(provider)
	normalized := make([]string, len(headers))
	for i, header := range headers {
		normalized[i] = normalizeHeader(header)
	}

	mapping := make(Mapping, len(Fields()))
	claimed := make(map[int]struct{}, len(headers))

	for _, field := range Fields() {
		for _, synonym := range preset[field] {
			key := normalizeHeader(synonym)
			index := -1
			for i, header := range normalized {
				if _, taken := claimed[i]; taken || header == "" {
					continue
				}
				if header == key {
					index = i
					break
				}
			}
			if index >= 0 {
				mapping[field] = index
				claimed[index] = struct{}{}
				break
			}
		}
	}

	unmapped := make([]string, 0, len(headers)-len(claimed))
	for i, header := range headers {
		if _, taken := claimed[i]; taken {
			continue
		}
		if strings.TrimSpace(header) == "" {
			continue
		}
		unmapped = append(unmapped, header)
	}

	return mapping, unmapped
}

func UnmappedHeaders(headers []string, mapping Mapping) []string {
	claimed := make(map[int]struct{}, len(mapping))
	for _, index := range mapping {
		claimed[index] = struct{}{}
	}
	unmapped := make([]string, 0, len(headers))
	for i, header := range headers {
		if _, taken := claimed[i]; taken || strings.TrimSpace(header) == "" {
			continue
		}
		unmapped = append(unmapped, header)
	}
	return unmapped
}

func UnitFromHeader(header string) fuelpurchase.QuantityUnit {
	switch normalizeHeader(header) {
	case "litres", "liters", "litre", "liter", "l", "ltr", "ltrs":
		return fuelpurchase.QuantityUnitLitre
	default:
		return ""
	}
}
