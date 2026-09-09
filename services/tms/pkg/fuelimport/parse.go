package fuelimport

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/shopspring/decimal"
)

var ErrBlankRow = errors.New("this row is empty")

const (
	syntheticPrefix   = "SYNTH:"
	cardDigitsNeeded  = 4
	excelEpochOffset  = 25569
	secondsPerDay     = 86400
	maxExcelSerial    = 2958465
	minExcelSerial    = 20000
	maxOdometerDigits = 9
)

type ParseOptions struct {
	Provider        fuelpurchase.CardProvider
	DefaultFuelType domaintypes.IFTAFuelType
	DefaultCurrency string
	DefaultUnit     fuelpurchase.QuantityUnit
	Location        *time.Location
}

func (o ParseOptions) location() *time.Location {
	if o.Location == nil {
		return time.UTC
	}
	return o.Location
}

type ParsedRow struct {
	Purchase         fuelpurchase.FuelPurchase
	CountryCode      string
	JurisdictionCode string
	TractorCode      string
	LicensePlate     string
	CardLastFour     string
	Reference        string
	DriverName       string
}

func (r *ParsedRow) JurisdictionKey() string {
	return r.CountryCode + "_" + r.JurisdictionCode
}

func cellAt(cells []string, mapping Mapping, field Field) string {
	index, ok := mapping[field]
	if !ok || index < 0 || index >= len(cells) {
		return ""
	}
	return strings.TrimSpace(cells[index])
}

func isBlank(cells []string) bool {
	for _, cell := range cells {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func ParseRow(cells []string, mapping Mapping, opts ParseOptions) (*ParsedRow, error) {
	if isBlank(cells) {
		return nil, ErrBlankRow
	}

	row := &ParsedRow{}
	purchase := &row.Purchase

	purchasedAt, err := ParseDate(cellAt(cells, mapping, FieldPurchasedAt), opts.location())
	if err != nil {
		return nil, err
	}
	purchase.PurchasedAt = purchasedAt

	country, code, ok := NormalizeJurisdiction(cellAt(cells, mapping, FieldJurisdiction))
	if !ok {
		return nil, fmt.Errorf(
			"%q is not a recognised state or province",
			cellAt(cells, mapping, FieldJurisdiction),
		)
	}
	row.CountryCode = country
	row.JurisdictionCode = code

	fuelType, err := resolveFuelType(cellAt(cells, mapping, FieldFuelType), opts.DefaultFuelType)
	if err != nil {
		return nil, err
	}
	purchase.FuelType = fuelType

	quantity, err := ParseNumber(cellAt(cells, mapping, FieldQuantity))
	if err != nil {
		return nil, fmt.Errorf(
			"quantity %q could not be read",
			cellAt(cells, mapping, FieldQuantity),
		)
	}
	if !quantity.IsPositive() {
		return nil, errors.New("quantity must be greater than zero")
	}
	purchase.Quantity = quantity
	purchase.QuantityUnit = resolveUnit(cellAt(cells, mapping, FieldUnit), opts.DefaultUnit)
	purchase.Gallons = purchase.QuantityUnit.ToGallons(quantity)

	if raw := cellAt(cells, mapping, FieldUnitPrice); raw != "" {
		unitPrice, priceErr := ParseNumber(raw)
		if priceErr != nil {
			return nil, fmt.Errorf("unit price %q could not be read", raw)
		}
		if unitPrice.IsNegative() {
			return nil, errors.New("unit price cannot be negative")
		}
		purchase.UnitPrice = decimal.NewNullDecimal(unitPrice)
	}

	rawAmount := cellAt(cells, mapping, FieldTotalAmount)
	amount, err := ParseNumber(rawAmount)
	if err != nil {
		return nil, fmt.Errorf("total amount %q could not be read", rawAmount)
	}
	if amount.IsNegative() {
		return nil, errors.New(
			"total amount is negative; reversals and credits are not imported as purchases",
		)
	}
	purchase.TotalAmountMinor = money.MinorUnits(amount)

	purchase.CurrencyCode = resolveCurrency(
		cellAt(cells, mapping, FieldCurrency),
		opts.DefaultCurrency,
	)
	if !domaintypes.CurrencyCodeRegex.MatchString(purchase.CurrencyCode) {
		return nil, fmt.Errorf("currency %q is not a three-letter code", purchase.CurrencyCode)
	}

	if raw := cellAt(cells, mapping, FieldOdometer); raw != "" {
		odometer, odoErr := parseOdometer(raw)
		if odoErr != nil {
			return nil, odoErr
		}
		if odometer != nil {
			purchase.Odometer = odometer
		}
	}

	purchase.Vendor = cellAt(cells, mapping, FieldVendor)
	purchase.VendorCity = cellAt(cells, mapping, FieldCity)
	purchase.TaxPaid = true
	purchase.Source = fuelpurchase.PurchaseSourceCardImport

	row.CardLastFour = CardLastFour(cellAt(cells, mapping, FieldCardLastFour))
	purchase.CardLastFour = row.CardLastFour
	row.Reference = strings.ToUpper(cellAt(cells, mapping, FieldTransactionReference))
	purchase.TransactionReference = row.Reference
	row.TractorCode = cellAt(cells, mapping, FieldTractorCode)
	row.LicensePlate = cellAt(cells, mapping, FieldLicensePlate)
	row.DriverName = cellAt(cells, mapping, FieldDriverName)

	return row, nil
}

var dateLayouts = [...]string{
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
	"01/02/2006 15:04:05",
	"01/02/2006 15:04",
	"01/02/2006 3:04:05 PM",
	"01/02/2006 3:04 PM",
	"01/02/2006",
	"1/2/2006 15:04:05",
	"1/2/2006 15:04",
	"1/2/2006 3:04:05 PM",
	"1/2/2006 3:04 PM",
	"1/2/2006",
	"01/02/06 15:04",
	"01/02/06",
	"1/2/06 15:04",
	"1/2/06",
	"2006/01/02",
	"01-02-2006",
	"1-2-2006",
	"02-Jan-2006",
	"2-Jan-2006",
	"Jan 2, 2006",
	"January 2, 2006",
	"20060102",
}

func ParseDate(raw string, loc *time.Location) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, errors.New("transaction date is missing")
	}
	if loc == nil {
		loc = time.UTC
	}

	for _, layout := range dateLayouts {
		parsed, err := time.ParseInLocation(layout, value, loc)
		if err == nil {
			return parsed.Unix(), nil
		}
	}

	if serial, err := strconv.ParseFloat(value, 64); err == nil {
		if serial >= minExcelSerial && serial <= maxExcelSerial {
			return excelSerialToUnix(serial, loc), nil
		}
	}

	return 0, fmt.Errorf("transaction date %q is not a recognised date", raw)
}

