package carrierintel

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
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

var _ bun.BeforeAppendModelHook = (*CarrierIntelUsageRecord)(nil)

type CarrierIntelUsageRecord struct {
	bun.BaseModel `bun:"table:carrier_intel_usage_ledger,alias:ciuse" json:"-"`

	ID              pulid.ID         `json:"id"              bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID  pulid.ID         `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID  pulid.ID         `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Provider        integration.Type `json:"provider"        bun:"provider,type:integration_type,notnull"`
	Endpoint        Endpoint         `json:"endpoint"        bun:"endpoint,type:VARCHAR(40),notnull"`
	Purpose         Purpose          `json:"purpose"         bun:"purpose,type:VARCHAR(20),notnull"`
	DOTNumber       string           `json:"dotNumber"       bun:"dot_number,type:VARCHAR(12),nullzero"`
	CarrierID       pulid.ID         `json:"carrierId"       bun:"carrier_id,type:VARCHAR(100),nullzero"`
	Billable        bool             `json:"billable"        bun:"billable,type:BOOLEAN,notnull"`
	BillableUnits   int              `json:"billableUnits"   bun:"billable_units,type:INTEGER,notnull"`
	EstimatedCost   decimal.Decimal  `json:"estimatedCost"   bun:"estimated_cost,type:NUMERIC(19,6),notnull"`
	Currency        string           `json:"currency"        bun:"currency,type:VARCHAR(3),notnull"`
	DedupeKey       string           `json:"dedupeKey"       bun:"dedupe_key,type:VARCHAR(128),nullzero"`
	StatusCode      int              `json:"statusCode"      bun:"status_code,type:INTEGER,notnull"`
	Outcome         UsageOutcome     `json:"outcome"         bun:"outcome,type:VARCHAR(20),notnull"`
	LatencyMS       int              `json:"latencyMs"       bun:"latency_ms,type:INTEGER,notnull"`
	InitiatedByType string           `json:"initiatedByType" bun:"initiated_by_type,type:VARCHAR(20),notnull"`
	InitiatedByID   pulid.ID         `json:"initiatedById"   bun:"initiated_by_id,type:VARCHAR(100),nullzero"`
	WorkflowID      string           `json:"workflowId"      bun:"workflow_id,type:VARCHAR(200),nullzero"`
	CreatedAt       int64            `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (u *CarrierIntelUsageRecord) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if u.ID.IsNil() {
			u.ID = pulid.MustNew("ciuse_")
		}
		if u.Currency == "" {
			u.Currency = "USD"
		}
		if u.InitiatedByType == "" {
			u.InitiatedByType = "System"
		}
		if u.CreatedAt == 0 {
			u.CreatedAt = timeutils.NowUnix()
		}
	}
	return nil
}

type CarrierIntelUsageDaily struct {
	bun.BaseModel `bun:"table:carrier_intel_usage_daily,alias:ciusd" json:"-"`

	OrganizationID pulid.ID         `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID         `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	Provider       integration.Type `json:"provider"       bun:"provider,type:integration_type,pk,notnull"`
	Endpoint       Endpoint         `json:"endpoint"       bun:"endpoint,type:VARCHAR(40),pk,notnull"`
	Day            int              `json:"day"            bun:"day,type:INTEGER,pk,notnull"`
	Calls          int              `json:"calls"          bun:"calls,type:INTEGER,notnull"`
	BillableUnits  int              `json:"billableUnits"  bun:"billable_units,type:INTEGER,notnull"`
	EstimatedCost  decimal.Decimal  `json:"estimatedCost"  bun:"estimated_cost,type:NUMERIC(19,6),notnull"`
}
