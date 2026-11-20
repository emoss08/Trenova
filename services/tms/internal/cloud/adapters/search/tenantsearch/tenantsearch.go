package tenantsearch

import (
	"context"

	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/meilisearch"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Client *meilisearch.Client
}

type purge struct {
	client *meilisearch.Client
}

func New(p Params) cloudtenantports.SearchPurge {
	return &purge{client: p.Client}
}

func (p *purge) DeleteOrganizationDocuments(
	ctx context.Context,
	organizationID pulid.ID,
) (int, error) {
	purged, err := p.client.DeleteOrganizationDocuments(ctx, organizationID)
	return len(purged), err
}
