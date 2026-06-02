package editenderrecipientrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.EDITenderRecipientRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-tender-recipient-repository"),
	}
}

func (r *repository) GetTenderRecipientByID(
	ctx context.Context,
	req repositories.GetEDITenderRecipientByIDRequest,
) (*edi.TenderRecipient, error) {
	entity := new(edi.TenderRecipient)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("etr.id = ?", req.ID).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return applyTenderRecipientTenantScope(query, req.TenantInfo)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITenderRecipient")
	}
	return entity, nil
}

func (r *repository) ListActiveTenderRecipientsForSourceShipment(
	ctx context.Context,
	req repositories.ListEDITenderRecipientsForSourceShipmentRequest,
) ([]*edi.TenderRecipient, error) {
	entities := make([]*edi.TenderRecipient, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("etr.source_organization_id = ?", req.TenantInfo.OrgID).
		Where("etr.source_business_unit_id = ?", req.TenantInfo.BuID).
		Where("etr.source_shipment_id = ?", req.SourceShipmentID).
		Where("etr.status = ?", edi.TenderRecipientStatusActive).
		Order("etr.created_at ASC").
		Scan(ctx)
	return entities, err
}

func (r *repository) UpsertTenderRecipient(
	ctx context.Context,
	req repositories.UpsertEDITenderRecipientRequest,
) (*edi.TenderRecipient, error) {
	entity := req.Recipient
	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On(`CONFLICT (
				source_shipment_id,
				source_business_unit_id,
				source_organization_id,
				recipient_kind,
				COALESCE(recipient_organization_id, ''),
				COALESCE(recipient_business_unit_id, ''),
				COALESCE(edi_partner_id, ''),
				COALESCE(partner_document_profile_id, '')
			) DO UPDATE`).
		Set("latest_baseline_payload = EXCLUDED.latest_baseline_payload").
		Set("latest_baseline_hash = EXCLUDED.latest_baseline_hash").
		Set("baseline_recorded_at = EXCLUDED.baseline_recorded_at").
		Set("baseline_status = EXCLUDED.baseline_status").
		Set("original_transfer_id = COALESCE(NULLIF(EXCLUDED.original_transfer_id, ''), edi_tender_recipients.original_transfer_id)").
		Set("shipment_link_id = COALESCE(NULLIF(EXCLUDED.shipment_link_id, ''), edi_tender_recipients.shipment_link_id)").
		Set("original_message_id = COALESCE(NULLIF(EXCLUDED.original_message_id, ''), edi_tender_recipients.original_message_id)").
		Set("partner_document_profile_id = COALESCE(NULLIF(EXCLUDED.partner_document_profile_id, ''), edi_tender_recipients.partner_document_profile_id)").
		Set("communication_profile_id = COALESCE(NULLIF(EXCLUDED.communication_profile_id, ''), edi_tender_recipients.communication_profile_id)").
		Set("status = ?", edi.TenderRecipientStatusActive).
		Set("updated_at = extract(epoch from current_timestamp)::bigint").
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) UpdateTenderRecipient(
	ctx context.Context,
	entity *edi.TenderRecipient,
) (*edi.TenderRecipient, error) {
	ov := entity.Version
	entity.Version++
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(
		results,
		"EDITenderRecipient",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}
	return entity, nil
}

func applyTenderRecipientTenantScope(
	query *bun.SelectQuery,
	tenantInfo pagination.TenantInfo,
) *bun.SelectQuery {
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(
			"(etr.source_organization_id = ? AND etr.source_business_unit_id = ?)",
			tenantInfo.OrgID,
			tenantInfo.BuID,
		).WhereOr(
			`(etr.recipient_organization_id = ?
				AND (
					etr.recipient_business_unit_id = ?
					OR etr.recipient_business_unit_id IS NULL
					OR etr.business_unit_id = ?
				))`,
			tenantInfo.OrgID,
			tenantInfo.BuID,
			tenantInfo.BuID,
		)
	})
}
