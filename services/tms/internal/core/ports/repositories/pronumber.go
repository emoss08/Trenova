/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
)

type GetProNumberRequest struct {
	OrgID pulid.ID
	BuID  pulid.ID
	Count int // For batch generation only
}

type ProNumberRepository interface {
	// GetNextProNumber generates the next PRO number for an organization
	GetNextProNumber(ctx context.Context, req *GetProNumberRequest) (string, error)

	// GetNextProNumberBatch generates a batch of sequential PRO numbers
	GetNextProNumberBatch(ctx context.Context, req *GetProNumberRequest) ([]string, error)

	// GetNextProNumberBatchWithBusinessUnit generates a batch of sequential PRO numbers for a specific business unit
	GetNextProNumberBatchWithBusinessUnit(
		ctx context.Context,
		req *GetProNumberRequest,
	) ([]string, error)
}
