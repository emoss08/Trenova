package documentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
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
	return loadercore.NewLoader(loadercore.BatchGroupFunc(
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
	return loadercore.NewLoader(loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*inboundmessage.InboundAttachment, error) {
			return f.messages.GetAttachmentsByIDs(
				ctx,
				inboundmessageports.GetInboundAttachmentsRequest{
					IDs:        ids,
					TenantInfo: tenantInfo,
				},
			)
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
	return loadercore.NewLoader(loadercore.BatchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*triage.Reply, error) {
			return f.replies.LatestByMessageIDs(ctx, tenantInfo, ids)
		},
	))
}

func newInboundReplyDraftByMessageIDLoader(
	f *InboundReplyLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *triage.Reply] {
	return loadercore.NewLoader(loadercore.BatchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*triage.Reply, error) {
			return f.replies.DraftsByMessageIDs(
				ctx,
				inboundmessageports.InboundRepliesByMessagesRequest{
					TenantInfo: tenantInfo,
					MessageIDs: ids,
				},
			)
		},
	))
}

func newInboundRepliesByMessageIDLoader(
	f *InboundReplyLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*triage.Reply] {
	return loadercore.NewLoader(loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*triage.Reply, error) {
			return f.replies.ListByMessageIDs(
				ctx,
				inboundmessageports.InboundRepliesByMessagesRequest{
					TenantInfo: tenantInfo,
					MessageIDs: ids,
				},
			)
		},
	))
}

func newInboundReplyByIDLoader(
	f *InboundReplyLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *triage.Reply] {
	return loadercore.NewLoader(loadercore.BatchByIDFunc(
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
	return loadercore.NewLoader(loadercore.BatchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*inboundmessageports.DocumentReading, error) {
			return f.peeks.Readings(ctx, tenantInfo, ids)
		},
	))
}

var (
	inboundAttachmentsByMessageIDKey = loadercore.NewKey(
		newInboundAttachmentsByMessageIDLoader,
	)
	inboundAttachmentByIDKey         = loadercore.NewKey(newInboundAttachmentByIDLoader)
	inboundLatestReplyByMessageIDKey = loadercore.NewKey(
		newInboundLatestReplyByMessageIDLoader,
	)
	inboundReplyDraftByMessageIDKey       = loadercore.NewKey(newInboundReplyDraftByMessageIDLoader)
	inboundRepliesByMessageIDKey          = loadercore.NewKey(newInboundRepliesByMessageIDLoader)
	inboundReplyByIDKey                   = loadercore.NewKey(newInboundReplyByIDLoader)
	inboundDocumentReadingByDocumentIDKey = loadercore.NewKey(
		newInboundDocumentReadingByDocumentIDLoader,
	)
)

func InboundAttachmentsByMessageID(
	ctx context.Context,
	key string,
) ([]*inboundmessage.InboundAttachment, error) {
	return loadercore.Load(ctx, inboundAttachmentsByMessageIDKey, key)
}

func (f *InboundAttachmentsByMessageIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func InboundAttachmentByID(
	ctx context.Context,
	key string,
) (*inboundmessage.InboundAttachment, error) {
	return loadercore.Load(ctx, inboundAttachmentByIDKey, key)
}

func (f *InboundAttachmentByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func InboundLatestReplyByMessageID(ctx context.Context, key string) (*triage.Reply, error) {
	return loadercore.Load(ctx, inboundLatestReplyByMessageIDKey, key)
}

func InboundReplyDraftByMessageID(ctx context.Context, key string) (*triage.Reply, error) {
	return loadercore.Load(ctx, inboundReplyDraftByMessageIDKey, key)
}

func InboundRepliesByMessageID(ctx context.Context, key string) ([]*triage.Reply, error) {
	return loadercore.Load(ctx, inboundRepliesByMessageIDKey, key)
}

func InboundReplyByID(ctx context.Context, key string) (*triage.Reply, error) {
	return loadercore.Load(ctx, inboundReplyByIDKey, key)
}

func (f *InboundReplyLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func InboundDocumentReadingByDocumentID(
	ctx context.Context,
	key string,
) (*inboundmessageports.DocumentReading, error) {
	return loadercore.Load(ctx, inboundDocumentReadingByDocumentIDKey, key)
}

func (f *InboundDocumentReadingLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
