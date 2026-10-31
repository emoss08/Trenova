package reviewprompt

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuereview"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const explanationRunes = 600

func shipmentText(shp *shipment.Shipment) string {
	lines := make([]string, 0, 16)
	if shp.ServiceType != nil {
		lines = append(lines, "Service: "+codeAndDescription(shp.ServiceType.Code, shp.ServiceType.Description))
	}
	if shp.ShipmentType != nil {
		lines = append(lines, "Shipment type: "+
			codeAndDescription(shp.ShipmentType.Code, shp.ShipmentType.Description))
	}
	if equipment := equipmentText(shp.TractorType, shp.TrailerType); equipment != "" {
		lines = append(lines, "Equipment: "+equipment)
	}
	if shp.FreightTerms != "" {
		lines = append(lines, "Freight terms: "+string(shp.FreightTerms))
	}
	if shp.Pieces != nil {
		lines = append(lines, "Pieces: "+strconv.FormatInt(*shp.Pieces, 10))
	}
	if shp.Weight != nil {
		lines = append(lines, "Weight: "+strconv.FormatInt(*shp.Weight, 10)+" lb")
	}
	if shp.TemperatureMin != nil || shp.TemperatureMax != nil {
		lines = append(lines, "Temperature: "+temperature(shp.TemperatureMin)+" to "+
			temperature(shp.TemperatureMax))
	}
	if miles := distance(shp.Moves); miles != "" {
		lines = append(lines, "Distance: "+miles)
	}
	for _, sc := range shp.Commodities {
		lines = append(lines, "- commodity "+commodityText(sc))
	}
	lines = append(lines, ratingText(shp)...)
	for _, line := range shp.FreightLines {
		lines = append(lines, "- freight line "+freightLineText(line))
	}

	return strings.Join(lines, "\n")
}

func ratingText(shp *shipment.Shipment) []string {
	lines := make([]string, 0, 6)
	if detail := shp.RatingDetail; detail != nil {
		rated := []string{}
		if detail.AgreementName != "" {
			rated = append(rated, "agreement "+detail.AgreementName)
		}
		if detail.RuleLabel != "" {
			rated = append(rated, "rule "+detail.RuleLabel)
		}
		if detail.FormulaTemplateName != "" {
			rated = append(rated, "formula "+detail.FormulaTemplateName)
		}
		if detail.Source != "" {
			rated = append(rated, "source "+detail.Source)
		}
		if len(rated) > 0 {
			lines = append(lines, "Rated by: "+strings.Join(rated, " · "))
		}
		if detail.Explanation != "" {
			lines = append(lines, "How it was rated: "+
				stringutils.Ellipsize(detail.Explanation, explanationRunes))
		}
		if detail.Refusal != "" {
			lines = append(lines, "Rating refused: "+detail.Refusal)
		}
	}
	if shp.RateOverrideAmount.Valid {
		override := "Rate overridden to " + billingqueuereview.FormatMoney(shp.RateOverrideAmount.Decimal)
		if shp.RateOverrideReason != "" {
			override += ": " + shp.RateOverrideReason
		}
		lines = append(lines, override)
	}
	if shp.RateLocked {
		lines = append(lines, "The rate is locked.")
	}
	if shp.FuelSurchargeLocked {
		lines = append(lines, "The fuel surcharge is locked.")
	}

	return lines
}

func stopsText(shp *shipment.Shipment) string {
	lines := make([]string, 0, len(shp.Moves)*2)
	for _, move := range shp.Moves {
		for _, stop := range move.Stops {
			lines = append(lines, stopText(move, stop))
		}
	}

	return strings.Join(lines, "\n")
}

func stopText(move *shipment.ShipmentMove, stop *shipment.Stop) string {
	text := "Move " + strconv.FormatInt(move.Sequence+1, 10) + " stop " +
		strconv.FormatInt(stop.Sequence+1, 10) + " · " + string(stop.Type) + " · " + place(stop) +
		" · window " + timeutils.FormatInstantUTC(stop.ScheduledWindowStart)
	if stop.ScheduledWindowEnd != nil {
		text += " to " + timeutils.FormatInstantUTC(*stop.ScheduledWindowEnd)
	}
	if stop.ActualArrival != nil {
		text += " · arrived " + timeutils.FormatInstantUTC(*stop.ActualArrival)
	}
	if stop.ActualDeparture != nil {
		text += " · departed " + timeutils.FormatInstantUTC(*stop.ActualDeparture)
	}
	if stop.ActualArrival != nil && stop.ActualDeparture != nil {
		text += " · on site " + timeutils.FormatLongDurationMs((*stop.ActualDeparture-*stop.ActualArrival)*1000)
	}
	if stop.Pieces != nil {
		text += " · " + strconv.FormatInt(*stop.Pieces, 10) + " pieces"
	}
	if stop.Weight != nil {
		text += " · " + strconv.FormatInt(*stop.Weight, 10) + " lb"
	}
	if stop.CountDetentionOverride != nil {
		if *stop.CountDetentionOverride {
			text += " · detention counted by hand"
		} else {
			text += " · detention waived by hand"
		}
	}

	return text
}

