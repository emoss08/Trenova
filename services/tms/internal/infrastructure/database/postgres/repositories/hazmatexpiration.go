/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/compliance"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type HazmatExpirationRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type hazmatExpirationRepository struct {
	db     db.Connection
	logger *zerolog.Logger
}

func NewHazmatExpirationRepository(
	p HazmatExpirationRepositoryParams,
) repositories.HazmatExpirationRepository {
	log := p.Logger.With().
		Str("repository", "hazmat_expiration").
		Logger()

	return &hazmatExpirationRepository{
		db:     p.DB,
		logger: &log,
	}
}

func (r *hazmatExpirationRepository) GetHazmatExpirationByStateID(
	ctx context.Context,
	stateID pulid.ID,
) (*compliance.HazmatExpiration, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	expiration := new(compliance.HazmatExpiration)
	err = dba.NewSelect().Model(expiration).
		Where("state_id = ?", stateID).
		Scan(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get hazmat expiration by state id")
	}

	return expiration, nil
}
