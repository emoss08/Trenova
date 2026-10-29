package record

import (
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	ModeSell = "Sell"
	ModeBuy  = "Buy"

	DistanceMiles      = "mi"
	DistanceKilometers = "km"

	unitWeight      = "lb"
	unitLength      = "ft"
	unitVolume      = "ft3"
	unitDuration    = "h"
	unitTemperature = "°F"
	unitDensity     = "lb/ft3"
	unitFuelPrice   = "$/gal"

	zip3Length = 3
)

type Place struct {
	City    string
	State   string
	Zip     string
	Country string
	Zones   []string
}

type Fuel struct {
	Price     decimal.NullDecimal
	PriceDate string
	Index     string
}

type Rating struct {
	At        time.Time
	Mode      string
	SellTotal decimal.NullDecimal
	Currency  string
}

type Input struct {
	Shipment     *shipment.Shipment
	Origin       Place
	Destination  Place
	Fuel         Fuel
	Rating       Rating
	DistanceUnit string
	Holidays     []Holiday
	ChargeCodes  map[pulid.ID]string
}

func PlaceOf(stop *shipment.Stop) Place {
	if stop == nil || stop.Location == nil {
		return Place{}
	}
	place := Place{City: stop.Location.City, Zip: stop.Location.PostalCode}
	if state := stop.Location.State; state != nil {
		place.State = state.Abbreviation
		place.Country = state.CountryIso3
	}
	return place
}

func Zip3(zip string) string {
	text := []rune(strings.ToUpper(strings.TrimSpace(zip)))
	if len(text) > zip3Length {
		text = text[:zip3Length]
	}
	return string(text)
}

func Build(in *Input) (rules.RawJSON, error) {
	if in == nil || in.Shipment == nil {
		return nil, ErrNoShipment
	}
	distanceUnit := in.DistanceUnit
	if distanceUnit != DistanceKilometers {
		distanceUnit = DistanceMiles
	}
	zone := in.Rating.At.Location()
	out := buildShipment(in.Shipment, distanceUnit, zone)
	out.Accessorials = buildAccessorials(in.Shipment.AdditionalCharges, in.ChargeCodes)
	encoded, err := sonic.Marshal(&root{
		Shipment: out,
		Lane:     buildLane(&in.Origin, &in.Destination),
		Customer: buildCustomer(in.Shipment),
		Fuel:     buildFuel(&in.Fuel),
		Rating:   buildRating(&in.Rating, in.Holidays),
	})
	if err != nil {
		return nil, fmt.Errorf("encode the tariff record: %w", err)
	}
	return encoded, nil
}

type root struct {
	Shipment *shipmentRoot `json:"shipment"`
	Lane     *laneRoot     `json:"lane"`
	Customer *customerRoot `json:"customer"`
	Fuel     *fuelRoot     `json:"fuel"`
	Rating   *ratingRoot   `json:"rating"`
}

type reference struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type temperature struct {
	Min rules.RawJSON `json:"min"`
	Max rules.RawJSON `json:"max"`
}

type shipmentRoot struct {
	ProNumber    string               `json:"proNumber"`
	Status       string               `json:"status"`
	Weight       rules.RawJSON        `json:"weight"`
	Pieces       rules.RawJSON        `json:"pieces"`
	Distance     rules.RawJSON        `json:"distance"`
	Duration     rules.RawJSON        `json:"duration"`
	LinearFeet   rules.RawJSON        `json:"linearFeet"`
	Volume       rules.RawJSON        `json:"volume"`
	Density      rules.RawJSON        `json:"density"`
	Hazmat       bool                 `json:"hazmat"`
	Reefer       bool                 `json:"reefer"`
	Temperature  temperature          `json:"temperature"`
	FreightClass rules.RawJSON        `json:"freightClass"`
	HighestClass rules.RawJSON        `json:"highestClass"`
	ServiceType  reference            `json:"serviceType"`
	ShipmentType *reference           `json:"shipmentType"`
	TractorType  *reference           `json:"tractorType"`
	TrailerType  *reference           `json:"trailerType"`
	Stops        []*stopRecord        `json:"stops"`
	Commodities  []*commodityRecord   `json:"commodities"`
	Pickup       *stopRecord          `json:"pickup"`
	Delivery     *stopRecord          `json:"delivery"`
	Accessorials []*accessorialRecord `json:"accessorials"`
}

