package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/uptrace/bun"
)

type AdditionalChargeDeletionRequest struct {
	ExistingAdditionalChargeMap map[pulid.ID]*shipment.AdditionalCharge
	UpdatedAdditionalChargeIDs  map[pulid.ID]struct{}
	AdditionalChargesToDelete   []*shipment.AdditionalCharge
}

type AdditionalChargeRepository interface {
	HandleOperations(
		ctx context.Context,
		tx bun.IDB,
		entity *shipment.Shipment,
		isCreate bool,
	) error
}
