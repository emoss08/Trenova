package carrierintel

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

type BillingModel string

const (
	BillingModelFree        = BillingModel("Free")
	BillingModelPerDOTMonth = BillingModel("PerDOTMonth")
	BillingModelPerMatch    = BillingModel("PerMatch")
	BillingModelPerRequest  = BillingModel("PerRequest")
)

type Endpoint string

const (
	EndpointProfileFull   = Endpoint("profile")
	EndpointProfileLite   = Endpoint("profile-lite")
	EndpointProfileFMCSA  = Endpoint("profile-fmcsa")
	EndpointSearch        = Endpoint("search")
	EndpointAutocomplete  = Endpoint("autocomplete")
	EndpointMonitorAdd    = Endpoint("monitoring-add")
	EndpointMonitorRemove = Endpoint("monitoring-remove")
	EndpointMonitorList   = Endpoint("monitoring-list")
	EndpointEquipment     = Endpoint("equipment")
	EndpointComposite     = Endpoint("composite")
)

func (e Endpoint) String() string { return string(e) }

type EndpointPrice struct {
	Model    BillingModel    `json:"model"`
	UnitCost decimal.Decimal `json:"unitCost"`
}

type PriceBook map[Endpoint]EndpointPrice

func (b PriceBook) Price(endpoint Endpoint) EndpointPrice {
	if price, ok := b[endpoint]; ok {
		return price
	}
	return EndpointPrice{Model: BillingModelFree, UnitCost: decimal.Zero}
}

func (b PriceBook) MonitoringMonthlyCost(count int) decimal.Decimal {
	price := b.Price(EndpointMonitorAdd)
	return price.UnitCost.Mul(decimal.NewFromInt(int64(count)))
}

func BillingDedupeKey(endpoint Endpoint, price EndpointPrice, dotNumber string, now int64) string {
	switch price.Model {
	case BillingModelPerDOTMonth:
		if dotNumber == "" {
			return ""
		}
		return endpoint.String() + ":" + dotNumber + ":" + timeutils.MonthKeyUTC(now)
	default:
		return ""
	}
}

type CarrierIntelUsageRecord struct {
	keel.Table `                                                            keel:"carrier_intel_usage_ledger"`

	ID              pulid.ID         `json:"id" keel:"id,pk,autoid:ciuse_"`
	BusinessUnitID  pulid.ID         `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID         `json:"organizationId" keel:"organization_id,pk,tenant"`
	Provider        integration.Type `json:"provider" keel:"provider"`
	Endpoint        Endpoint         `json:"endpoint" keel:"endpoint"`
	Purpose         Purpose          `json:"purpose" keel:"purpose"`
	DOTNumber       string           `json:"dotNumber" keel:"dot_number,nullzero"`
	CarrierID       pulid.ID         `json:"carrierId" keel:"carrier_id,nullzero"`
	Billable        bool             `json:"billable" keel:"billable"`
	BillableUnits   int              `json:"billableUnits" keel:"billable_units"`
	EstimatedCost   decimal.Decimal  `json:"estimatedCost" keel:"estimated_cost"`
	Currency        string           `json:"currency" keel:"currency"`
	DedupeKey       string           `json:"dedupeKey" keel:"dedupe_key,nullzero"`
	StatusCode      int              `json:"statusCode" keel:"status_code"`
	Outcome         UsageOutcome     `json:"outcome" keel:"outcome"`
	LatencyMS       int              `json:"latencyMs" keel:"latency_ms"`
	InitiatedByType string           `json:"initiatedByType" keel:"initiated_by_type"`
	InitiatedByID   pulid.ID         `json:"initiatedById" keel:"initiated_by_id,nullzero"`
	WorkflowID      string           `json:"workflowId" keel:"workflow_id,nullzero"`
	CreatedAt       int64            `json:"createdAt" keel:"created_at,autocreate"`
}

func (u *CarrierIntelUsageRecord) BeforeCreate(_ context.Context) error {
	if u.Currency == "" {
		u.Currency = "USD"
	}
	if u.InitiatedByType == "" {
		u.InitiatedByType = "System"
	}

	return nil
}

type CarrierIntelUsageDaily struct {
	keel.Table `                                                           keel:"carrier_intel_usage_daily"`

	OrganizationID pulid.ID         `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID         `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	Provider       integration.Type `json:"provider" keel:"provider,pk"`
	Endpoint       Endpoint         `json:"endpoint" keel:"endpoint,pk"`
	Day            int              `json:"day" keel:"day,pk"`
	Calls          int              `json:"calls" keel:"calls"`
	BillableUnits  int              `json:"billableUnits" keel:"billable_units"`
	EstimatedCost  decimal.Decimal  `json:"estimatedCost" keel:"estimated_cost"`
}