type accessorialRecord struct {
	Code     string        `json:"code"`
	Quantity rules.RawJSON `json:"quantity"`
}

type window struct {
	Start rules.RawJSON `json:"start"`
	End   rules.RawJSON `json:"end"`
}

type stopLocation struct {
	City                string        `json:"city"`
	State               string        `json:"state"`
	Zip                 string        `json:"zip"`
	Zip3                string        `json:"zip3"`
	Country             string        `json:"country"`
	Timezone            string        `json:"timezone"`
	Category            rules.RawJSON `json:"category"`
	FacilityType        rules.RawJSON `json:"facilityType"`
	RequiresAppointment bool          `json:"requiresAppointment"`
}

type stopRecord struct {
	Sequence    rules.RawJSON `json:"sequence"`
	Kind        string        `json:"kind"`
	Status      string        `json:"status"`
	Appointment bool          `json:"appointment"`
	Window      window        `json:"window"`
	Arrival     rules.RawJSON `json:"arrival"`
	Departure   rules.RawJSON `json:"departure"`
	Dwell       rules.RawJSON `json:"dwell"`
	Pieces      rules.RawJSON `json:"pieces"`
	Weight      rules.RawJSON `json:"weight"`
	LegDistance rules.RawJSON `json:"legDistance"`
	Location    stopLocation  `json:"location"`
}

type dimensions struct {
	Length rules.RawJSON `json:"length"`
	Width  rules.RawJSON `json:"width"`
	Height rules.RawJSON `json:"height"`
}

type commodityRecord struct {
	Name         string        `json:"name"`
	FreightClass rules.RawJSON `json:"freightClass"`
	Weight       rules.RawJSON `json:"weight"`
	Pieces       rules.RawJSON `json:"pieces"`
	Dimensions   dimensions    `json:"dimensions"`
	Volume       rules.RawJSON `json:"volume"`
	Density      rules.RawJSON `json:"density"`
	Hazmat       bool          `json:"hazmat"`
	Stackable    bool          `json:"stackable"`
}

type lanePlace struct {
	City    string   `json:"city"`
	State   string   `json:"state"`
	Zip     string   `json:"zip"`
	Zip3    string   `json:"zip3"`
	Country string   `json:"country"`
	Zones   []string `json:"zones"`
}

type laneRoot struct {
	Origin      lanePlace `json:"origin"`
	Destination lanePlace `json:"destination"`
	Interstate  bool      `json:"interstate"`
	CrossBorder bool      `json:"crossBorder"`
}

type customerRoot struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type fuelRoot struct {
	Price     rules.RawJSON `json:"price"`
	PriceDate rules.RawJSON `json:"priceDate"`
	Index     rules.RawJSON `json:"index"`
}

type ratingRoot struct {
	Date      rules.RawJSON    `json:"date"`
	Mode      string           `json:"mode"`
	SellTotal rules.RawJSON    `json:"sellTotal"`
	Currency  string           `json:"currency"`
	Holidays  []*holidayRecord `json:"holidays"`
}

type holidayRecord struct {
	Date rules.RawJSON `json:"date"`
	Name string        `json:"name"`
}

