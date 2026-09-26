package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type MarkSettlementPaidRequest struct {
	TenantInfo       pagination.TenantInfo
	SettlementID     pulid.ID
	PaymentMethod    string
	PaymentReference string
	PaidAt           int64
}

type CarrierSettlementPayer interface {
	MarkPaid(
		ctx context.Context,
		req *MarkSettlementPaidRequest,
		actor *RequestActor,
	) (*carriersettlement.CarrierSettlement, error)
}

type DriverSettlementPayer interface {
	MarkPaid(
		ctx context.Context,
		req *MarkSettlementPaidRequest,
		actor *RequestActor,
	) (*driversettlement.Settlement, error)
}
