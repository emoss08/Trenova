//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package editenderrecipientrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
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

func New(p Params) ediports.EDITenderRecipientRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-tender-recipient-repository"),
	}
}

func (r *repository) GetTenderRecipientByID(
	ctx context.Context,
	req ediports.GetEDITenderRecipientByIDRequest,
) (*edi.TenderRecipient, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.TenderRecipient, error) {
		tr := edidb.TenderRecipients
		entity, err := tr.Where(tr.ID.Eq(req.ID), visibleTo(tr, req.TenantInfo)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, recipientErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListActiveTenderRecipientsForSourceShipment(
	ctx context.Context,
	req ediports.ListEDITenderRecipientsForSourceShipmentRequest,
) ([]*edi.TenderRecipient, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.TenderRecipient, error) {
		tr := edidb.TenderRecipients
		return tr.Where(
			tr.SourceOrganizationID.Eq(req.TenantInfo.OrgID),
			tr.SourceBusinessUnitID.Eq(req.TenantInfo.BuID),
			tr.SourceShipmentID.Eq(req.SourceShipmentID),
			tr.Status.Eq(edi.TenderRecipientStatusActive),
		).
			Order(tr.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) GetActiveExternalRecipientByShipmentReference(
	ctx context.Context,
	req ediports.GetActiveExternalEDITenderRecipientByReferenceRequest,
) (*edi.TenderRecipient, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.TenderRecipient, error) {
		tr := edidb.TenderRecipients
		entity, err := tr.Where(
			tr.SourceOrganizationID.Eq(req.TenantInfo.OrgID),
			tr.SourceBusinessUnitID.Eq(req.TenantInfo.BuID),
			tr.EDIPartnerID.Eq(req.PartnerID),
			tr.RecipientKind.Eq(edi.TenderRecipientKindExternal),
			tr.Status.Eq(edi.TenderRecipientStatusActive),
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
	})
}

func (r *repository) UpsertTenderRecipient(
	ctx context.Context,
	req ediports.UpsertEDITenderRecipientRequest,
) (*edi.TenderRecipient, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.TenderRecipient, error) {
		db := r.db.Keel(ctx)
		tr := edidb.TenderRecipients
		entity := req.Recipient
		_, err := tr.Upsert(ctx, db, []*edi.TenderRecipient{entity}, keel.OnConflict(
			tr.SourceShipmentID,
			tr.SourceBusinessUnitID,
			tr.SourceOrganizationID,
			tr.RecipientKind,
			keel.Raw("COALESCE(?, '')", tr.RecipientOrganizationID),
			keel.Raw("COALESCE(?, '')", tr.RecipientBusinessUnitID),
			keel.Raw("COALESCE(?, '')", tr.EDIPartnerID),
			keel.Raw("COALESCE(?, '')", tr.PartnerDocumentProfileID),
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
				tr.PartnerDocumentProfileID.ToExpr(
					keepStored(tr.PartnerDocumentProfileID.Col),
				),
				tr.CommunicationProfileID.ToExpr(
					keepStored(tr.CommunicationProfileID.Col),
				),
				tr.Status.To(edi.TenderRecipientStatusActive),
			))
		if err != nil {
			return nil, recipientErrors.Map(err, entity.ID.String())
		}

		stored, err := tr.Where(
			tr.SourceShipmentID.Eq(entity.SourceShipmentID),
			tr.SourceBusinessUnitID.Eq(entity.SourceBusinessUnitID),
			tr.SourceOrganizationID.Eq(entity.SourceOrganizationID),
			tr.RecipientKind.Eq(entity.RecipientKind),
			tr.RecipientOrganizationID.Eq(entity.RecipientOrganizationID),
			tr.RecipientBusinessUnitID.Eq(entity.RecipientBusinessUnitID),
			tr.EDIPartnerID.Eq(entity.EDIPartnerID),
			tr.PartnerDocumentProfileID.Eq(entity.PartnerDocumentProfileID),
		).One(ctx, db)
		if err != nil {
			return nil, recipientErrors.Map(err, entity.ID.String())
		}

		return stored, nil
	})
}

func (r *repository) UpdateTenderRecipient(
	ctx context.Context,
	entity *edi.TenderRecipient,
) (*edi.TenderRecipient, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.TenderRecipient, error) {
		if err := edidb.TenderRecipients.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, recipientErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func keepStored(col keel.Col) keel.Expr {
	return keel.Raw("COALESCE(NULLIF(?, ''), ?)", col.Excluded(), col)
}

func visibleTo(tr *edidb.TenderRecipientsTable, tenant pagination.TenantInfo) keel.Pred {
	return keel.Or(
		keel.And(
			tr.SourceOrganizationID.Eq(tenant.OrgID),
			tr.SourceBusinessUnitID.Eq(tenant.BuID),
		),
		keel.And(
			tr.RecipientOrganizationID.Eq(tenant.OrgID),
			keel.Or(
				tr.RecipientBusinessUnitID.Eq(tenant.BuID),
				tr.RecipientBusinessUnitID.IsNull(),
				tr.BusinessUnitID.Eq(tenant.BuID),
			),
		),
	)
}
