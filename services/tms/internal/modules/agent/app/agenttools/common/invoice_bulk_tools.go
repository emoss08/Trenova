package common

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const ParamBillingQueueItemIDs = "billingQueueItemIds"

const NounBillingQueueItem = "billing queue item"

const NounBillingQueueItems = "billing queue items"

func QueueItemNumbers(billing BillingQueueDecider) RecordLabels {
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

func BulkIDsSchema(batch *RecordBatch, description string, extra map[string]any) map[string]any {
	properties := map[string]any{batch.Param: batch.property(description)}
	for name, property := range extra {
		properties[name] = property
	}

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyRequired:             []string{batch.Param},
		toolschema.KeyAdditionalProperties: false,
	}
}
