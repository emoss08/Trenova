package shipmentimportchatrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipmentimportchat"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.ShipmentImportChatRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-import-chat-repository"),
	}
}

func (r *repository) GetConversationByDocument(
	ctx context.Context,
	req repositories.GetShipmentImportConversationRequest,
) (*shipmentimportchat.Conversation, error) {
	entity := new(shipmentimportchat.Conversation)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("sicc.document_id = ?", req.DocumentID).
		Where("sicc.organization_id = ?", req.TenantInfo.OrgID).
		Where("sicc.business_unit_id = ?", req.TenantInfo.BuID).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			if req.Status != "" {
				return q.Where("sicc.status = ?", req.Status)
			}

			return q.OrderExpr(
				"CASE WHEN sicc.status = ? THEN 0 ELSE 1 END, sicc.updated_at DESC",
				shipmentimportchat.ConversationStatusActive,
			)
		}).
		OrderExpr("sicc.updated_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "ShipmentImportConversation")
	}

	return entity, nil
}

func (r *repository) CreateConversation(
	ctx context.Context,
	entity *shipmentimportchat.Conversation,
) (*shipmentimportchat.Conversation, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateConversation(
	ctx context.Context,
	entity *shipmentimportchat.Conversation,
) (*shipmentimportchat.Conversation, error) {
	previousVersion := entity.Version
	entity.Version++
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", previousVersion).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(result, "ShipmentImportConversation", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) AppendTurn(
	ctx context.Context,
	entity *shipmentimportchat.Turn,
) (*shipmentimportchat.Turn, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListTurns(
	ctx context.Context,
	req repositories.ListShipmentImportTurnsRequest,
) ([]*shipmentimportchat.Turn, error) {
	items := make([]*shipmentimportchat.Turn, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("sict.conversation_id = ?", req.ConversationID).
		Where("sict.organization_id = ?", req.TenantInfo.OrgID).
		Where("sict.business_unit_id = ?", req.TenantInfo.BuID).
		OrderExpr("sict.turn_index ASC, sict.created_at ASC").
		Scan(ctx)
	return items, err
}

func (r *repository) UpdateActiveConversationStatusByDocument(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	status shipmentimportchat.ConversationStatus,
	reason shipmentimportchat.ConversationStatusReason,
) error {
	if status == shipmentimportchat.ConversationStatusActive {
		return errortypes.NewBusinessError("archive status must not be Active")
	}

	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*shipmentimportchat.Conversation)(nil)).
		Set("status = ?", status).
		Set("status_reason = ?", reason).
		Set("updated_at = extract(epoch from current_timestamp)::bigint").
		Where("document_id = ?", documentID).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Where("status = ?", shipmentimportchat.ConversationStatusActive).
		Exec(ctx)
	return err
}

var _ repositories.ShipmentImportChatRepository = (*repository)(nil)
