//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package editenderrecipientrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange/editenderchangedb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var recipientErrors = dberror.Mapping{Entity: "EDITenderRecipient"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) editenderports.EDITenderRecipientRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-tender-recipient-repository"),
	}
}

func (r *repository) GetTenderRecipientByID(
	ctx context.Context,
	req editenderports.GetEDITenderRecipientByIDRequest,
) (*editenderchange.TenderRecipient, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*editenderchange.TenderRecipient, error) {
			tr := editenderchangedb.TenderRecipients
			entity, err := tr.Where(tr.ID.Eq(req.ID), visibleTo(tr, req.TenantInfo)).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, recipientErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ListActiveTenderRecipientsForSourceShipment(
	ctx context.Context,
	req editenderports.ListEDITenderRecipientsForSourceShipmentRequest,
) ([]*editenderchange.TenderRecipient, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*editenderchange.TenderRecipient, error) {
			tr := editenderchangedb.TenderRecipients
			return tr.Where(
				tr.SourceOrganizationID.Eq(req.TenantInfo.OrgID),
				tr.SourceShipmentID.Eq(req.SourceShipmentID),
				tr.Status.Eq(editenderchange.TenderRecipientStatusActive),
			).
				Order(tr.CreatedAt.Asc()).
				All(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *repository) GetActiveExternalRecipientByShipmentReference(
	ctx context.Context,
	req editenderports.GetActiveExternalEDITenderRecipientByReferenceRequest,
) (*editenderchange.TenderRecipient, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*editenderchange.TenderRecipient, error) {
			tr := editenderchangedb.TenderRecipients
			entity, err := tr.Where(
				tr.SourceOrganizationID.Eq(req.TenantInfo.OrgID),
				tr.EDIPartnerID.Eq(req.PartnerID),
				tr.RecipientKind.Eq(editenderchange.TenderRecipientKindExternal),
				tr.Status.Eq(editenderchange.TenderRecipientStatusActive),
				keel.Or(
					tr.SourceShipmentID.Eq(pulid.ID(req.Reference)),
					tr.LatestBaselinePayload.Text("bol").Eq(req.Reference),
					tr.LatestBaselinePayload.Text("shipmentId").Eq(req.Reference),
				),
			).
				Order(tr.CreatedAt.Desc()).
				First(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, recipientErrors.Map(err, req.Reference)
			}

			return entity, nil
		},
	)
}

func (r *repository) UpsertTenderRecipient(
	ctx context.Context,
	req editenderports.UpsertEDITenderRecipientRequest,
) (*editenderchange.TenderRecipient, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*editenderchange.TenderRecipient, error) {
			db := r.db.Keel(ctx)
			tr := editenderchangedb.TenderRecipients
			entity := req.Recipient
			_, err := tr.Upsert(
				ctx,
				db,
				[]*editenderchange.TenderRecipient{entity},
				keel.OnConflict(
					tr.SourceShipmentID,
					tr.BusinessUnitID,
					tr.SourceOrganizationID,
					tr.RecipientKind,
					tr.RecipientOrganizationID.Coalesce(keel.ConstText("")),
					tr.EDIPartnerID.Coalesce(keel.ConstText("")),
				).
					Update(
						tr.LatestBaselinePayload,
						tr.LatestBaselineHash,
						tr.BaselineRecordedAt,
						tr.BaselineStatus,
					).
					Set(
						tr.OriginalTransferID.ToExpr(keepStored(tr.OriginalTransferID.Col)),
						tr.ShipmentLinkID.ToExpr(keepStored(tr.ShipmentLinkID.Col)),
						tr.OriginalMessageID.ToExpr(keepStored(tr.OriginalMessageID.Col)),
						tr.CommunicationProfileID.ToExpr(
							keepStored(tr.CommunicationProfileID.Col),
						),
						tr.Status.To(editenderchange.TenderRecipientStatusActive),
					),
			)
			if err != nil {
				return nil, recipientErrors.Map(err, entity.ID.String())
			}

			stored, err := tr.Where(
				tr.SourceShipmentID.Eq(entity.SourceShipmentID),
				tr.SourceOrganizationID.Eq(entity.SourceOrganizationID),
				tr.RecipientKind.Eq(entity.RecipientKind),
				tr.RecipientOrganizationID.Eq(entity.RecipientOrganizationID),
				tr.EDIPartnerID.Eq(entity.EDIPartnerID),
			).One(ctx, db)
			if err != nil {
				return nil, recipientErrors.Map(err, entity.ID.String())
			}

			return stored, nil
		},
	)
}

func (r *repository) UpdateTenderRecipient(
	ctx context.Context,
	entity *editenderchange.TenderRecipient,
) (*editenderchange.TenderRecipient, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*editenderchange.TenderRecipient, error) {
			if err := editenderchangedb.TenderRecipients.Save(ctx, r.db.Keel(ctx), entity); err != nil {
				return nil, recipientErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func keepStored(col keel.Col) keel.StringExpr {
	return keel.StringOf(col.Excluded()).NullIf("").Coalesce(col)
}

func visibleTo(
	tr *editenderchangedb.TenderRecipientsTable,
	tenant pagination.TenantInfo,
) keel.Pred {
	return keel.Or(
		tr.SourceOrganizationID.Eq(tenant.OrgID),
		tr.RecipientOrganizationID.Eq(tenant.OrgID),
	)
}
