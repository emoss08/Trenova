package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetCustomerPaymentByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CustomerPaymentRepository interface {
	GetByID(ctx context.Context, req GetCustomerPaymentByIDRequest) (*customerpayment.Payment, error)
	Create(ctx context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error)
	Update(ctx context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error)
}
