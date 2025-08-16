package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/shared/pulid"
)

var HoldReasonFieldConfig = &ports.FieldConfiguration{
	FilterableFields: map[string]bool{
		"active":                   true,
		"code":                     true,
		"label":                    true,
		"type":                     true,
		"defaultSeverity":          true,
		"defaultBlocksDispatch":    true,
		"defaultBlocksDelivery":    true,
		"defaultBlocksBilling":     true,
		"defaultVisibleToCustomer": true,
	},
	SortableFields: map[string]bool{
		"active":                   true,
		"code":                     true,
		"label":                    true,
		"type":                     true,
		"defaultSeverity":          true,
		"defaultBlocksDispatch":    true,
		"defaultBlocksDelivery":    true,
		"defaultBlocksBilling":     true,
		"defaultVisibleToCustomer": true,
	},
	FieldMap: map[string]string{
		"active":                   "active",
		"code":                     "code",
		"label":                    "label",
		"type":                     "type",
		"defaultSeverity":          "default_severity",
		"defaultBlocksDispatch":    "default_blocks_dispatch",
		"defaultBlocksDelivery":    "default_blocks_delivery",
		"defaultBlocksBilling":     "default_blocks_billing",
		"defaultVisibleToCustomer": "default_visible_to_customer",
	},
	EnumMap: map[string]bool{
		"type":            true,
		"defaultSeverity": true,
	},
}

type ListHoldReasonRequest struct {
	Filter *ports.QueryOptions
}

func BuildHoldReasonListOptions(
	filter *ports.QueryOptions,
) *ListHoldReasonRequest {
	return &ListHoldReasonRequest{
		Filter: filter,
	}
}

type GetHoldReasonByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type HoldReasonRepository interface {
	List(
		ctx context.Context,
		req *ListHoldReasonRequest,
	) (*ports.ListResult[*shipment.HoldReason], error)
	GetByID(
		ctx context.Context,
		req *GetHoldReasonByIDRequest,
	) (*shipment.HoldReason, error)
	Create(
		ctx context.Context,
		h *shipment.HoldReason,
	) (*shipment.HoldReason, error)
	Update(
		ctx context.Context,
		h *shipment.HoldReason,
	) (*shipment.HoldReason, error)
}
