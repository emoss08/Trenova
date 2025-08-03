/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetAccessorialChargeByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type AccessorialChargeRepository interface {
	List(
		ctx context.Context,
		opts *ports.LimitOffsetQueryOptions,
	) (*ports.ListResult[*accessorialcharge.AccessorialCharge], error)
	GetByID(
		ctx context.Context,
		opts GetAccessorialChargeByIDRequest,
	) (*accessorialcharge.AccessorialCharge, error)
	Create(
		ctx context.Context,
		a *accessorialcharge.AccessorialCharge,
	) (*accessorialcharge.AccessorialCharge, error)
	Update(
		ctx context.Context,
		a *accessorialcharge.AccessorialCharge,
	) (*accessorialcharge.AccessorialCharge, error)
}
