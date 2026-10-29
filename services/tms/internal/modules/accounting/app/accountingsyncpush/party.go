package accountingsyncpush

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/shared/pulid"
)

func PartyLabel(objectType accountingsync.SyncObjectType) string {
	if objectType == accountingsync.SyncObjectCarrierVendor {
		return "Carrier"
	}
	if objectType == accountingsync.SyncObjectDriverVendor {
		return "Driver"
	}
	return "Customer"
}

func (k *Kit) PartyRef(
	ctx context.Context,
	sess *Session,
	res *Resolver,
	objectType accountingsync.SyncObjectType,
	partyID pulid.ID,
) (string, error) {
	targetType, _ := objectType.PartyTarget()
	row, err := res.Mapping(ctx, MappingTarget{TargetType: targetType, ObjectID: partyID})
	if err != nil {
		return "", err
	}
	if row.State == accountingmapping.MappingStateConfirmed && row.ExternalID != "" {
		res.Use(row)
		return row.ExternalID, nil
	}

	label := PartyLabel(objectType) + " " + row.TargetLabel
	dependency, err := k.EnqueueDependency(
		ctx,
		sess,
		&accountingsyncports.AccountingSyncEnqueueRequest{
			ObjectType:   objectType,
			ObjectID:     partyID,
			ObjectNumber: row.TargetLabel,
			Operation:    accountingsync.SyncOperationCreate,
			Revision:     1,
		},
	)
	if err != nil {
		return "", err
	}
	if dependency == nil || dependency.Status == accountingsync.SyncStatusSynced ||
		dependency.Status == accountingsync.SyncStatusSkipped {
		return "", res.Missing(row, MappingTarget{
			TargetType: targetType,
			ObjectID:   partyID,
			Label:      label,
		})
	}
	k.Kick(ctx, sess)
	return "", WaitingOn(dependency, label)
}
