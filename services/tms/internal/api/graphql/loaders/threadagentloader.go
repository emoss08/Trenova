package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type threadAgentsByIDsLister interface {
	ListThreadAgentsByIDs(
		ctx context.Context,
		req assistantports.ListThreadAgentsByIDsRequest,
	) ([]*conversation.Thread, error)
}

type ThreadAgentByIDLoaderFactoryParams struct {
	fx.In

	Conversations assistantports.ConversationRepository
}

type ThreadAgentByIDLoaderFactory struct {
	threads threadAgentsByIDsLister
}

func NewThreadAgentByIDLoaderFactory(
	p ThreadAgentByIDLoaderFactoryParams,
) *ThreadAgentByIDLoaderFactory {
	return &ThreadAgentByIDLoaderFactory{threads: p.Conversations}
}

func newThreadAgentByIDLoader(f *ThreadAgentByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *conversation.Thread] {
	return newLoader(batchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*conversation.Thread, error) {
			return f.threads.ListThreadAgentsByIDs(ctx, assistantports.ListThreadAgentsByIDsRequest{
				IDs:        ids,
				TenantInfo: tenantInfo,
			})
		},
		"Thread not found",
	))
}
