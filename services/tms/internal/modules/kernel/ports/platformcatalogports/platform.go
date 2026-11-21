package platformcatalogports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

type UsageLimitCheckRequest struct {
	OrganizationID pulid.ID                  `json:"organizationId"`
	BusinessUnitID pulid.ID                  `json:"businessUnitId"`
	PrincipalType  kernelports.PrincipalType `json:"principalType"`
	PrincipalID    pulid.ID                  `json:"principalId"`
	UserID         pulid.ID                  `json:"userId"`
	APIKeyID       pulid.ID                  `json:"apiKeyId"`
	MeterKey       platformcatalog.MeterKey  `json:"meterKey"`
	Quantity       int64                     `json:"quantity"`
	CheckedAt      int64                     `json:"checkedAt"`
	IdempotencyKey string                    `json:"idempotencyKey,omitempty"`
}

type UsageLimitCheckResult struct {
	MeterKey   platformcatalog.MeterKey `json:"meterKey"`
	Allowed    bool                     `json:"allowed"`
	Reason     string                   `json:"reason,omitempty"`
	Limit      int64                    `json:"limit,omitempty"`
	Used       int64                    `json:"used,omitempty"`
	Remaining  int64                    `json:"remaining,omitempty"`
	Ceiling    int64                    `json:"ceiling,omitempty"`
	CeilingHit bool                     `json:"ceilingHit,omitempty"`
	Plan       string                   `json:"plan,omitempty"`
	CheckedAt  int64                    `json:"checkedAt"`
	FailOpen   bool                     `json:"failOpen"`
}

type UsageRecordRequest struct {
	OrganizationID pulid.ID                  `json:"organizationId"`
	BusinessUnitID pulid.ID                  `json:"businessUnitId"`
	PrincipalType  kernelports.PrincipalType `json:"principalType"`
	PrincipalID    pulid.ID                  `json:"principalId"`
	UserID         pulid.ID                  `json:"userId"`
	APIKeyID       pulid.ID                  `json:"apiKeyId"`
	MeterKey       platformcatalog.MeterKey  `json:"meterKey"`
	Quantity       int64                     `json:"quantity"`
	RecordedAt     int64                     `json:"recordedAt"`
	IdempotencyKey string                    `json:"idempotencyKey,omitempty"`
}

type UsageRecordResult struct {
	MeterKey       platformcatalog.MeterKey `json:"meterKey"`
	Recorded       bool                     `json:"recorded"`
	Quantity       int64                    `json:"quantity"`
	RecordedAt     int64                    `json:"recordedAt"`
	IdempotencyKey string                   `json:"idempotencyKey,omitempty"`
}

type UsageProvider interface {
	CheckLimit(context.Context, *UsageLimitCheckRequest) (*UsageLimitCheckResult, error)
	RecordUsage(context.Context, *UsageRecordRequest) (*UsageRecordResult, error)
}
