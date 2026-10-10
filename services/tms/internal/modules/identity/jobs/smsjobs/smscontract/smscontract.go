package smscontract

import (
	"github.com/emoss08/trenova/shared/pulid"
)

type SendSMSPayload struct {
	PhoneNumber    string   `json:"phoneNumber"`
	Message        string   `json:"message"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}
