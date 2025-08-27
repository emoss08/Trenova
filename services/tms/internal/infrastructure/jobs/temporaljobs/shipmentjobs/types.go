package shipmentjobs

import (
	"github.com/emoss08/trenova/pkg/types/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

type DuplicateShipmentPayload struct {
	temporaltype.BasePayload
	ShipmentID               pulid.ID `json:"shipmentId"`
	Count                    int      `json:"count"`
	OverrideDates            bool     `json:"overrideDates"`
	IncludeCommodities       bool     `json:"includeCommodities"`
	IncludeAdditionalCharges bool     `json:"includeAdditionalCharges"`
}

type DuplicateShipmentResult struct {
	Count      int      `json:"count"`
	ProNumbers []string `json:"proNumbers"`
}
