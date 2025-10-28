package shipmentjobs

import (
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
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
	Count       int            `json:"count"`
	ShipmentIDs []pulid.ID     `json:"shipmentIds"`
	ProNumbers  []string       `json:"proNumbers"`
	Result      string         `json:"result"`
	Data        map[string]any `json:"data"`
}
