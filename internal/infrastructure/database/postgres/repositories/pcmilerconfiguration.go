/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/pcmilerconfiguration"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type PCMilerConfigurationRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type pcmilerConfigurationRepository struct {
	db     db.Connection
	logger *zerolog.Logger
}

func NewPCMilerConfigurationRepository(
	p PCMilerConfigurationRepositoryParams,
) repositories.PCMilerConfigurationRepository {
	log := p.Logger.With().Str("repository", "pcmiler").Logger()

	return &pcmilerConfigurationRepository{
		db:     p.DB,
		logger: &log,
	}
}

func (r *pcmilerConfigurationRepository) GetPCMilerConfiguration(
	ctx context.Context,
	opts repositories.GetPCMilerConfigurationOptions,
) (*pcmilerconfiguration.PCMilerConfiguration, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	config := new(pcmilerconfiguration.PCMilerConfiguration)
	err = dba.NewSelect().Model(config).
		Where("pcm.organization_id = ?", opts.OrgID).
		Where("pcm.business_unit_id = ?", opts.BuID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return config, nil
}
