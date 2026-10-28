package triagelookupports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type TriagePeekDetail struct {
	ShipmentID         pulid.ID
	Status             shipment.Status
	CustomerName       string
	TrailerType        string
	ActualDeliveryDate *int64
}

type TriageCustomerShipmentsQuery struct {
	TenantInfo pagination.TenantInfo
	CustomerID pulid.ID
	WeekStart  int64
	Limit      int
}

type TriageShipmentDefaults struct {
	ServiceTypeID     pulid.ID
	ShipmentTypeID    pulid.ID
	FormulaTemplateID pulid.ID
	TractorTypeID     pulid.ID
	TrailerTypeID     pulid.ID
}

type TriageRecordMatch struct {
	ID    pulid.ID
	Label string
	Exact bool
}

type TriageLocationMatch struct {
	ID       pulid.ID
	Label    string
	Timezone string
	Exact    bool
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