func excelSerialToUnix(serial float64, loc *time.Location) int64 {
	days := math.Floor(serial)
	fraction := serial - days
	utcDay := int64(days-excelEpochOffset) * secondsPerDay
	seconds := int64(math.Round(fraction * secondsPerDay))
	utc := time.Unix(utcDay+seconds, 0).UTC()
	local := time.Date(
		utc.Year(), utc.Month(), utc.Day(),
		utc.Hour(), utc.Minute(), utc.Second(), 0, loc,
	)
	return local.Unix()
}

func ParseNumber(raw string) (decimal.Decimal, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return decimal.Zero, errors.New("value is missing")
	}

	negative := false
	if strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")") {
		negative = true
		value = strings.TrimSuffix(strings.TrimPrefix(value, "("), ")")
	}

	var b strings.Builder
	b.Grow(len(value))
scan:
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9', r == '.':
			b.WriteRune(r)
		case r == '-':
			negative = !negative
		case r == ',', r == '$', r == ' ', r == '\u00a0', r == '+':
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
			if b.Len() > 0 {
				break scan
			}
		default:
			return decimal.Zero, fmt.Errorf("%q is not a number", raw)
		}
	}
	cleaned := b.String()
	if cleaned == "" || cleaned == "." {
		return decimal.Zero, fmt.Errorf("%q is not a number", raw)
	}

	parsed, err := decimal.NewFromString(cleaned)
	if err != nil {
		return decimal.Zero, fmt.Errorf("%q is not a number", raw)
	}
	if negative {
		parsed = parsed.Neg()
	}

	return parsed, nil
}

func parseOdometer(raw string) (*int64, error) {
	value, err := ParseNumber(raw)
	if err != nil {
		return nil, fmt.Errorf("odometer %q could not be read", raw)
	}
	if value.IsNegative() {
		return nil, errors.New("odometer cannot be negative")
	}
	if value.IsZero() {
		return nil, nil
	}
	rounded := value.Round(0)
	if len(rounded.String()) > maxOdometerDigits {
		return nil, fmt.Errorf("odometer %q is out of range", raw)
	}
	odometer := rounded.IntPart()
	return &odometer, nil
}

