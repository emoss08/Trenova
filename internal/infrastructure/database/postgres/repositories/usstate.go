/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type UsStateRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
	Cache  repositories.UsStateCacheRepository
}

type usStateRepository struct {
	db    db.Connection
	l     *zerolog.Logger
	cache repositories.UsStateCacheRepository
}

func NewUsStateRepository(p UsStateRepositoryParams) repositories.UsStateRepository {
	log := p.Logger.With().
		Str("repository", "us_state").
		Str("component", "database").
		Logger()

	return &usStateRepository{
		db:    p.DB,
		l:     &log,
		cache: p.Cache,
	}
}

func (r *usStateRepository) List(ctx context.Context) (*ports.ListResult[*usstate.UsState], error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "List").
		Logger()

	// Try to get from cache first
	states, err := r.cache.Get(ctx)
	if err == nil && states.Total > 0 {
		log.Debug().Int("stateCount", states.Total).Msg("got states from cache")
		return states, nil
	}

	dbStates := make([]*usstate.UsState, 0)

	// If cache miss, get from database
	count, err := dba.NewSelect().Model(&dbStates).ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to list us states")
		return nil, eris.Wrap(err, "failed to list us states")
	}

	// Set in cache for next time
	if err = r.cache.Set(ctx, dbStates); err != nil {
		log.Error().Err(err).Msg("failed to set states in cache")
	}

	return &ports.ListResult[*usstate.UsState]{
		Items: dbStates,
		Total: count,
	}, nil
}

func (r *usStateRepository) GetByAbbreviation(
	ctx context.Context,
	abbreviation string,
) (*usstate.UsState, error) {
	log := r.l.With().
		Str("operation", "GetByAbbreviation").
		Str("abbreviation", abbreviation).
		Logger()

	// Try to get from cache first
	state, err := r.cache.GetByAbbreviation(ctx, abbreviation)
	if err == nil {
		log.Debug().Msg("got state from cache")
		return state, nil
	}

	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	state = new(usstate.UsState)

	if err = dba.NewSelect().
		Model(state).
		Where("abbreviation = ?", abbreviation).
		Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to get us state by abbreviation")
		return nil, eris.Wrap(err, "failed to get us state by abbreviation")
	}

	// Don't need to explicitly cache this single state as it's already cached
	// as part of the full state list when List() is called

	return state, nil
}
