/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
	JobID      string         `json:"jobId"`
	Count      int            `json:"count"`
	ProNumbers []string       `json:"proNumbers"`
	Result     string         `json:"result"`
	Data       map[string]any `json:"data"`
}

type OrgCancellationResult struct {
	OrganizationID   pulid.ID `json:"organizationId"`
	BusinessUnitID   pulid.ID `json:"businessUnitId"`
	CancelledCount   int      `json:"cancelledCount"`
	CancelledProNums []string `json:"cancelledProNums"`
}

type CancelShipmentsByCreatedAtResult struct {
	JobID          string                  `json:"jobId"`
	TotalCancelled int                     `json:"totalCancelled"`
	SkippedOrgs    []pulid.ID              `json:"skippedOrgs"`
	ProcessedOrgs  []OrgCancellationResult `json:"processedOrgs"`
	Result         string                  `json:"result"`
	Data           map[string]any          `json:"data"`
}
