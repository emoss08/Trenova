package aiproviderports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/pagination"
)

type BuiltInEmbedding interface {
	Provider(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (provider *aiprovider.Provider, ok bool, err error)
}