func buildShipment(
	entity *shipment.Shipment,
	distanceUnit string,
	zone *time.Location,
) *shipmentRoot {
	weight := ratingWeight(entity)
	volume := entity.RatingCubicFeet()
	out := &shipmentRoot{
		ProNumber:  entity.ProNumber,
		Status:     orNew(string(entity.Status), string(shipmentvocab.StatusNew)),
		Weight:     optionalQuantity(weight, unitWeight),
		Pieces:     optionalInt(ratingPieces(entity)),
		Distance:   rules.QuantityValue(entity.RatingDistance(), distanceUnit),
		Duration:   rules.QuantityValue(entity.RatingDurationHours(), unitDuration),
		LinearFeet: rules.QuantityValue(entity.RatingLinearFeet(), unitLength),
		Volume:     rules.QuantityValue(volume, unitVolume),
		Density:    density(weight, volume),
		Hazmat:     entity.HasHazardousMaterial(),
		Reefer:     entity.TemperatureMin != nil || entity.TemperatureMax != nil,
		Temperature: temperature{
			Min: temperatureValue(entity.TemperatureMin),
			Max: temperatureValue(entity.TemperatureMax),
		},
		FreightClass: freightClass(entity.PrimaryFreightClass()),
		HighestClass: freightClass(entity.HighestFreightClass()),
		ServiceType:  reference{},
		Stops:        make([]*stopRecord, 0, len(entity.Moves)*2),
		Commodities:  make([]*commodityRecord, 0, len(entity.Commodities)),
	}
	if entity.ServiceType != nil {
		out.ServiceType = reference{
			Code:        entity.ServiceType.Code,
			Description: entity.ServiceType.Description,
		}
	}
	if entity.ShipmentType != nil {
		out.ShipmentType = &reference{
			Code:        entity.ShipmentType.Code,
			Description: entity.ShipmentType.Description,
		}
	}
	if entity.TractorType != nil {
		out.TractorType = &reference{
			Code:        entity.TractorType.Code,
			Description: entity.TractorType.Description,
		}
	}
	if entity.TrailerType != nil {
		out.TrailerType = &reference{
			Code:        entity.TrailerType.Code,
			Description: entity.TrailerType.Description,
		}
	}
	ordered := entity.OrderedStops()
	legs := legDistances(entity, ordered)
	for i, stop := range ordered {
		record := buildStop(stop, int64(i+1), zone)
		if distance, ok := legs[stop]; ok {
			record.LegDistance = rules.QuantityValue(distance, distanceUnit)
		}
		out.Stops = append(out.Stops, record)
		if stop.IsCanceled() {
			continue
		}
		if stop.IsOriginStop() && out.Pickup == nil {
			out.Pickup = record
		}
		if stop.IsDestinationStop() {
			out.Delivery = record
		}
	}
	for _, line := range entity.Commodities {
		if line != nil {
			out.Commodities = append(out.Commodities, buildCommodity(line))
		}
	}
	return out
}

func ratingWeight(entity *shipment.Shipment) decimal.NullDecimal {
	weight := entity.RatingWeight()
	if weight.IsPositive() {
		return decimal.NewNullDecimal(weight)
	}
	if entity.Weight != nil {
		return decimal.NewNullDecimal(decimal.NewFromInt(*entity.Weight))
	}
	return decimal.NullDecimal{}
}

func ratingPieces(entity *shipment.Shipment) *int64 {
	pieces := entity.RatingPieces()
	if pieces > 0 {
		return &pieces
	}
	return entity.Pieces
}

func buildStop(stop *shipment.Stop, sequence int64, fallback *time.Location) *stopRecord {
	zone := fallback
	location := stopLocation{
		Timezone:     fallback.String(),
		Category:     rules.NoneValue(),
		FacilityType: rules.NoneValue(),
	}
	if stop.Location != nil {
		place := PlaceOf(stop)
		location.City = place.City
		location.State = place.State
		location.Zip = place.Zip
		location.Zip3 = Zip3(place.Zip)
		location.Country = place.Country
		if stop.Location.Timezone != "" {
			zone = timeutils.LoadLocation(stop.Location.Timezone)
			location.Timezone = zone.String()
		}
		applyCategory(&location, stop.Location.LocationCategory)
	}
	record := &stopRecord{
		Sequence:    rules.IntValue(sequence),
		Kind:        string(stop.Type),
		Status:      orNew(string(stop.Status), string(shipmentvocab.StopStatusNew)),
		Appointment: stop.ScheduleType == shipmentvocab.StopScheduleTypeAppointment,
		Window: window{
			Start: rules.NoneValue(),
			End:   optionalTime(stop.ScheduledWindowEnd, zone),
		},
		Arrival:     optionalTime(stop.ActualArrival, zone),
		Departure:   optionalTime(stop.ActualDeparture, zone),
		Dwell:       rules.NoneValue(),
		Pieces:      optionalInt(stop.Pieces),
		Weight:      rules.NoneValue(),
		LegDistance: rules.NoneValue(),
		Location:    location,
	}
	if stop.ScheduledWindowStart > 0 {
		record.Window.Start = timeValue(stop.ScheduledWindowStart, zone)
	}
	if seconds, ok := stop.DwellSeconds(); ok {
		record.Dwell = rules.QuantityValue(
			decimal.NewFromInt(seconds).Div(decimal.NewFromInt(3600)),
			unitDuration,
		)
	}
	if stop.Weight != nil {
		record.Weight = rules.QuantityValue(decimal.NewFromInt(*stop.Weight), unitWeight)
	}
	return record
}

