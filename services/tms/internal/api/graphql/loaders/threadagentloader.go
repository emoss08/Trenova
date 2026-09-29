package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type threadAgentsByIDsLister interface {
	ListThreadAgentsByIDs(
		ctx context.Context,
		req repositories.ListThreadAgentsByIDsRequest,
	) ([]*conversation.Thread, error)
}

type ThreadAgentByIDLoaderFactoryParams struct {
	fx.In

	Conversations repositories.ConversationRepository
}

type ThreadAgentByIDLoaderFactory struct {
	threads threadAgentsByIDsLister
}

func NewThreadAgentByIDLoaderFactory(
	p ThreadAgentByIDLoaderFactoryParams,
) *ThreadAgentByIDLoaderFactory {
	return &ThreadAgentByIDLoaderFactory{threads: p.Conversations}
}

func (f *ThreadAgentByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *conversation.Thread] {
	return newLoader(batchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*conversation.Thread, error) {
			return f.threads.ListThreadAgentsByIDs(ctx, repositories.ListThreadAgentsByIDsRequest{
				IDs:        ids,
				TenantInfo: tenantInfo,
			})
		},
		"Thread not found",
	))
}
