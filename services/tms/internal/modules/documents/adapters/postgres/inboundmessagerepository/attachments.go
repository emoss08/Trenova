package inboundmessagerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/shared/pulid"
)

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

			a := inboundmessagedb.InboundAttachments
			entities, err := a.Where(a.MessageID.In(req.MessageIDs...)).
				Order(a.MessageID.Asc(), a.CreatedAt.Asc(), a.ID.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
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
			entity, err := inboundmessagedb.InboundAttachments.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, attachmentErrors.Map(err, req.ID.String())
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
			if len(req.IDs) == 0 {
				return []*inboundmessage.InboundAttachment{}, nil
			}

			a := inboundmessagedb.InboundAttachments
			return orEmptyAttachments(a.Where(a.ID.In(req.IDs...)).
				Limit(len(req.IDs)).
				List(ctx, r.db.Keel(ctx)))
		},
	)
}
