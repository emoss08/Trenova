package accountingsyncparty

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncpush"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
)

func (p *Pusher) Push(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsyncpush.Result, error) {
	targetType, _ := record.ObjectType.PartyTarget()
	res := p.kit.NewResolver(sess)
	row, err := res.Mapping(
		ctx,
		accountingsyncpush.MappingTarget{TargetType: targetType, ObjectID: record.ObjectID},
	)
	if err != nil {
		return nil, err
	}

	if record.Operation == accountingsync.SyncOperationCreate {
		if row.State == accountingmapping.MappingStateConfirmed && row.ExternalID != "" {
			res.Use(row)
			return accountingsyncpush.Finished(
				sess,
				record,
				&accountingconnectorports.AccountingDocumentResult{
					ExternalID: row.ExternalID,
					DocNumber:  row.ExternalName,
				},
				nil,
				res.MappingIDs(),
			)
		}
		created, createErr := p.creator.CreateReferenceRecord(
			ctx,
			&accountingmappingports.CreateAccountingReferenceRecordRequest{
				TenantInfo: sess.Tenant,
				UserID:     sess.Conn.ConnectedByID,
				MappingID:  row.ID,
				Source:     accountingmapping.MappingSourceCreatedInProvider,
			},
		)
		if createErr != nil {
			return nil, partyCreateError(sess, record.ObjectType, row, createErr)
		}
		return accountingsyncpush.Finished(
			sess,
			record,
			&accountingconnectorports.AccountingDocumentResult{
				ExternalID: created.ExternalID,
				DocNumber:  created.ExternalName,
			},
			nil,
			[]string{created.ID.String()},
		)
	}

	if row.State != accountingmapping.MappingStateConfirmed || row.ExternalID == "" {
		return nil, &accountingsyncpush.NoopError{
			Reason: "The " + strings.ToLower(accountingsyncpush.PartyLabel(record.ObjectType)) +
				" is not mapped, so there is nothing to update",
		}
	}
	written, doc, err := p.upsertParty(ctx, sess, record, targetType, row.ExternalID)
	if err != nil {
		return nil, err
	}
	res.Use(row)
	return accountingsyncpush.Finished(sess, record, written, doc, res.MappingIDs())
}

func (p *Pusher) upsertParty(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
	targetType accountingsync.MappingTargetType,
	externalID string,
) (*accountingconnectorports.AccountingDocumentResult, any, error) {
	if record.ObjectType == accountingsync.SyncObjectCustomer {
		party, err := p.creator.CustomerParty(ctx, sess.Tenant, record.ObjectID)
		if err != nil {
			return nil, nil, err
		}
		doc := &accountingconnectorports.AccountingCustomerDocument{
			Auth:       sess.Auth,
			RequestID:  record.RequestID,
			ExternalID: externalID,
			Party:      *party,
		}
		written, err := sess.Writer.UpsertCustomer(ctx, doc)
		return written, doc, err
	}

	party, err := p.creator.VendorParty(ctx, sess.Tenant, targetType, record.ObjectID)
	if err != nil {
		return nil, nil, err
	}
	doc := &accountingconnectorports.AccountingVendorDocument{
		Auth:       sess.Auth,
		RequestID:  record.RequestID,
		ExternalID: externalID,
		Party:      *party,
	}
	written, err := sess.Writer.UpsertVendor(ctx, doc)
	return written, doc, err
}

func partyCreateError(
	sess *accountingsyncpush.Session,
	objectType accountingsync.SyncObjectType,
	row *accountingmapping.AccountingMapping,
	err error,
) error {
	if errortypes.IsNotFoundError(err) {
		return accountingsyncpush.Blocked(
			accountingsync.SyncErrorNotFound,
			err.Error(),
			"Skip this record",
		)
	}
	kind := "customer"
	if objectType.IsVendor() {
		kind = "vendor"
	}
	classified := sess.Writer.ClassifyDocumentError(err)
	if classified != nil && classified.Category == accountingsync.SyncErrorDuplicate {
		return accountingsyncpush.Blocked(
			accountingsync.SyncErrorMapping,
			err.Error(),
			"Map "+row.TargetLabel+" to the existing "+sess.ProviderName+" "+kind+", then retry",
		)
	}
	if classified != nil &&
		(classified.Category.Retries() || classified.Category.WaitsOnConnection()) {
		return err
	}
	if errortypes.IsBusinessError(err) || errortypes.IsError(err) {
		return accountingsyncpush.Blocked(
			accountingsync.SyncErrorValidation,
			err.Error(),
			"Correct the "+strings.ToLower(accountingsyncpush.PartyLabel(objectType))+
				" in Trenova or map it by hand, then retry",
		)
	}
	return err
}

type Params struct {
	fx.In

	Kit     *accountingsyncpush.Kit
	Creator accountingmappingports.AccountingReferenceCreator
}

type Pusher struct {
	kit     *accountingsyncpush.Kit
	creator accountingmappingports.AccountingReferenceCreator
}

func New(p Params) *Pusher {
	return &Pusher{kit: p.Kit, creator: p.Creator}
}