func applyCategory(location *stopLocation, category *locationcategory.LocationCategory) {
	if category == nil {
		return
	}
	if category.Type.IsValid() {
		location.Category = rules.TextValue(category.Type.String())
	}
	if category.FacilityType.IsValid() {
		location.FacilityType = rules.TextValue(category.FacilityType.String())
	}
	location.RequiresAppointment = category.RequiresAppointment
}

func legDistances(
	entity *shipment.Shipment,
	ordered []*shipment.Stop,
) map[*shipment.Stop]decimal.Decimal {
	moveOf := make(map[*shipment.Stop]*shipment.ShipmentMove, len(ordered))
	for _, move := range entity.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop != nil {
				moveOf[stop] = move
			}
		}
	}

	out := make(map[*shipment.Stop]decimal.Decimal, len(ordered))
	for i := 1; i < len(ordered); i++ {
		previous, stop := ordered[i-1], ordered[i]
		move := moveOf[stop]
		if move == moveOf[previous] {
			if distance, ok := singleLegDistance(move); ok {
				out[stop] = distance
			}
			continue
		}
		if stop.LocationID.IsNotNil() && stop.LocationID == previous.LocationID {
			out[stop] = decimal.Zero
		}
	}
	return out
}

func singleLegDistance(move *shipment.ShipmentMove) (decimal.Decimal, bool) {
	if move == nil || move.IsCanceled() || move.Distance == nil || *move.Distance < 0 {
		return decimal.Zero, false
	}
	stops := 0
	for _, stop := range move.Stops {
		if stop != nil {
			stops++
		}
	}
	if stops != 2 {
		return decimal.Zero, false
	}
	return decimal.NewFromFloat(*move.Distance), true
}

func buildAccessorials(
	charges []*shipment.AdditionalCharge,
	codes map[pulid.ID]string,
) []*accessorialRecord {
	out := make([]*accessorialRecord, 0, len(charges))
	for _, charge := range charges {
		if charge == nil || charge.IsSystemGenerated || charge.IsTariff {
			continue
		}
		code := codes[charge.AccessorialChargeID]
		if item := charge.AccessorialCharge; item != nil &&
			item.ID == charge.AccessorialChargeID {
			code = item.Code
		}
		code = accessorialcharge.NormalizeCode(code)
		if code == "" {
			continue
		}
		out = append(out, &accessorialRecord{
			Code:     code,
			Quantity: rules.IntValue(int64(charge.Unit)),
		})
	}
	return out
}

func buildCommodity(line *shipment.ShipmentCommodity) *commodityRecord {
	weight := decimal.NewFromInt(line.Weight)
	out := &commodityRecord{
		Weight: rules.QuantityValue(weight, unitWeight),
		Pieces: rules.IntValue(line.Pieces),
		Dimensions: dimensions{
			Length: optionalFeet(line.ResolvedLengthFeet()),
			Width:  optionalFeet(line.ResolvedWidthFeet()),
			Height: optionalFeet(line.ResolvedHeightFeet()),
		},
		Volume:       rules.NoneValue(),
		Density:      rules.NoneValue(),
		FreightClass: rules.NoneValue(),
	}
	if line.HasDimensions() && line.Pieces > 0 {
		volume := line.CubicFeet()
		out.Volume = rules.QuantityValue(volume, unitVolume)
		out.Density = density(decimal.NewNullDecimal(weight), volume)
	}
	if item := line.Commodity; item != nil {
		out.Name = item.Name
		out.FreightClass = freightClass(item.FreightClass)
		out.Hazmat = item.HazardousMaterialID.IsNotNil() || item.HazardousMaterial != nil
		out.Stackable = item.Stackable
	}
	return out
}

