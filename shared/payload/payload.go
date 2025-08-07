package payload

import "github.com/emoss08/trenova/shared/pulid"

type JobBasePayload struct {
	JobID          string         `json:"jobId"`
	OrganizationID pulid.ID       `json:"organizationId"`
	BusinessUnitID pulid.ID       `json:"businessUnitId"`
	UserID         pulid.ID       `json:"userId,omitempty"`
	Timestamp      int64          `json:"timestamp"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}
