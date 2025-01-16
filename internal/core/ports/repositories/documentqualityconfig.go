package repositories

import (
	"context"

	"github.com/trenova-app/transport/internal/core/domain/documentqualityconfig"
	"github.com/trenova-app/transport/pkg/types/pulid"
)

type GetDocumentQualityConfigOptions struct {
	OrgID pulid.ID
	BuID  pulid.ID
	// UserID is the ID of the user making the request
	UserID pulid.ID
}

type DocumentQualityConfigRepository interface {
	Get(ctx context.Context, opts *GetDocumentQualityConfigOptions) (*documentqualityconfig.DocumentQualityConfig, error)
	Update(ctx context.Context, dqc *documentqualityconfig.DocumentQualityConfig) (*documentqualityconfig.DocumentQualityConfig, error)
}