func termsText(e *Evidence) string {
	profile := billingProfile(e)
	if profile == nil {
		return ""
	}
	lines := []string{"Payment terms: " + string(profile.PaymentTerm)}
	if fuel := fuelTerms(profile.FuelSurchargeMode); fuel != "" {
		lines = append(lines, fuel)
	}
	if profile.AutoApplyAccessorials {
		lines = append(lines, "Accessorials are applied automatically.")
	}
	required := []string{}
	if profile.RequirePONumber {
		required = append(required, "a PO number")
	}
	if profile.RequireBOLNumber {
		required = append(required, "a BOL number")
	}
	if profile.RequireDeliveryNumber {
		required = append(required, "a delivery number")
	}
	if len(required) > 0 {
		lines = append(lines, "Every invoice needs "+strings.Join(required, ", ")+".")
	}
	if len(profile.DocumentTypes) > 0 {
		names := make([]string, 0, len(profile.DocumentTypes))
		for _, kind := range profile.DocumentTypes {
			names = append(names, kind.Name)
		}
		lines = append(lines, "Documents the payer requires: "+strings.Join(names, ", "))
	}

	return strings.Join(lines, "\n")
}

// fuelTerms says what the payer's fuel setting means for a fuel charge on the
// invoice. None only stops one being added automatically.
func fuelTerms(mode customer.FuelSurchargeMode) string {
	switch mode {
	case customer.FuelSurchargeModeNone:
		return "Fuel surcharge: none is added automatically; one entered by hand is normal."
	case customer.FuelSurchargeModeProgram:
		return "Fuel surcharge: added automatically from the payer's fuel surcharge program."
	case customer.FuelSurchargeModeFuelIncluded:
		return "Fuel is included in the rates: a separate fuel charge would bill it twice."
	default:
		return ""
	}
}

func billingProfile(e *Evidence) *customer.CustomerBillingProfile {
	if e.Item.BillToCustomer == nil {
		return nil
	}

	return e.Item.BillToCustomer.BillingProfile
}

func place(stop *shipment.Stop) string {
	loc := stop.Location
	if loc == nil {
		return stop.AddressLine
	}
	text := loc.Name
	if loc.City != "" {
		text += ", " + loc.City
	}
	if loc.State != nil && loc.State.Abbreviation != "" {
		text += " " + loc.State.Abbreviation
	}

	return text
}

func equipmentText(tractor, trailer *equipmenttype.EquipmentType) string {
	parts := []string{}
	for _, kind := range []*equipmenttype.EquipmentType{tractor, trailer} {
		if kind != nil {
			parts = append(parts, codeAndDescription(kind.Code, kind.Description))
		}
	}

	return strings.Join(parts, " with ")
}

func commodityText(sc *shipment.ShipmentCommodity) string {
	name := "unnamed"
	hazmat := false
	if sc.Commodity != nil {
		name = sc.Commodity.Name
		hazmat = !sc.Commodity.HazardousMaterialID.IsNil()
	}
	text := name + " · " + strconv.FormatInt(sc.Pieces, 10) + " pieces · " +
		strconv.FormatInt(sc.Weight, 10) + " lb"
	if hazmat {
		text += " · hazardous"
	}

	return text
}

func freightLineText(line shipment.FreightLine) string {
	text := line.Label
	if line.Quantity != nil {
		text += " · " + line.Quantity.String() + " " + line.QuantityUnit
	}
	if line.Rate != nil {
		text += " at " + line.Rate.String() + " " + line.RateUnit
	}

	return text + " = " + billingqueuereview.FormatMoney(line.Amount)
}

func distance(moves []*shipment.ShipmentMove) string {
	total := 0.0
	units := ""
	for _, move := range moves {
		if move.Distance != nil {
			total += *move.Distance
			units = firstOf(units, move.DistanceUnits)
		}
	}
	if total == 0 {
		return ""
	}

	return strings.TrimSpace(number(total) + " " + firstOf(units, "miles"))
}

func temperature(value *int16) string {
	if value == nil {
		return "any"
	}

	return strconv.Itoa(int(*value))
}

func codeAndDescription(code, description string) string {
	if description == "" || description == code {
		return code
	}

	return code + " (" + description + ")"
}
