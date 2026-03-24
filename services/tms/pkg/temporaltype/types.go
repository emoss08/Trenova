package temporaltype

import (
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
)

type BasePayload struct {
	OrganizationID pulid.ID       `json:"organizationId"`
	BusinessUnitID pulid.ID       `json:"businessUnitId"`
	UserID         pulid.ID       `json:"userId"`
	Timestamp      int64          `json:"timestamp"`
	Metadata       map[string]any `json:"metadata"`
}

func (p *BasePayload) GetOrganizationID() pulid.ID {
	return p.OrganizationID
}

func (p *BasePayload) GetBusinessUnitID() pulid.ID {
	return p.BusinessUnitID
}

type WorkflowDefinition struct {
	Name        string
	Fn          any
	TaskQueue   string
	Description string
}

var DefaultRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumInterval:    time.Second * 100,
}
