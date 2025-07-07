package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/consolidation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

var ConsolidationFieldConfig = &ports.FieldConfiguration{
	FilterableFields: map[string]bool{
		"consolidationNumber": true,
		"status":              true,
	},
	SortableFields: map[string]bool{
		"consolidationNumber": true,
		"status":              true,
	},
	FieldMap: map[string]string{
		"consolidationNumber": "consolidation_number",
		"status":              "status",
	},
	EnumMap: map[string]bool{
		"status": true,
	},
}

type ConsolidationOptions struct {
	ExpandDetails bool `query:"expandDetails"`
}

func BuildConsolidationListOptions(
	filter *ports.QueryOptions,
	additionalOpts ConsolidationOptions,
) *ListConsolidationRequest {
	return &ListConsolidationRequest{
		Filter:               filter,
		ConsolidationOptions: additionalOpts,
	}
}

type ListConsolidationRequest struct {
	Filter               *ports.QueryOptions `json:"filter"               query:"filter"`
	ConsolidationOptions `json:"consolidationOptions" query:"consolidationOptions"`
}

// type CreateConsolidationRequest struct {
// 	BuID        pulid.ID   `json:"buId"`
// 	OrgID       pulid.ID   `json:"orgId"`
// 	UserID      pulid.ID   `json:"userId"`
// 	ShipmentIDs []pulid.ID `json:"shipmentIds"`
// }

// ConsolidationRepository defines operations for consolidation management
type ConsolidationRepository interface {
	// * Consolidation number generation
	GetNextConsolidationNumber(ctx context.Context, orgID, buID pulid.ID) (string, error)
	GetNextConsolidationNumberBatch(
		ctx context.Context,
		orgID, buID pulid.ID,
		count int,
	) ([]string, error)

	// * CRUD operations (no delete - use status cancellation instead)
	Create(
		ctx context.Context,
		cg *consolidation.ConsolidationGroup,
	) (*consolidation.ConsolidationGroup, error)
	Get(ctx context.Context, id pulid.ID) (*consolidation.ConsolidationGroup, error)
	GetByConsolidationNumber(
		ctx context.Context,
		consolidationNumber string,
	) (*consolidation.ConsolidationGroup, error)
	Update(
		ctx context.Context,
		group *consolidation.ConsolidationGroup,
	) (*consolidation.ConsolidationGroup, error)
	List(
		ctx context.Context,
		req *ListConsolidationRequest,
	) (*ports.ListResult[*consolidation.ConsolidationGroup], error)

	// * Shipment management
	AddShipmentToGroup(ctx context.Context, groupID, shipmentID pulid.ID) error
	RemoveShipmentFromGroup(ctx context.Context, groupID, shipmentID pulid.ID) error
	GetGroupShipments(ctx context.Context, groupID pulid.ID) ([]*shipment.Shipment, error)

	// * Cancellation - cancels consolidation and all associated shipments/moves/stops
	CancelConsolidation(ctx context.Context, groupID pulid.ID) error

	// * Settings
	// GetConsolidationSettings(
	// 	ctx context.Context,
	// 	orgID, buID pulid.ID,
	// ) (*consolidation.ConsolidationSettings, error)
}
