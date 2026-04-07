package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type ListBillingQueueItemsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetBillingQueueItemByIDRequest struct {
	TenantInfo             pagination.TenantInfo `json:"-"`
	ItemID                 pulid.ID              `json:"itemId"`
	ExpandShipmentDetails  bool                  `json:"expandShipmentDetails"`
}

type GetBillingQueueStatsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
}

type BillingQueueRepository interface {
	List(
		ctx context.Context,
		req *ListBillingQueueItemsRequest,
	) (*pagination.ListResult[*billingqueue.BillingQueueItem], error)
	GetByID(
		ctx context.Context,
		req *GetBillingQueueItemByIDRequest,
	) (*billingqueue.BillingQueueItem, error)
	Create(
		ctx context.Context,
		entity *billingqueue.BillingQueueItem,
	) (*billingqueue.BillingQueueItem, error)
	Update(
		ctx context.Context,
		entity *billingqueue.BillingQueueItem,
	) (*billingqueue.BillingQueueItem, error)
	ExistsByShipmentAndType(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
		billType billingqueue.BillType,
	) (bool, error)
	GetStatusCounts(
		ctx context.Context,
		req *GetBillingQueueStatsRequest,
	) (map[billingqueue.Status]int, error)
}

func (r *GetBillingQueueItemByIDRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.ItemID, validation.Required.Error("Item ID is required")),
		validation.Field(&r.TenantInfo.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&r.TenantInfo.BuID, validation.Required.Error("Business unit ID is required")),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