func buildLane(origin, destination *Place) *laneRoot {
	return &laneRoot{
		Origin:      lanePlaceOf(origin),
		Destination: lanePlaceOf(destination),
		Interstate:  differs(origin.State, destination.State),
		CrossBorder: differs(origin.Country, destination.Country),
	}
}

func lanePlaceOf(place *Place) lanePlace {
	zones := place.Zones
	if zones == nil {
		zones = []string{}
	}
	return lanePlace{
		City:    place.City,
		State:   place.State,
		Zip:     place.Zip,
		Zip3:    Zip3(place.Zip),
		Country: place.Country,
		Zones:   zones,
	}
}

func buildCustomer(entity *shipment.Shipment) *customerRoot {
	if entity.Customer == nil {
		return &customerRoot{}
	}
	return &customerRoot{Name: entity.Customer.Name, Code: entity.Customer.Code}
}

func buildFuel(fuel *Fuel) *fuelRoot {
	out := &fuelRoot{
		Price:     rules.NoneValue(),
		PriceDate: rules.NoneValue(),
		Index:     rules.NoneValue(),
	}
	if !fuel.Price.Valid {
		return out
	}
	out.Price = rules.QuantityValue(fuel.Price.Decimal, unitFuelPrice)
	if date, err := time.Parse(time.DateOnly, fuel.PriceDate); err == nil {
		out.PriceDate = rules.DateValue(date)
	}
	if fuel.Index != "" {
		out.Index = rules.TextValue(fuel.Index)
	}
	return out
}

func buildRating(rating *Rating, holidays []Holiday) *ratingRoot {
	mode := rating.Mode
	if mode != ModeBuy {
		mode = ModeSell
	}
	out := &ratingRoot{
		Date:      rules.DateTimeValue(rating.At),
		Mode:      mode,
		SellTotal: rules.NoneValue(),
		Currency:  rating.Currency,
		Holidays:  make([]*holidayRecord, 0, len(holidays)),
	}
	for i := range holidays {
		out.Holidays = append(out.Holidays, &holidayRecord{
			Date: rules.DateValue(holidays[i].Date),
			Name: holidays[i].Name,
		})
	}
	if rating.SellTotal.Valid {
		out.SellTotal = rules.MoneyValue(rating.SellTotal.Decimal)
	}
	return out
}

func differs(a, b string) bool {
	return a != "" && b != "" && !strings.EqualFold(a, b)
}

func optionalQuantity(value decimal.NullDecimal, unit string) rules.RawJSON {
	if !value.Valid {
		return rules.NoneValue()
	}
	return rules.QuantityValue(value.Decimal, unit)
}

func optionalInt(value *int64) rules.RawJSON {
	if value == nil {
		return rules.NoneValue()
	}
	return rules.IntValue(*value)
}

func optionalFeet(value float64) rules.RawJSON {
	if value <= 0 {
		return rules.NoneValue()
	}
	return rules.QuantityValue(decimal.NewFromFloat(value), unitLength)
}

func temperatureValue(value *int16) rules.RawJSON {
	if value == nil {
		return rules.NoneValue()
	}
	return rules.QuantityValue(decimal.NewFromInt(int64(*value)), unitTemperature)
}

func density(weight decimal.NullDecimal, volume decimal.Decimal) rules.RawJSON {
	if !weight.Valid || !volume.IsPositive() {
		return rules.NoneValue()
	}
	return rules.QuantityValue(weight.Decimal.Div(volume), unitDensity)
}

func freightClass(class commodity.FreightClass) rules.RawJSON {
	if !class.IsValid() {
		return rules.NoneValue()
	}
	return rules.TextValue(string(class))
}

func optionalTime(value *int64, zone *time.Location) rules.RawJSON {
	if value == nil || *value <= 0 {
		return rules.NoneValue()
	}
	return timeValue(*value, zone)
}

func timeValue(value int64, zone *time.Location) rules.RawJSON {
	return rules.DateTimeValue(time.Unix(value, 0).In(zone))
}

func orNew(status, unsaved string) string {
	if status == "" {
		return unsaved
	}
	return status
}
