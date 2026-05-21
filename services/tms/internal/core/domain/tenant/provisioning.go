package tenant

import (
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

type ProvisioningCustomer struct {
	ID       pulid.ID       `json:"id"`
	Name     string         `json:"name"`
	Code     string         `json:"code"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type ProvisioningWorkspace struct {
	ID             pulid.ID       `json:"id"`
	BusinessUnitID pulid.ID       `json:"businessUnitId"`
	Name           string         `json:"name"`
	StateID        pulid.ID       `json:"stateId,omitempty"`
	State          string         `json:"state,omitempty"`
	AddressLine1   string         `json:"addressLine1"`
	AddressLine2   string         `json:"addressLine2,omitempty"`
	City           string         `json:"city"`
	PostalCode     string         `json:"postalCode"`
	Timezone       string         `json:"timezone"`
	BucketName     string         `json:"bucketName"`
	TaxID          string         `json:"taxId"`
	ScacCode       string         `json:"scacCode"`
	DOTNumber      string         `json:"dotNumber"`
	LoginSlug      string         `json:"loginSlug"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type ProvisioningAssignment struct {
	Status string `json:"status"`
}

type ProvisioningSubscription struct {
	ID       string `json:"id,omitempty"`
	PlanID   string `json:"planId,omitempty"`
	PlanKey  string `json:"planKey,omitempty"`
	Status   string `json:"status,omitempty"`
	Active   bool   `json:"active"`
	PeriodTo int64  `json:"periodTo,omitempty"`
}

type ProvisioningEntitlement struct {
	FeatureKey platformcatalog.FeatureKey `json:"featureKey"`
	Allowed    bool                       `json:"allowed"`
}

type ProvisioningLimit struct {
	MeterKey platformcatalog.MeterKey `json:"meterKey"`
	Limit    int64                    `json:"limit"`
	Window   string                   `json:"window,omitempty"`
}

type ProvisioningRequest struct {
	InstanceID     string                    `json:"instanceId"`
	Customer       ProvisioningCustomer      `json:"customer"`
	Workspace      ProvisioningWorkspace     `json:"workspace"`
	Assignment     ProvisioningAssignment    `json:"assignment"`
	Subscription   ProvisioningSubscription  `json:"subscription"`
	Entitlements   []ProvisioningEntitlement `json:"entitlements"`
	Limits         []ProvisioningLimit       `json:"limits"`
	SentAt         int64                     `json:"sentAt"`
	IdempotencyKey string                    `json:"idempotencyKey,omitempty"`
}

type ProvisioningResult struct {
	Accepted              bool     `json:"accepted"`
	BusinessUnitID        pulid.ID `json:"businessUnitId"`
	OrganizationID        pulid.ID `json:"organizationId"`
	BusinessUnitsUpserted int      `json:"businessUnitsUpserted"`
	OrganizationsUpserted int      `json:"organizationsUpserted"`
	ReceivedAt            int64    `json:"receivedAt"`
}
