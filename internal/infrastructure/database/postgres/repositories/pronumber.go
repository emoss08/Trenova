/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/sequencestore"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/seqgen"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ProNumberRepositoryParams struct {
	fx.In

	Generator seqgen.Generator
	Logger    *logger.Logger
}

type proNumberRepository struct {
	generator seqgen.Generator
	l         *zerolog.Logger
}

func NewProNumberRepository(p ProNumberRepositoryParams) repositories.ProNumberRepository {
	log := p.Logger.With().
		Str("repository", "proNumber").
		Logger()

	return &proNumberRepository{
		generator: p.Generator,
		l:         &log,
	}
}

// GetNextProNumber gets the next pro number for an organization and business unit
func (r *proNumberRepository) GetNextProNumber(
	ctx context.Context,
	req *repositories.GetProNumberRequest,
) (string, error) {
	genReq := &seqgen.GenerateRequest{
		Type:           sequencestore.SequenceTypeProNumber,
		OrganizationID: req.OrgID,
		BusinessUnitID: req.BuID,
		Count:          1,
	}

	proNumber, err := r.generator.Generate(ctx, genReq)
	if err != nil {
		r.l.Error().
			Err(err).
			Str("orgID", req.OrgID.String()).
			Msg("failed to generate pro number")
		return "", eris.Wrap(err, "generate pro number")
	}

	return proNumber, nil
}

// GetNextProNumberBatch generates a batch of pro numbers
func (r *proNumberRepository) GetNextProNumberBatch(
	ctx context.Context,
	req *repositories.GetProNumberRequest,
) ([]string, error) {
	return r.GetNextProNumberBatchWithBusinessUnit(ctx, req)
}

// GetNextProNumberBatchWithBusinessUnit generates a batch of pro numbers for a specific business unit
func (r *proNumberRepository) GetNextProNumberBatchWithBusinessUnit(
	ctx context.Context,
	req *repositories.GetProNumberRequest,
) ([]string, error) {
	if req.Count <= 0 {
		return []string{}, nil
	}

	genReq := &seqgen.GenerateRequest{
		Type:           sequencestore.SequenceTypeProNumber,
		OrganizationID: req.OrgID,
		BusinessUnitID: req.BuID,
		Count:          req.Count,
	}

	proNumbers, err := r.generator.GenerateBatch(ctx, genReq)
	if err != nil {
		r.l.Error().
			Err(err).
			Str("orgID", req.OrgID.String()).
			Int("count", req.Count).
			Msg("failed to generate pro number batch")
		return nil, eris.Wrap(err, "generate pro number batch")
	}

	return proNumbers, nil
}
