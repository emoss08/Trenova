package shipmentquerytools

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/typeutils"
)

type FullRow struct {
	Row

	CustomerReference   string                  `json:"customerReference,omitempty"`
	BillTo              string                  `json:"billTo,omitempty"`
	FreightTerms        string                  `json:"freightTerms,omitempty"`
	ServiceType         string                  `json:"serviceType,omitempty"`
	ShipmentType        string                  `json:"shipmentType,omitempty"`
	Equipment           string                  `json:"equipment,omitempty"`
	TenderStatus        string                  `json:"tenderStatus,omitempty"`
	EntryMethod         string                  `json:"entryMethod,omitempty"`
	Owner               string                  `json:"owner,omitempty"`
	Stops               []string                `json:"stops,omitempty"`
	Moves               []string                `json:"moves,omitempty"`
	Miles               string                  `json:"miles,omitempty"`
	Commodities         []string                `json:"commodities,omitempty"`
	Pieces              *int64                  `json:"pieces,omitempty"`
	Weight              *int64                  `json:"weight,omitempty"`
	Temperature         string                  `json:"temperature,omitempty"`
	FreightCharge       string                  `json:"freightCharge,omitempty"`
	OtherCharges        string                  `json:"otherCharges,omitempty"`
	MarkedReadyToBillAt querytool.OptionalDate  `json:"markedReadyToBillAt"`
	BilledAt            querytool.OptionalDate  `json:"billedAt"`
	CanceledAt          *querytool.OptionalDate `json:"canceledAt,omitempty"`
	CancelReason        string                  `json:"cancelReason,omitempty"`
	UpdatedAt           querytool.OptionalDate  `json:"updatedAt"`
}

var (
	summaryColumns = querytool.ColumnsOf([]Row{{}})
	fullColumns    = querytool.ColumnsOf([]FullRow{{}})
	fullOnly       = slices.DeleteFunc(slices.Clone(fullColumns), func(name string) bool {
		return slices.Contains(summaryColumns, name)
	})
)

func FullOnlyColumns() []string { return slices.Clone(fullOnly) }

func ToFullRow(row Row) FullRow {
	full := FullRow{Row: row}
	item := row.source
	if item == nil {
		return full
	}

	full.CustomerReference = item.ExternalReference
	full.FreightTerms = string(item.FreightTerms)
	full.EntryMethod = string(item.EntryMethod)
	if item.TenderStatus != nil {
		full.TenderStatus = string(*item.TenderStatus)
	}
	if item.HasExplicitBillTo() && item.BillToCustomer != nil &&
		item.BillToCustomer.Name != row.Customer {
		full.BillTo = item.BillToCustomer.Name
	}
	if item.ServiceType != nil {
		full.ServiceType = codeAndDescription(item.ServiceType.Code, item.ServiceType.Description)
	}
	if item.ShipmentType != nil {
		full.ShipmentType = codeAndDescription(
			item.ShipmentType.Code,
			item.ShipmentType.Description,
		)
	}
	full.Equipment = equipmentLine(item)
	if item.Owner != nil {
		full.Owner = strings.TrimSpace(item.Owner.Name)
	}

	full.Stops, full.Moves, full.Miles = routeDetail(item)
	full.Commodities = commodityLines(item.Commodities)
	full.Pieces = item.Pieces
	full.Weight = item.Weight
	full.Temperature = temperatureRange(item.TemperatureMin, item.TemperatureMax)
	if item.FreightChargeAmount.Valid {
		full.FreightCharge = item.FreightChargeAmount.Decimal.String()
	}
	if item.OtherChargeAmount.Valid {
		full.OtherCharges = item.OtherChargeAmount.Decimal.String()
	}

	full.MarkedReadyToBillAt = querytool.ExpectedDate(
		typeutils.ValueOrZero(item.MarkedReadyToBillAt), "not marked ready to bill")
	full.BilledAt = querytool.ExpectedDate(
		typeutils.ValueOrZero(item.BilledAt), "not billed yet")
	if item.CanceledAt != nil && *item.CanceledAt > 0 {
		canceled := querytool.RecordedDate(*item.CanceledAt)
		full.CanceledAt = &canceled
		full.CancelReason = item.CancelReason
	}
	full.UpdatedAt = querytool.RecordedDate(item.UpdatedAt)

	return full
}

func codeAndDescription(code, description string) string {
	code = strings.TrimSpace(code)
	description = strings.TrimSpace(description)
	switch {
	case description == "" || strings.EqualFold(description, code):
		return code
	case code == "":
		return description
	default:
		return code + " (" + description + ")"
	}
}

