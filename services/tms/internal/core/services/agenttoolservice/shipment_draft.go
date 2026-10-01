package agenttoolservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type shipmentDraft struct {
	CustomerID        string            `json:"customerId"`
	BillToCustomerID  string            `json:"billToCustomerId"`
	ServiceTypeID     string            `json:"serviceTypeId"`
	ShipmentTypeID    string            `json:"shipmentTypeId"`
	FormulaTemplateID string            `json:"formulaTemplateId"`
	BaseRate          string            `json:"baseRate"`
	FreightTerms      string            `json:"freightTerms"`
	TractorTypeID     string            `json:"tractorTypeId"`
	TrailerTypeID     string            `json:"trailerTypeId"`
	BOL               string            `json:"bol"`
	ExternalReference string            `json:"externalReference"`
	Pieces            *int64            `json:"pieces"`
	Weight            *int64            `json:"weight"`
	TemperatureMin    *int16            `json:"temperatureMin"`
	TemperatureMax    *int16            `json:"temperatureMax"`
	RatingUnit        *int64            `json:"ratingUnit"`
	Moves             []moveDraft       `json:"moves"`
	Commodities       []commodityDraft  `json:"commodities"`
	AdditionalCharges []chargeLineDraft `json:"additionalCharges"`
}

type moveDraft struct {
	Loaded   *bool       `json:"loaded"`
	Sequence *int64      `json:"sequence"`
	Stops    []stopDraft `json:"stops"`
}

type stopDraft struct {
	LocationID           string `json:"locationId"`
	Type                 string `json:"type"`
	ScheduleType         string `json:"scheduleType"`
	Sequence             *int64 `json:"sequence"`
	ScheduledWindowStart string `json:"scheduledWindowStart"`
	ScheduledWindowEnd   string `json:"scheduledWindowEnd"`
	Pieces               *int64 `json:"pieces"`
	Weight               *int64 `json:"weight"`
	AddressLine          string `json:"addressLine"`
}

type commodityDraft struct {
	CommodityID string `json:"commodityId"`
	Pieces      *int64 `json:"pieces"`
	Weight      *int64 `json:"weight"`
}

type chargeLineDraft struct {
	AccessorialChargeID string `json:"accessorialChargeId"`
	Unit                *int16 `json:"unit"`
	Method              string `json:"method"`
	Amount              string `json:"amount"`
}