func resolveUnit(raw string, fallback fuelpurchase.QuantityUnit) fuelpurchase.QuantityUnit {
	switch normalizeHeader(raw) {
	case "l", "ltr", "ltrs", "litre", "litres", "liter", "liters":
		return fuelpurchase.QuantityUnitLitre
	case "g", "gal", "gals", "gallon", "gallons", "us gal", "us gallons", "usg":
		return fuelpurchase.QuantityUnitGallon
	}
	if fallback.IsValid() {
		return fallback
	}
	return fuelpurchase.QuantityUnitGallon
}

func resolveCurrency(raw, fallback string) string {
	code := strings.ToUpper(strings.TrimSpace(raw))
	switch code {
	case "$", "US$", "USD$":
		code = money.DefaultCurrencyCode
	case "C$", "CA$", "CAD$":
		code = "CAD"
	}
	if code == "" {
		code = strings.ToUpper(strings.TrimSpace(fallback))
	}
	if code == "" {
		code = money.DefaultCurrencyCode
	}
	return code
}

var fuelSynonyms = map[string]domaintypes.IFTAFuelType{
	"diesel":                 domaintypes.IFTAFuelTypeDiesel,
	"dsl":                    domaintypes.IFTAFuelTypeDiesel,
	"ulsd":                   domaintypes.IFTAFuelTypeDiesel,
	"uls diesel":             domaintypes.IFTAFuelTypeDiesel,
	"diesel 2":               domaintypes.IFTAFuelTypeDiesel,
	"diesel no 2":            domaintypes.IFTAFuelTypeDiesel,
	"2 diesel":               domaintypes.IFTAFuelTypeDiesel,
	"tractor diesel":         domaintypes.IFTAFuelTypeDiesel,
	"tractor fuel":           domaintypes.IFTAFuelTypeDiesel,
	"trac":                   domaintypes.IFTAFuelTypeDiesel,
	"dsl premium":            domaintypes.IFTAFuelTypeDiesel,
	"premium diesel":         domaintypes.IFTAFuelTypeDiesel,
	"clear diesel":           domaintypes.IFTAFuelTypeDiesel,
	"winter diesel":          domaintypes.IFTAFuelTypeDiesel,
	"renewable diesel":       domaintypes.IFTAFuelTypeDiesel,
	"r99":                    domaintypes.IFTAFuelTypeDiesel,
	"gasoline":               domaintypes.IFTAFuelTypeGasoline,
	"gas":                    domaintypes.IFTAFuelTypeGasoline,
	"unleaded":               domaintypes.IFTAFuelTypeGasoline,
	"unl":                    domaintypes.IFTAFuelTypeGasoline,
	"regular":                domaintypes.IFTAFuelTypeGasoline,
	"regular unleaded":       domaintypes.IFTAFuelTypeGasoline,
	"premium unleaded":       domaintypes.IFTAFuelTypeGasoline,
	"mid grade":              domaintypes.IFTAFuelTypeGasoline,
	"midgrade":               domaintypes.IFTAFuelTypeGasoline,
	"petrol":                 domaintypes.IFTAFuelTypeGasoline,
	"gasohol":                domaintypes.IFTAFuelTypeGasohol,
	"e10":                    domaintypes.IFTAFuelTypeGasohol,
	"e85":                    domaintypes.IFTAFuelTypeE85,
	"m85":                    domaintypes.IFTAFuelTypeM85,
	"a55":                    domaintypes.IFTAFuelTypeA55,
	"ethanol":                domaintypes.IFTAFuelTypeEthanol,
	"methanol":               domaintypes.IFTAFuelTypeMethanol,
	"propane":                domaintypes.IFTAFuelTypePropane,
	"lpg":                    domaintypes.IFTAFuelTypePropane,
	"autogas":                domaintypes.IFTAFuelTypePropane,
	"cng":                    domaintypes.IFTAFuelTypeCNG,
	"compressed natural gas": domaintypes.IFTAFuelTypeCNG,
	"lng":                    domaintypes.IFTAFuelTypeLNG,
	"liquefied natural gas":  domaintypes.IFTAFuelTypeLNG,
	"biodiesel":              domaintypes.IFTAFuelTypeBiodiesel,
	"bio diesel":             domaintypes.IFTAFuelTypeBiodiesel,
	"b20":                    domaintypes.IFTAFuelTypeBiodiesel,
	"b5":                     domaintypes.IFTAFuelTypeBiodiesel,
	"b99":                    domaintypes.IFTAFuelTypeBiodiesel,
	"b100":                   domaintypes.IFTAFuelTypeBiodiesel,
	"electric":               domaintypes.IFTAFuelTypeElectricity,
	"electricity":            domaintypes.IFTAFuelTypeElectricity,
	"ev charge":              domaintypes.IFTAFuelTypeElectricity,
	"ev charging":            domaintypes.IFTAFuelTypeElectricity,
	"kwh":                    domaintypes.IFTAFuelTypeElectricity,
	"hydrogen":               domaintypes.IFTAFuelTypeHydrogen,
	"h2":                     domaintypes.IFTAFuelTypeHydrogen,
	"def":                    domaintypes.IFTAFuelTypeDEF,
	"diesel exhaust fluid":   domaintypes.IFTAFuelTypeDEF,
	"exhaust fluid":          domaintypes.IFTAFuelTypeDEF,
	"adblue":                 domaintypes.IFTAFuelTypeDEF,
	"reefer":                 domaintypes.IFTAFuelTypeReefer,
	"reefer fuel":            domaintypes.IFTAFuelTypeReefer,
	"reefer diesel":          domaintypes.IFTAFuelTypeReefer,
	"rfr":                    domaintypes.IFTAFuelTypeReefer,
	"refrigeration":          domaintypes.IFTAFuelTypeReefer,
	"dyed diesel":            domaintypes.IFTAFuelTypeReefer,
	"red diesel":             domaintypes.IFTAFuelTypeReefer,
	"off road diesel":        domaintypes.IFTAFuelTypeReefer,
	"other":                  domaintypes.IFTAFuelTypeOther,
	"misc":                   domaintypes.IFTAFuelTypeOther,
	"oil":                    domaintypes.IFTAFuelTypeOther,
	"additive":               domaintypes.IFTAFuelTypeOther,
}

