/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentqualityconfig"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetDocumentQualityConfigOptions struct {
	OrgID pulid.ID
	BuID  pulid.ID
	// UserID is the ID of the user making the request
	UserID pulid.ID
}

type DocumentQualityConfigRepository interface {
	Get(
		ctx context.Context,
		opts *GetDocumentQualityConfigOptions,
	) (*documentqualityconfig.DocumentQualityConfig, error)
	Update(
		ctx context.Context,
		dqc *documentqualityconfig.DocumentQualityConfig,
	) (*documentqualityconfig.DocumentQualityConfig, error)
}