func (d *shipmentDraft) locationIDs() []pulid.ID {
	ids := make([]pulid.ID, 0, 2*len(d.Moves))
	seen := make(map[pulid.ID]struct{}, cap(ids))
	for _, move := range d.Moves {
		for _, stop := range move.Stops {
			id, err := pulid.Parse(strings.TrimSpace(stop.LocationID))
			if err != nil || id.IsNil() {
				continue
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}

	return ids
}

type draftReader struct {
	multiErr *errortypes.MultiError
}

func (r draftReader) id(field, value string) pulid.ID {
	text := strings.TrimSpace(value)
	if text == "" {
		return pulid.Nil
	}

	id, err := pulid.Parse(text)
	if err != nil {
		r.multiErr.Add(field, errortypes.ErrInvalid, "{0} is not an id", text)

		return pulid.Nil
	}

	return id
}

func (r draftReader) requiredID(field, value string) pulid.ID {
	if strings.TrimSpace(value) == "" {
		r.multiErr.Add(field, errortypes.ErrRequired, "An id is required")

		return pulid.Nil
	}

	return r.id(field, value)
}

func (r draftReader) decimal(field, value string) decimal.NullDecimal {
	text := strings.TrimSpace(value)
	if text == "" {
		return decimal.NullDecimal{}
	}

	parsed, err := decimal.NewFromString(text)
	if err != nil {
		r.multiErr.Add(field, errortypes.ErrInvalid,
			"{0} is not a decimal such as 1250.00", text)

		return decimal.NullDecimal{}
	}

	return decimal.NewNullDecimal(parsed)
}

func (r draftReader) sequence(field string, given *int64, position int) {
	if given == nil || *given == int64(position) {
		return
	}

	r.multiErr.Add(field, errortypes.ErrInvalid,
		"Sequences count from 0 in travel order, so this one is {0}, not {1}; "+
			"or leave sequence out", position, *given)
}

func (d *shipmentDraft) build(
	tenant pagination.TenantInfo,
	zones locationZones,
) (*shipment.Shipment, error) {
	multiErr := errortypes.NewMultiError()
	read := draftReader{multiErr: multiErr}

	entity := &shipment.Shipment{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		EnteredByID:       tenant.UserID,
		Status:            shipment.StatusNew,
		CustomerID:        read.requiredID("customerId", d.CustomerID),
		ServiceTypeID:     read.requiredID("serviceTypeId", d.ServiceTypeID),
		ShipmentTypeID:    read.requiredID("shipmentTypeId", d.ShipmentTypeID),
		FormulaTemplateID: read.requiredID("formulaTemplateId", d.FormulaTemplateID),
		TractorTypeID:     read.id("tractorTypeId", d.TractorTypeID),
		TrailerTypeID:     read.id("trailerTypeId", d.TrailerTypeID),
		BaseRate:          read.decimal("baseRate", d.BaseRate),
		BOL:               strings.TrimSpace(d.BOL),
		ExternalReference: strings.TrimSpace(d.ExternalReference),
		Pieces:            d.Pieces,
		Weight:            d.Weight,
		TemperatureMin:    d.TemperatureMin,
		TemperatureMax:    d.TemperatureMax,
		RatingUnit:        1,
	}
	if billTo := read.id("billToCustomerId", d.BillToCustomerID); billTo.IsNotNil() {
		entity.BillToCustomerID = &billTo
	}
	if d.RatingUnit != nil {
		entity.RatingUnit = *d.RatingUnit
	}
	if terms := strings.TrimSpace(d.FreightTerms); terms != "" {
		entity.FreightTerms = shipment.FreightTerms(terms)
		if !entity.FreightTerms.IsValid() {
			multiErr.Add("freightTerms", errortypes.ErrInvalid,
				"{0} is not one of Prepaid, Collect or ThirdParty", terms)
		}
	}

	d.buildMoves(entity, read, zones)
	d.buildCommodities(entity, read)
	d.buildCharges(entity, read)

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return entity, nil
}

func (d *shipmentDraft) buildMoves(
	entity *shipment.Shipment,
	read draftReader,
	zones locationZones,
) {
	if len(d.Moves) == 0 {
		read.multiErr.Add("moves", errortypes.ErrRequired,
			"A shipment needs a move with its stops in travel order")

		return
	}

	entity.Moves = make([]*shipment.ShipmentMove, 0, len(d.Moves))
	for moveIdx := range d.Moves {
		draft := &d.Moves[moveIdx]
		path := fmt.Sprintf("moves[%d]", moveIdx)
		read.sequence(path+".sequence", draft.Sequence, moveIdx)

		move := &shipment.ShipmentMove{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			Status:         shipment.MoveStatusNew,
			Loaded:         draft.Loaded == nil || *draft.Loaded,
			Sequence:       int64(moveIdx),
			Stops:          make([]*shipment.Stop, 0, len(draft.Stops)),
		}
		for stopIdx := range draft.Stops {
			stop := buildStop(&draft.Stops[stopIdx], stopBuild{
				path:     fmt.Sprintf("%s.stops[%d]", path, stopIdx),
				position: stopIdx,
				read:     read,
				zones:    zones,
				tenant:   entity,
			})
			move.Stops = append(move.Stops, stop)
		}
		entity.Moves = append(entity.Moves, move)
	}
}

type stopBuild struct {
	path     string
	position int
	read     draftReader
	zones    locationZones
	tenant   *shipment.Shipment
}

func buildStop(draft *stopDraft, b stopBuild) *shipment.Stop {
	b.read.sequence(b.path+".sequence", draft.Sequence, b.position)

	stop := &shipment.Stop{
		OrganizationID: b.tenant.OrganizationID,
		BusinessUnitID: b.tenant.BusinessUnitID,
		LocationID:     b.read.requiredID(b.path+".locationId", draft.LocationID),
		Status:         shipment.StopStatusNew,
		Type:           shipment.StopType(strings.TrimSpace(draft.Type)),
		ScheduleType:   shipment.StopScheduleType(strings.TrimSpace(draft.ScheduleType)),
		Sequence:       int64(b.position),
		Pieces:         draft.Pieces,
		Weight:         draft.Weight,
		AddressLine:    strings.TrimSpace(draft.AddressLine),
	}
	if stop.ScheduleType == "" {
		stop.ScheduleType = shipment.StopScheduleTypeOpen
	}

	zone := b.zones.at(stop.LocationID)
	if strings.TrimSpace(draft.ScheduledWindowStart) == "" {
		b.read.multiErr.Add(b.path+".scheduledWindowStart", errortypes.ErrRequired,
			"A local date and time such as 2026-10-01T08:00 is required")
	} else if start, err := parseLocalTime(
		b.path+".scheduledWindowStart", draft.ScheduledWindowStart, zone,
	); err != nil {
		b.read.multiErr.AddError(err)
	} else {
		stop.ScheduledWindowStart = start
	}

	if strings.TrimSpace(draft.ScheduledWindowEnd) != "" {
		end, err := parseLocalTime(b.path+".scheduledWindowEnd", draft.ScheduledWindowEnd, zone)
		if err != nil {
			b.read.multiErr.AddError(err)
		} else {
			stop.ScheduledWindowEnd = &end
		}
	}

	return stop
}

func (d *shipmentDraft) buildCommodities(entity *shipment.Shipment, read draftReader) {
	if len(d.Commodities) == 0 {
		return
	}

	entity.Commodities = make([]*shipment.ShipmentCommodity, 0, len(d.Commodities))
	for idx := range d.Commodities {
		draft := &d.Commodities[idx]
		line := &shipment.ShipmentCommodity{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			CommodityID: read.requiredID(
				fmt.Sprintf("commodities[%d].commodityId", idx), draft.CommodityID,
			),
			Pieces: 1,
		}
		if draft.Pieces != nil {
			line.Pieces = *draft.Pieces
		}
		if draft.Weight != nil {
			line.Weight = *draft.Weight
		}
		entity.Commodities = append(entity.Commodities, line)
	}
}

func (d *shipmentDraft) buildCharges(entity *shipment.Shipment, read draftReader) {
	if len(d.AdditionalCharges) == 0 {
		return
	}

	entity.AdditionalCharges = make([]*shipment.AdditionalCharge, 0, len(d.AdditionalCharges))
	for idx := range d.AdditionalCharges {
		draft := &d.AdditionalCharges[idx]
		path := fmt.Sprintf("additionalCharges[%d]", idx)
		charge := &shipment.AdditionalCharge{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			AccessorialChargeID: read.requiredID(
				path+".accessorialChargeId",
				draft.AccessorialChargeID,
			),
			Method: accessorialcharge.Method(strings.TrimSpace(draft.Method)),
			Unit:   1,
		}
		if draft.Unit != nil {
			charge.Unit = *draft.Unit
		}
		if amount := read.decimal(path+".amount", draft.Amount); amount.Valid {
			charge.Amount = amount.Decimal
		}
		entity.AdditionalCharges = append(entity.AdditionalCharges, charge)
	}
}
