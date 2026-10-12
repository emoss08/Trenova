package inboundmessagerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const attachmentsPerMessageHint = 2

func (r *repository) ListAttachmentsByMessageIDs(
	ctx context.Context,
	req inboundmessageports.ListAttachmentsByMessageIDsRequest,
) (map[pulid.ID][]*inboundmessage.InboundAttachment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*inboundmessage.InboundAttachment, error) {
			grouped := make(map[pulid.ID][]*inboundmessage.InboundAttachment, len(req.MessageIDs))
			if len(req.MessageIDs) == 0 {
				return grouped, nil
			}

			cols := inboundmessagecol.InboundAttachmentColumns
			entities := make(
				[]*inboundmessage.InboundAttachment,
				0,
				len(req.MessageIDs)*attachmentsPerMessageHint,
			)
			if err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Where(cols.MessageID.In(), bun.List(req.MessageIDs)).
				Apply(inboundmessagecol.InboundAttachmentApplyTenant(req.TenantInfo)).
				Order(cols.MessageID.OrderAsc(), cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
				Scan(ctx); err != nil {
				return nil, err
			}

			for _, entity := range entities {
				grouped[entity.MessageID] = append(grouped[entity.MessageID], entity)
			}

			return grouped, nil
		},
	)
}

func (r *repository) GetAttachmentByID(
	ctx context.Context,
	req inboundmessageports.GetInboundAttachmentRequest,
) (*inboundmessage.InboundAttachment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*inboundmessage.InboundAttachment, error) {
			entity := new(inboundmessage.InboundAttachment)
			if err := r.db.DBForContext(ctx).
				NewSelect().
				Model(entity).
				Where(inboundmessagecol.InboundAttachmentColumns.ID.Eq(), req.ID).
				Apply(inboundmessagecol.InboundAttachmentApplyTenant(req.TenantInfo)).
				Scan(ctx); err != nil {
				return nil, dberror.HandleNotFoundError(err, "InboundAttachment")
			}

			return entity, nil
		},
	)
}

func (r *repository) GetAttachmentsByIDs(
	ctx context.Context,
	req inboundmessageports.GetInboundAttachmentsRequest,
) ([]*inboundmessage.InboundAttachment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessage.InboundAttachment, error) {
			entities := make([]*inboundmessage.InboundAttachment, 0, len(req.IDs))
			if len(req.IDs) == 0 {
				return entities, nil
			}

			if err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Where(inboundmessagecol.InboundAttachmentColumns.ID.In(), bun.List(req.IDs)).
				Apply(inboundmessagecol.InboundAttachmentApplyTenant(req.TenantInfo)).
				Scan(ctx); err != nil {
				return nil, err
			}

			return entities, nil
		},
	)
}
