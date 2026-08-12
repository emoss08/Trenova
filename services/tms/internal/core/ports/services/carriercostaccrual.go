package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// CarrierCostAccrual is the narrow hook carrier assignment mutations use to
// re-derive a move's cost events after coverage changes, without depending on
// the full settlement service surface.
type CarrierCostAccrual interface {
	ReaccrueMove(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveID pulid.ID,
	) error
}