func equipmentLine(item *shipment.Shipment) string {
	parts := make([]string, 0, 2)
	if item.TractorType != nil {
		parts = append(parts, "tractor "+codeAndDescription(
			item.TractorType.Code, item.TractorType.Description))
	}
	if item.TrailerType != nil {
		parts = append(parts, "trailer "+codeAndDescription(
			item.TrailerType.Code, item.TrailerType.Description))
	}

	return strings.Join(parts, "; ")
}

func routeDetail(item *shipment.Shipment) (stops, moves []string, miles string) {
	var total float64
	measured := false
	number := 0
	for _, move := range item.Moves {
		if move == nil || move.Status == shipmentvocab.MoveStatusCanceled {
			continue
		}
		number++
		line := fmt.Sprintf("Move %d (%s): %s", number, move.Status, moveCrewDetail(move))
		if move.Distance != nil && *move.Distance > 0 {
			total += *move.Distance
			measured = true
			line += " · " + formatMiles(*move.Distance) + " mi"
		}
		moves = append(moves, line)

		for _, stop := range move.Stops {
			if stop == nil || stop.Status == shipmentvocab.StopStatusCanceled {
				continue
			}
			stops = append(stops, fmt.Sprintf("%s (%s): %s", stop.Type, stop.Status, stopLine(stop)))
		}
	}
	if measured {
		miles = formatMiles(total)
	}

	return stops, moves, miles
}

func formatMiles(miles float64) string {
	return strconv.FormatFloat(miles, 'f', -1, 64)
}

func moveCrewDetail(move *shipment.ShipmentMove) string {
	if assignment := move.Assignment; assignment != nil && assignment.PrimaryWorkerID != nil &&
		assignment.PrimaryWorkerID.IsNotNil() {
		crew := querytool.WorkerName(assignment.PrimaryWorker)
		if crew == "" {
			crew = "a driver"
		}
		if second := querytool.WorkerName(assignment.SecondaryWorker); second != "" {
			crew += " and " + second
		}
		if tractor := assignment.Tractor; tractor != nil && tractor.Code != "" {
			crew += " on tractor " + tractor.Code
		}
		if trailer := assignment.Trailer; trailer != nil && trailer.Code != "" {
			crew += " with trailer " + trailer.Code
		}

		return crew
	}
	if carrier := move.CarrierAssignment; carrier != nil {
		if carrier.Carrier != nil && strings.TrimSpace(carrier.Carrier.Name) != "" {
			return "carrier " + strings.TrimSpace(carrier.Carrier.Name)
		}

		return "a carrier"
	}

	return CoverageNone
}

func commodityLines(commodities []*shipment.ShipmentCommodity) []string {
	lines := make([]string, 0, len(commodities))
	for _, line := range commodities {
		if line == nil || line.Commodity == nil {
			continue
		}
		name := strings.TrimSpace(line.Commodity.Name)
		if name == "" {
			continue
		}
		parts := []string{name}
		if line.Pieces > 0 {
			parts = append(parts, strconv.FormatInt(line.Pieces, 10)+" pieces")
		}
		if line.Weight > 0 {
			parts = append(parts, "weight "+strconv.FormatInt(line.Weight, 10))
		}
		if hazmat := line.Commodity.HazardousMaterial; hazmat != nil &&
			strings.TrimSpace(hazmat.Name) != "" {
			parts = append(parts, "hazmat "+strings.TrimSpace(hazmat.Name))
		}
		lines = append(lines, strings.Join(parts, " · "))
	}

	return lines
}

func temperatureRange(low, high *int16) string {
	switch {
	case low != nil && high != nil:
		return fmt.Sprintf("%d to %d", *low, *high)
	case low != nil:
		return fmt.Sprintf("at least %d", *low)
	case high != nil:
		return fmt.Sprintf("at most %d", *high)
	default:
		return ""
	}
}

func withFullRows(result any) any {
	outcome, ok := result.(querytool.SearchOutcome)
	if !ok {
		return result
	}

	var rows []Row
	switch typed := outcome.Items.(type) {
	case []Row:
		rows = typed
	case []any:
		rows = make([]Row, 0, len(typed))
		for _, item := range typed {
			if row, isRow := item.(Row); isRow {
				rows = append(rows, row)
			}
		}
	default:
		return result
	}
	if len(rows) == 0 {
		return result
	}

	full := make([]FullRow, 0, len(rows))
	for idx := range rows {
		full = append(full, ToFullRow(rows[idx]))
	}
	outcome.Items = full
	outcome.Columns = querytool.ColumnsOf(full)

	return outcome
}
