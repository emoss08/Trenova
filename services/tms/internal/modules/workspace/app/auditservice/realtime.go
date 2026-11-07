package auditservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *service) publishRealtimeInvalidation(ctx context.Context, entries []*audit.Entry) {
	if len(entries) == 0 || s.realtime == nil {
		return
	}

	type tenantBatch struct {
		orgID         pulid.ID
		buID          pulid.ID
		actorUserID   pulid.ID
		actorID       pulid.ID
		actorType     kernelports.PrincipalType
		actorAPIKeyID pulid.ID
		record        pulid.ID
		count         int
	}

	tenantBatches := make(map[string]*tenantBatch, len(entries))
	for _, entry := range entries {
		if entry == nil || entry.OrganizationID.IsNil() || entry.BusinessUnitID.IsNil() {
			continue
		}

		key := entry.RealtimeBatchKey()
		batch, ok := tenantBatches[key]
		if !ok {
			batch = &tenantBatch{
				orgID:         entry.OrganizationID,
				buID:          entry.BusinessUnitID,
				actorUserID:   entry.UserID,
				actorID:       entry.PrincipalID,
				actorType:     kernelports.PrincipalType(entry.PrincipalType),
				actorAPIKeyID: entry.APIKeyID,
				record:        entry.ID,
			}
			tenantBatches[key] = batch
		}

		batch.count++
	}

	for _, batch := range tenantBatches {
		action := "created"
		recordID := batch.record
		if batch.count > 1 {
			action = "bulk_created"
			recordID = pulid.ID("")
		}

		if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
			OrganizationID: batch.orgID,
			BusinessUnitID: batch.buID,
			ActorUserID:    batch.actorUserID,
			ActorType:      batch.actorType,
			ActorID:        batch.actorID,
			ActorAPIKeyID:  batch.actorAPIKeyID,
			Resource:       "audit-logs",
			Action:         action,
			RecordID:       recordID,
		}); err != nil {
			s.logger.Warn(
				"failed to publish audit realtime invalidation",
				zap.Error(err),
				zap.String("organizationID", batch.orgID.String()),
				zap.String("businessUnitID", batch.buID.String()),
				zap.String("action", action),
			)
		}
	}
}

func (s *service) publishRecordChanges(ctx context.Context, entries []*audit.Entry) {
	if len(entries) == 0 || s.realtime == nil {
		return
	}

	type resourceBatch struct {
		entry *audit.Entry
		count int
	}

	batches := make(map[string]*resourceBatch, len(entries))
	order := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry == nil || entry.OrganizationID.IsNil() || entry.BusinessUnitID.IsNil() {
			continue
		}
		if !realtimeinvalidation.IsRecordChange(entry.Resource, entry.Operation) {
			continue
		}

		key := entry.RealtimeBatchKey() + ":" + string(entry.Resource)
		batch, ok := batches[key]
		if !ok {
			batch = &resourceBatch{entry: entry}
			batches[key] = batch
			order = append(order, key)
		}
		batch.count++
	}

	for _, key := range order {
		batch := batches[key]
		entry := batch.entry
		action := realtimeinvalidation.RecordChangeAction(entry.Operation)
		recordID := pulid.ID(entry.ResourceID)
		if batch.count > 1 {
			action = "bulk_" + action
			recordID = pulid.ID("")
		}

		if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
			OrganizationID: entry.OrganizationID,
			BusinessUnitID: entry.BusinessUnitID,
			ActorUserID:    entry.UserID,
			ActorType:      kernelports.PrincipalType(entry.PrincipalType),
			ActorID:        entry.PrincipalID,
			ActorAPIKeyID:  entry.APIKeyID,
			Resource:       realtimeinvalidation.RecordChangeResource(entry.Resource),
			Action:         action,
			RecordID:       recordID,
		}); err != nil {
			s.logger.Warn(
				"failed to publish record change",
				zap.Error(err),
				zap.String("resource", string(entry.Resource)),
				zap.String("action", action),
			)
		}
	}
}
