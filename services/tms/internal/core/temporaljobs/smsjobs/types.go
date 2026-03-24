package smsjobs

import "github.com/emoss08/trenova/shared/pulid"

type SendSMSPayload struct {
	PhoneNumber    string   `json:"phoneNumber"`
	Message        string   `json:"message"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

type SendSMSResult struct {
	MessageID string `json:"messageId,omitempty"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}
