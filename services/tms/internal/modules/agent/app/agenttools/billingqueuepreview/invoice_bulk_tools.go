package billingqueuepreview

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/recordbatch"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const ParamBillingQueueItemIDs = "billingQueueItemIds"

const NounBillingQueueItem = "billing queue item"

const NounBillingQueueItems = "billing queue items"

func QueueItemNumbers(billing BillingQueueDecider) recordbatch.RecordLabels {
	return func(
		ctx context.Context,
		tenant pagination.TenantInfo,
		ids []pulid.ID,
	) map[pulid.ID]string {
		if billing == nil {
			return nil
		}
		labels := make(map[pulid.ID]string, len(ids))
		for _, id := range ids {
			item, err := billing.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
				ItemID:     id,
				TenantInfo: tenant,
			})
			if err == nil && item != nil {
				labels[id] = item.Number
			}
		}

		return labels
	}
}
