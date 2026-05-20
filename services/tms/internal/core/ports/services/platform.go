package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

type FeatureCheckRequest struct {
	OrganizationID pulid.ID                   `json:"organizationId"`
	BusinessUnitID pulid.ID                   `json:"businessUnitId"`
	PrincipalType  PrincipalType              `json:"principalType"`
	PrincipalID    pulid.ID                   `json:"principalId"`
	UserID         pulid.ID                   `json:"userId"`
	APIKeyID       pulid.ID                   `json:"apiKeyId"`
	FeatureKey     platformcatalog.FeatureKey `json:"featureKey"`
	CheckedAt      int64                      `json:"checkedAt"`
}

type AccessAuthorizeRequest struct {
	OrganizationID pulid.ID                   `json:"organizationId"`
	BusinessUnitID pulid.ID                   `json:"businessUnitId"`
	PrincipalType  PrincipalType              `json:"principalType"`
	PrincipalID    pulid.ID                   `json:"principalId"`
	UserID         pulid.ID                   `json:"userId"`
	APIKeyID       pulid.ID                   `json:"apiKeyId"`
	HTTPMethod     string                     `json:"httpMethod"`
	HTTPPath       string                     `json:"httpPath"`
	RoutePattern   string                     `json:"routePattern"`
	FeatureKey     platformcatalog.FeatureKey `json:"featureKey"`
	CheckedAt      int64                      `json:"checkedAt"`
}

type AccessAuthorizeResult struct {
	FeatureKey platformcatalog.FeatureKey `json:"featureKey"`
	Allowed    bool                       `json:"allowed"`
	Reason     string                     `json:"reason,omitempty"`
	CheckedAt  int64                      `json:"checkedAt"`
	FailOpen   bool                       `json:"failOpen"`
}

type FeatureCheckResult struct {
	FeatureKey platformcatalog.FeatureKey `json:"featureKey"`
	Allowed    bool                       `json:"allowed"`
	Reason     string                     `json:"reason,omitempty"`
	CheckedAt  int64                      `json:"checkedAt"`
	FailOpen   bool                       `json:"failOpen"`
}

type EntitlementsRequest struct {
	OrganizationID pulid.ID      `json:"organizationId"`
	BusinessUnitID pulid.ID      `json:"businessUnitId"`
	PrincipalType  PrincipalType `json:"principalType"`
	PrincipalID    pulid.ID      `json:"principalId"`
	UserID         pulid.ID      `json:"userId"`
	APIKeyID       pulid.ID      `json:"apiKeyId"`
	CheckedAt      int64         `json:"checkedAt"`
}

type EntitlementsResult struct {
	Features  []FeatureCheckResult `json:"features"`
	CheckedAt int64                `json:"checkedAt"`
}

type BillingSummaryRequest struct {
	OrganizationID pulid.ID      `json:"organizationId"`
	BusinessUnitID pulid.ID      `json:"businessUnitId"`
	PrincipalType  PrincipalType `json:"principalType"`
	PrincipalID    pulid.ID      `json:"principalId"`
	UserID         pulid.ID      `json:"userId"`
	APIKeyID       pulid.ID      `json:"apiKeyId"`
	CheckedAt      int64         `json:"checkedAt"`
}

type BillingPlanSummary struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type BillingSubscriptionSummary struct {
	ID                 string `json:"id"`
	PlanID             string `json:"planId"`
	Status             string `json:"status"`
	CurrentPeriodStart int64  `json:"currentPeriodStart"`
	CurrentPeriodEnd   int64  `json:"currentPeriodEnd"`
}

type BillingFeatureSummary struct {
	FeatureKey platformcatalog.FeatureKey `json:"featureKey"`
	Allowed    bool                       `json:"allowed"`
}

type BillingUsageSummary struct {
	MeterKey    platformcatalog.MeterKey `json:"meterKey"`
	Unit        string                   `json:"unit"`
	Limit       int64                    `json:"limit"`
	Used        int64                    `json:"used"`
	Remaining   int64                    `json:"remaining"`
	WindowStart int64                    `json:"windowStart"`
	WindowEnd   int64                    `json:"windowEnd"`
}

type BillingSummaryResult struct {
	BusinessUnitID pulid.ID                    `json:"businessUnitId"`
	OrganizationID pulid.ID                    `json:"organizationId"`
	Active         bool                        `json:"active"`
	Reason         string                      `json:"reason,omitempty"`
	Plan           *BillingPlanSummary         `json:"plan,omitempty"`
	Subscription   *BillingSubscriptionSummary `json:"subscription,omitempty"`
	Features       []BillingFeatureSummary     `json:"features"`
	Usage          []BillingUsageSummary       `json:"usage"`
	CheckedAt      int64                       `json:"checkedAt"`
}

