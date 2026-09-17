package carrierokconnector

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/intelkit"
	"github.com/emoss08/trenova/shared/carrierok"
	"github.com/emoss08/trenova/shared/stringutils"
)

const messageEquipmentIdentifier = "equipment lookup requires a VIN, plate number, or unit number"

func (c *Client) FindByEquipment(
	ctx context.Context,
	req *services.CarrierIntelEquipmentRequest,
) (*services.CarrierIntelEquipmentResult, error) {
	if req == nil {
		return nil, errorMapper.InvalidRequest(messageEquipmentIdentifier)
	}
	query, ok := equipmentQuery(req)
	if !ok {
		return nil, errorMapper.InvalidRequest(messageEquipmentIdentifier)
	}

	outcome := &intelkit.CallOutcome{
		Endpoint: carrierintel.EndpointEquipment,
		Started:  time.Now(),
		Units:    1,
	}
	profiles, err := c.sdk.Profiles(ctx, query)
	if err == nil && len(profiles) == 0 {
		err = carrierok.ErrNotFound
	}
	if err != nil {
		outcome.RawErr = err
		outcome.Mapped = errorMapper.Map(err)
		c.recorder.Record(ctx, outcome)
		return nil, outcome.Mapped
	}

	matches := make([]services.CarrierIntelEquipmentMatch, 0, len(profiles))
	raws := make([][]byte, 0, len(profiles))
	for idx := range profiles {
		profile := normalizeProfile(&profiles[idx])
		legalName := ""
		if profile.Identity != nil {
			legalName = profile.Identity.LegalName
		}
		matches = append(matches, services.CarrierIntelEquipmentMatch{
			DOTNumber: profile.DOTNumber(),
			LegalName: legalName,
			Unit:      matchingUnit(profile.Equipment, req),
			Profile:   profile,
		})
		if len(profiles[idx].Raw) > 0 {
			raws = append(raws, profiles[idx].Raw)
		}
	}

	outcome.Found = true
	outcome.Units = len(matches)
	if len(matches) == 1 {
		outcome.DOTNumber = matches[0].DOTNumber
	}
	c.recorder.Record(ctx, outcome)

	return &services.CarrierIntelEquipmentResult{
		Matches: matches,
		Raw:     joinRaw(raws),
	}, nil
}

func equipmentQuery(req *services.CarrierIntelEquipmentRequest) (carrierok.ProfileQuery, bool) {
	vin := strings.TrimSpace(req.VIN)
	plate := strings.TrimSpace(req.PlateNumber)
	unit := strings.TrimSpace(req.UnitNumber)
	switch {
	case vin != "":
		return carrierok.ProfileQuery{VIN: vin}, true
	case plate != "":
		return carrierok.ProfileQuery{
			PlateNumber: plate,
			PlateState:  strings.TrimSpace(req.PlateState),
		}, true
	case unit != "":
		return carrierok.ProfileQuery{
			UnitNumber: unit,
			UnitType:   vendorUnitType(req.UnitType),
		}, true
	default:
		return carrierok.ProfileQuery{}, false
	}
}

func vendorUnitType(unitType carrierintel.UnitType) string {
	switch unitType {
	case carrierintel.UnitTypeTractor, carrierintel.UnitTypeStraight:
		return carrierok.UnitTypeTruck
	case carrierintel.UnitTypeTrailer:
		return carrierok.UnitTypeTrailer
	default:
		return ""
	}
}

func matchingUnit(
	equipment []carrierintel.Equipment,
	req *services.CarrierIntelEquipmentRequest,
) *carrierintel.Equipment {
	vin := stringutils.NormalizeIdentifier(req.VIN)
	plate := stringutils.NormalizeIdentifier(req.PlateNumber)
	plateState := strings.ToUpper(strings.TrimSpace(req.PlateState))
	unit := stringutils.NormalizeIdentifier(req.UnitNumber)

	for idx := range equipment {
		item := &equipment[idx]
		switch {
		case vin != "":
			if stringutils.NormalizeIdentifier(item.VIN) == vin {
				return cloneEquipment(item)
			}
		case plate != "":
			if stringutils.NormalizeIdentifier(item.PlateNumber) == plate &&
				(plateState == "" || item.PlateState == "" ||
					strings.EqualFold(item.PlateState, plateState)) {
				return cloneEquipment(item)
			}
		case unit != "":
			if stringutils.NormalizeIdentifier(item.UnitNumber) == unit &&
				unitTypeCompatible(item.UnitType, req.UnitType) {
				return cloneEquipment(item)
			}
		}
	}
	return nil
}

func unitTypeCompatible(have, want carrierintel.UnitType) bool {
	if have == "" || want == "" {
		return true
	}
	return vendorUnitType(have) == vendorUnitType(want)
}

func cloneEquipment(item *carrierintel.Equipment) *carrierintel.Equipment {
	clone := *item
	if item.Year != nil {
		year := *item.Year
		clone.Year = &year
	}
	return &clone
}

func joinRaw(raws [][]byte) []byte {
	if len(raws) == 0 {
		return []byte("[]")
	}
	size := 2 + len(raws) - 1
	for _, raw := range raws {
		size += len(raw)
	}
	var buf bytes.Buffer
	buf.Grow(size)
	buf.WriteByte('[')
	for idx, raw := range raws {
		if idx > 0 {
			buf.WriteByte(',')
		}
		buf.Write(bytes.TrimSpace(raw))
	}
	buf.WriteByte(']')
	return buf.Bytes()
}