func FuelTypeFromProduct(raw string) (domaintypes.IFTAFuelType, bool) {
	key := normalizeHeader(raw)
	if key == "" {
		return "", false
	}
	if exact, ok := fuelSynonyms[key]; ok {
		return exact, true
	}
	if candidate := domaintypes.IFTAFuelType(strings.TrimSpace(raw)); candidate.IsValid() {
		return candidate, true
	}

	words := strings.Fields(key)
	if len(words) == 1 {
		return "", false
	}
	for size := len(words) - 1; size >= 1; size-- {
		for start := 0; start+size <= len(words); start++ {
			phrase := strings.Join(words[start:start+size], " ")
			if match, ok := fuelSynonyms[phrase]; ok {
				return match, true
			}
		}
	}

	return "", false
}

func resolveFuelType(
	raw string,
	fallback domaintypes.IFTAFuelType,
) (domaintypes.IFTAFuelType, error) {
	if raw != "" {
		if fuelType, ok := FuelTypeFromProduct(raw); ok {
			return fuelType, nil
		}
		if fallback.IsValid() {
			return fallback, nil
		}
		return "", fmt.Errorf("fuel product %q is not recognised", raw)
	}
	if fallback.IsValid() {
		return fallback, nil
	}
	return "", errors.New("fuel product is missing and the import has no default fuel type")
}

func CardLastFour(raw string) string {
	digits := make([]byte, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		if raw[i] >= '0' && raw[i] <= '9' {
			digits = append(digits, raw[i])
		}
	}
	if len(digits) < cardDigitsNeeded {
		return ""
	}
	return string(digits[len(digits)-cardDigitsNeeded:])
}

func SyntheticReference(
	provider fuelpurchase.CardProvider,
	lastFour string,
	purchasedAt int64,
	jurisdictionCode string,
	gallons decimal.Decimal,
	amountMinor int64,
) string {
	var b strings.Builder
	b.Grow(96)
	b.WriteString(string(provider))
	b.WriteByte('|')
	b.WriteString(strings.TrimSpace(lastFour))
	b.WriteByte('|')
	b.WriteString(strconv.FormatInt(purchasedAt, 10))
	b.WriteByte('|')
	b.WriteString(strings.ToUpper(strings.TrimSpace(jurisdictionCode)))
	b.WriteByte('|')
	b.WriteString(gallons.StringFixed(3))
	b.WriteByte('|')
	b.WriteString(strconv.FormatInt(amountMinor, 10))

	sum := sha256.Sum256([]byte(b.String()))
	return syntheticPrefix + strings.ToUpper(hex.EncodeToString(sum[:]))
}

func IsSyntheticReference(reference string) bool {
	return strings.HasPrefix(strings.ToUpper(reference), syntheticPrefix)
}