type UsageLimitCheckRequest struct {
	OrganizationID pulid.ID                 `json:"organizationId"`
	BusinessUnitID pulid.ID                 `json:"businessUnitId"`
	PrincipalType  PrincipalType            `json:"principalType"`
	PrincipalID    pulid.ID                 `json:"principalId"`
	UserID         pulid.ID                 `json:"userId"`
	APIKeyID       pulid.ID                 `json:"apiKeyId"`
	MeterKey       platformcatalog.MeterKey `json:"meterKey"`
	Quantity       int64                    `json:"quantity"`
	CheckedAt      int64                    `json:"checkedAt"`
	IdempotencyKey string                   `json:"idempotencyKey,omitempty"`
}

type UsageLimitCheckResult struct {
	MeterKey  platformcatalog.MeterKey `json:"meterKey"`
	Allowed   bool                     `json:"allowed"`
	Reason    string                   `json:"reason,omitempty"`
	Limit     int64                    `json:"limit,omitempty"`
	Used      int64                    `json:"used,omitempty"`
	Remaining int64                    `json:"remaining,omitempty"`
	CheckedAt int64                    `json:"checkedAt"`
	FailOpen  bool                     `json:"failOpen"`
}

type UsageRecordRequest struct {
	OrganizationID pulid.ID                 `json:"organizationId"`
	BusinessUnitID pulid.ID                 `json:"businessUnitId"`
	PrincipalType  PrincipalType            `json:"principalType"`
	PrincipalID    pulid.ID                 `json:"principalId"`
	UserID         pulid.ID                 `json:"userId"`
	APIKeyID       pulid.ID                 `json:"apiKeyId"`
	MeterKey       platformcatalog.MeterKey `json:"meterKey"`
	Quantity       int64                    `json:"quantity"`
	RecordedAt     int64                    `json:"recordedAt"`
	IdempotencyKey string                   `json:"idempotencyKey,omitempty"`
}

type UsageRecordResult struct {
	MeterKey       platformcatalog.MeterKey `json:"meterKey"`
	Recorded       bool                     `json:"recorded"`
	Quantity       int64                    `json:"quantity"`
	RecordedAt     int64                    `json:"recordedAt"`
	IdempotencyKey string                   `json:"idempotencyKey,omitempty"`
}

type InstanceHeartbeatRequest struct {
	InstanceID     string                    `json:"instanceId"`
	AppVersion     string                    `json:"appVersion"`
	DeploymentMode string                    `json:"deploymentMode"`
	Metadata       map[string]string         `json:"metadata"`
	CatalogHash    string                    `json:"catalogHash"`
	Products       []platformcatalog.Product `json:"products"`
	Features       []platformcatalog.Feature `json:"features"`
	Meters         []platformcatalog.Meter   `json:"meters"`
	SentAt         int64                     `json:"sentAt"`
}

type InstanceHeartbeatResult struct {
	Accepted   bool   `json:"accepted"`
	InstanceID string `json:"instanceId"`
	ReceivedAt int64  `json:"receivedAt"`
}

type TenantSyncMode string

const (
	TenantSyncModeFull  TenantSyncMode = "full"
	TenantSyncModeDelta TenantSyncMode = "delta"
)

type TenantSyncBusinessUnit = tenant.SyncBusinessUnit

type TenantSyncOrganization = tenant.SyncOrganization

type TenantSyncRequest struct {
	Mode          TenantSyncMode           `json:"mode"`
	BusinessUnits []TenantSyncBusinessUnit `json:"businessUnits"`
	Organizations []TenantSyncOrganization `json:"organizations"`
	SentAt        int64                    `json:"sentAt"`
}

type TenantSyncResult struct {
	Accepted              bool   `json:"accepted"`
	Mode                  string `json:"mode"`
	BusinessUnitsUpserted int    `json:"businessUnitsUpserted"`
	OrganizationsUpserted int    `json:"organizationsUpserted"`
	ReceivedAt            int64  `json:"receivedAt"`
}

type TenantSyncDelta struct {
	BusinessUnitIDs []pulid.ID
	OrganizationIDs []pulid.ID
}

type EntitlementProvider interface {
	CheckFeature(context.Context, *FeatureCheckRequest) (*FeatureCheckResult, error)
	ListEntitlements(context.Context, *EntitlementsRequest) (*EntitlementsResult, error)
}

type BillingProvider interface {
	GetBillingSummary(context.Context, *BillingSummaryRequest) (*BillingSummaryResult, error)
}

type AccessAuthorizer interface {
	AuthorizeAccess(context.Context, *AccessAuthorizeRequest) (*AccessAuthorizeResult, error)
}

type TenantSyncService interface {
	SyncFull(context.Context) error
	SyncDelta(context.Context, TenantSyncDelta) error
}

type UsageProvider interface {
	CheckLimit(context.Context, *UsageLimitCheckRequest) (*UsageLimitCheckResult, error)
	RecordUsage(context.Context, *UsageRecordRequest) (*UsageRecordResult, error)
}
