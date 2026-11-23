package vehiclematch

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/shared/pulid"
)

type Candidate struct {
	UnitID     pulid.ID
	ExternalID string
	VIN        string
	Code       string
}

type Assignment struct {
	UnitID     pulid.ID
	ExternalID string
}

func Match(
	candidates []Candidate,
	vehicles []telematicsports.ProviderVehicle,
	unmatched *int,
) []Assignment {
	assignedExternalIDs := make(map[string]struct{}, len(candidates))
	unmappedByVin := make(map[string]pulid.ID)
	unmappedByCode := make(map[string]pulid.ID)
	for _, candidate := range candidates {
		if candidate.ExternalID != "" {
			assignedExternalIDs[candidate.ExternalID] = struct{}{}
			continue
		}
		if vin := strings.ToUpper(strings.TrimSpace(candidate.VIN)); vin != "" {
			unmappedByVin[vin] = candidate.UnitID
		}
		if code := strings.ToLower(strings.TrimSpace(candidate.Code)); code != "" {
			unmappedByCode[code] = candidate.UnitID
		}
	}

	if len(unmappedByVin) == 0 && len(unmappedByCode) == 0 {
		return nil
	}

	matches := make([]Assignment, 0)
	claimedUnits := make(map[pulid.ID]struct{})
	for i := range vehicles {
		vehicle := &vehicles[i]
		if _, taken := assignedExternalIDs[vehicle.ID]; taken {
			continue
		}

		unitID, matched := matchVehicleToUnit(vehicle, unmappedByVin, unmappedByCode)
		if !matched {
			if unmatched != nil {
				*unmatched++
			}
			continue
		}
		if _, claimed := claimedUnits[unitID]; claimed {
			continue
		}
		claimedUnits[unitID] = struct{}{}
		assignedExternalIDs[vehicle.ID] = struct{}{}
		matches = append(matches, Assignment{
			UnitID:     unitID,
			ExternalID: vehicle.ID,
		})
	}
	return matches
}

func matchVehicleToUnit(
	vehicle *telematicsports.ProviderVehicle,
	unmappedByVin map[string]pulid.ID,
	unmappedByCode map[string]pulid.ID,
) (pulid.ID, bool) {
	if vin := strings.ToUpper(strings.TrimSpace(vehicle.VIN)); vin != "" {
		if unitID, ok := unmappedByVin[vin]; ok {
			return unitID, true
		}
	}
	if name := strings.ToLower(strings.TrimSpace(vehicle.Name)); name != "" {
		if unitID, ok := unmappedByCode[name]; ok {
			return unitID, true
		}
	}
	return pulid.Nil, false
}
