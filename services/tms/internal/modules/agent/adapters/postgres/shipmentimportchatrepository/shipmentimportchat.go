//nolint:gocritic // existing legacy workflow/API shape is intentionally kept stable
package shipmentimportchatrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/shipmentimportchat"
	"github.com/emoss08/trenova/internal/modules/agent/domain/shipmentimportchat/shipmentimportchatdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/shipmentimportchatports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var conversationErrors = dberror.Mapping{Entity: "ShipmentImportConversation"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) shipmentimportchatports.ShipmentImportChatRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-import-chat-repository"),
	}
}

func conversationByDocumentQuery(
	req shipmentimportchatports.GetShipmentImportConversationRequest,
) *keel.Query[shipmentimportchat.Conversation] {
	c := shipmentimportchatdb.Conversations
	q := c.Where(c.DocumentID.Eq(req.DocumentID))
	if req.Status != "" {
		q = q.Where(c.Status.Eq(req.Status))
	} else {
		q = q.Order(
			keel.Desc(c.Status.Eq(shipmentimportchat.ConversationStatusActive)),
		)
	}

	return q.Order(c.UpdatedAt.Desc()).Limit(1)
}

func listTurnsQuery(
	req shipmentimportchatports.ListShipmentImportTurnsRequest,
) *keel.Query[shipmentimportchat.Turn] {
	t := shipmentimportchatdb.Turns

	return t.Where(t.ConversationID.Eq(req.ConversationID)).
		Order(t.TurnIndex.Asc(), t.CreatedAt.Asc())
}

func archiveActiveQuery(documentID pulid.ID) *keel.Query[shipmentimportchat.Conversation] {
	c := shipmentimportchatdb.Conversations

	return c.Where(
		c.DocumentID.Eq(documentID),
		c.Status.Eq(shipmentimportchat.ConversationStatusActive),
	)
}

func prepareConversation(entity *shipmentimportchat.Conversation, creating bool) {
	if creating && entity.Status == "" {
		entity.Status = shipmentimportchat.ConversationStatusActive
	}
	if entity.Status == shipmentimportchat.ConversationStatusActive {
		entity.StatusReason = shipmentimportchat.ConversationStatusReasonNone
	}
}

func (r *repository) GetConversationByDocument(
	ctx context.Context,
	req shipmentimportchatports.GetShipmentImportConversationRequest,
) (*shipmentimportchat.Conversation, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentimportchat.Conversation, error) {
			entity, err := conversationByDocumentQuery(req).First(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, conversationErrors.Map(err, req.DocumentID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateConversation(
	ctx context.Context,
	entity *shipmentimportchat.Conversation,
) (*shipmentimportchat.Conversation, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentimportchat.Conversation, error) {
			prepareConversation(entity, true)
			if err := shipmentimportchatdb.Conversations.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				return nil, conversationErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateConversation(
	ctx context.Context,
	entity *shipmentimportchat.Conversation,
) (*shipmentimportchat.Conversation, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentimportchat.Conversation, error) {
			prepareConversation(entity, false)
			if err := shipmentimportchatdb.Conversations.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				return nil, conversationErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) AppendTurn(
	ctx context.Context,
	entity *shipmentimportchat.Turn,
) (*shipmentimportchat.Turn, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipmentimportchat.Turn, error) {
		if entity.ResultStatus == "" {
			entity.ResultStatus = shipmentimportchat.TurnResultStatusCompleted
		}
		if err := shipmentimportchatdb.Turns.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) ListTurns(
	ctx context.Context,
	req shipmentimportchatports.ListShipmentImportTurnsRequest,
) ([]*shipmentimportchat.Turn, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipmentimportchat.Turn, error) {
		return listTurnsQuery(req).All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) UpdateActiveConversationStatusByDocument(
	ctx context.Context,
	documentID pulid.ID,
	_ pagination.TenantInfo,
	status shipmentimportchat.ConversationStatus,
	reason shipmentimportchat.ConversationStatusReason,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if status == shipmentimportchat.ConversationStatusActive {
			return errortypes.NewBusinessError("archive status must not be Active")
		}

		c := shipmentimportchatdb.Conversations
		_, err := archiveActiveQuery(documentID).Update(
			ctx,
			r.db.Keel(ctx),
			c.Status.To(status),
			c.StatusReason.To(reason),
		)
		return err
	})
}

var _ shipmentimportchatports.ShipmentImportChatRepository = (*repository)(nil)
