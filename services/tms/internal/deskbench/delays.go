package deskbench

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
)

const (
	delayBatch     = 500
	maxDelayPasses = 40
)

// bringStatusesCurrent runs the organization's auto-delay sweep before the
// cases, as production's schedule would have: the bench's worker starts no
// schedules, so on aged data loads past their windows stay InTransit, the
// board calls them late and a count of Delayed does not.
func (b *Bench) bringStatusesCurrent(ctx context.Context, session *Session) (int, error) {
	if b.Shipments == nil {
		return 0, nil
	}
	marked := 0
	for range maxDelayPasses {
		delayed, err := b.Shipments.DelayShipments(
			session.Context(ctx),
			&shipmentports.DelayShipmentsRequest{TenantInfo: session.Tenant, Limit: delayBatch},
			&session.Actor,
		)
		if err != nil {
			return marked, fmt.Errorf("mark late shipments delayed: %w", err)
		}
		marked += len(delayed)
		if len(delayed) < delayBatch {
			return marked, nil
		}
	}

	return marked, nil
}
