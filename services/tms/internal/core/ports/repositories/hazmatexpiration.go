/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/compliance"
	"github.com/emoss08/trenova/shared/pulid"
)

type HazmatExpirationRepository interface {
	GetHazmatExpirationByStateID(
		ctx context.Context,
		stateID pulid.ID,
	) (*compliance.HazmatExpiration, error)
}
