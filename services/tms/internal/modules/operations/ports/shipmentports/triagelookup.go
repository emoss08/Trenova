package shipmentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type TriagePeekDetail struct {
	ShipmentID         pulid.ID        `bun:"shipment_id"`
	Status             shipment.Status `bun:"status"`
	CustomerName       string          `bun:"customer_name"`
	TrailerType        string          `bun:"trailer_type"`
	ActualDeliveryDate *int64          `bun:"actual_delivery_date"`
}

type TriageCustomerShipmentsQuery struct {
	TenantInfo pagination.TenantInfo
	CustomerID pulid.ID
	WeekStart  int64
	Limit      int
}

type TriageShipmentDefaults struct {
	ServiceTypeID     pulid.ID `bun:"service_type_id"`
	ShipmentTypeID    pulid.ID `bun:"shipment_type_id"`
	FormulaTemplateID pulid.ID `bun:"formula_template_id"`
	TractorTypeID     pulid.ID `bun:"tractor_type_id"`
	TrailerTypeID     pulid.ID `bun:"trailer_type_id"`
}

type TriageRecordMatch struct {
	ID    pulid.ID `bun:"id"`
	Label string   `bun:"label"`
	Exact bool     `bun:"exact"`
}

type TriageLocationMatch struct {
	ID       pulid.ID `bun:"id"`
	Label    string   `bun:"label"`
	Timezone string   `bun:"timezone"`
	Exact    bool     `bun:"exact"`
}

type TriageLookupQuery struct {
	TenantInfo pagination.TenantInfo
	Text       string
	Limit      int
}

type TriageLocationQuery struct {
	TenantInfo pagination.TenantInfo
	Name       string
	City       string
	State      string
	Limit      int
}

type TriageLookupRepository interface {
	PeekDetails(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentIDs []pulid.ID,
	) (map[pulid.ID]*TriagePeekDetail, error)
	CustomerShipments(
		ctx context.Context,
		req *TriageCustomerShipmentsQuery,
	) ([]pulid.ID, int, error)
	CustomerDefaults(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
	) (*TriageShipmentDefaults, error)
	ActiveServiceTypes(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		limit int,
	) ([]pulid.ID, error)
	FindCustomers(ctx context.Context, req *TriageLookupQuery) ([]TriageRecordMatch, error)
	FindLocations(ctx context.Context, req *TriageLocationQuery) ([]TriageLocationMatch, error)
	FindTrailerTypes(ctx context.Context, req *TriageLookupQuery) ([]TriageRecordMatch, error)
	FindCommodities(ctx context.Context, req *TriageLookupQuery) ([]TriageRecordMatch, error)
	HoldReasonByCode(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		code string,
	) (pulid.ID, error)
}
