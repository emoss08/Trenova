/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/pcmilerconfiguration"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetPCMilerConfigurationOptions struct {
	OrgID pulid.ID
	BuID  pulid.ID
}

type PCMilerConfigurationRepository interface {
	GetPCMilerConfiguration(
		ctx context.Context,
		opts GetPCMilerConfigurationOptions,
	) (*pcmilerconfiguration.PCMilerConfiguration, error)
}
