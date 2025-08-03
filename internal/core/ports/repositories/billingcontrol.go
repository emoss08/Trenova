/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billing"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetBillingControlRequest struct {
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type BillingControlRepository interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*billing.BillingControl, error)
	Update(ctx context.Context, bc *billing.BillingControl) (*billing.BillingControl, error)
}
