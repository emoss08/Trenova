package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagepeekservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type InboundAttachmentsByMessageIDLoaderFactoryParams struct {
	fx.In

	MessageRepo inboundmessageports.InboundMessageRepository
}

type InboundAttachmentsByMessageIDLoaderFactory struct {
	messages inboundmessageports.InboundMessageRepository
}

func NewInboundAttachmentsByMessageIDLoaderFactory(
	p InboundAttachmentsByMessageIDLoaderFactoryParams,
) *InboundAttachmentsByMessageIDLoaderFactory {
	return &InboundAttachmentsByMessageIDLoaderFactory{messages: p.MessageRepo}
}

func newInboundAttachmentsByMessageIDLoader(
	f *InboundAttachmentsByMessageIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*inboundmessage.InboundAttachment] {
	return newLoader(batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*inboundmessage.InboundAttachment, error) {
			return f.messages.ListAttachmentsByMessageIDs(
				ctx,
				inboundmessageports.ListAttachmentsByMessageIDsRequest{
					TenantInfo: tenantInfo,
					MessageIDs: ids,
				},
			)
		},
	))
}

type InboundAttachmentByIDLoaderFactoryParams struct {
	fx.In

	MessageRepo inboundmessageports.InboundMessageRepository
}

type InboundAttachmentByIDLoaderFactory struct {
	messages inboundmessageports.InboundMessageRepository
}

func NewInboundAttachmentByIDLoaderFactory(
	p InboundAttachmentByIDLoaderFactoryParams,
) *InboundAttachmentByIDLoaderFactory {
	return &InboundAttachmentByIDLoaderFactory{messages: p.MessageRepo}
}

func newInboundAttachmentByIDLoader(
	f *InboundAttachmentByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *inboundmessage.InboundAttachment] {
	return newLoader(batchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*inboundmessage.InboundAttachment, error) {
			return f.messages.GetAttachmentsByIDs(ctx, inboundmessageports.GetInboundAttachmentsRequest{
				IDs:        ids,
				TenantInfo: tenantInfo,
			})
		},
		"Attachment not found",
	))
}

type InboundReplyLoaderFactoryParams struct {
	fx.In

	ReplyRepo inboundmessageports.InboundReplyRepository
}

type InboundReplyLoaderFactory struct {
	replies inboundmessageports.InboundReplyRepository
}

func NewInboundReplyLoaderFactory(p InboundReplyLoaderFactoryParams) *InboundReplyLoaderFactory {
	return &InboundReplyLoaderFactory{replies: p.ReplyRepo}
}

func newInboundLatestReplyByMessageIDLoader(
	f *InboundReplyLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *triage.Reply] {
	return newLoader(batchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*triage.Reply, error) {
			return f.replies.LatestByMessageIDs(ctx, tenantInfo, ids)
		},
	))
}

func newInboundReplyDraftByMessageIDLoader(
	f *InboundReplyLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *triage.Reply] {
	return newLoader(batchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*triage.Reply, error) {
			return f.replies.DraftsByMessageIDs(ctx, inboundmessageports.InboundRepliesByMessagesRequest{
				TenantInfo: tenantInfo,
				MessageIDs: ids,
			})
		},
	))
}

func newInboundRepliesByMessageIDLoader(
	f *InboundReplyLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*triage.Reply] {
	return newLoader(batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*triage.Reply, error) {
			return f.replies.ListByMessageIDs(ctx, inboundmessageports.InboundRepliesByMessagesRequest{
				TenantInfo: tenantInfo,
				MessageIDs: ids,
			})
		},
	))
}

func newInboundReplyByIDLoader(
	f *InboundReplyLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *triage.Reply] {
	return newLoader(batchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*triage.Reply, error) {
			return f.replies.GetByIDs(ctx, inboundmessageports.GetInboundRepliesRequest{
				IDs:        ids,
				TenantInfo: tenantInfo,
			})
		},
		"Reply not found",
	))
}

type InboundDocumentReadingLoaderFactoryParams struct {
	fx.In

	Peeks *triagepeekservice.Service
}

type InboundDocumentReadingLoaderFactory struct {
	peeks *triagepeekservice.Service
}

func NewInboundDocumentReadingLoaderFactory(
	p InboundDocumentReadingLoaderFactoryParams,
) *InboundDocumentReadingLoaderFactory {
	return &InboundDocumentReadingLoaderFactory{peeks: p.Peeks}
}

func newInboundDocumentReadingByDocumentIDLoader(
	f *InboundDocumentReadingLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *inboundmessageports.DocumentReading] {
	return newLoader(batchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*inboundmessageports.DocumentReading, error) {
			return f.peeks.Readings(ctx, tenantInfo, ids)
		},
	))
}
