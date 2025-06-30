package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/sequencestore"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/seqgen"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ConsolidationRepositoryParams struct {
	fx.In

	Generator seqgen.Generator
	Logger    *logger.Logger
}

type consolidationRepository struct {
	generator seqgen.Generator
	l         *zerolog.Logger
}

// NewConsolidationRepository creates a new consolidation repository
func NewConsolidationRepository(
	p ConsolidationRepositoryParams,
) repositories.ConsolidationRepository {
	log := p.Logger.With().
		Str("repository", "consolidation").
		Logger()

	return &consolidationRepository{
		generator: p.Generator,
		l:         &log,
	}
}

// GetNextConsolidationNumber generates the next consolidation number
func (r *consolidationRepository) GetNextConsolidationNumber(
	ctx context.Context,
	orgID, buID pulid.ID,
) (string, error) {
	req := &seqgen.GenerateRequest{
		Type:           sequencestore.SequenceTypeConsolidation,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Count:          1,
	}

	consolidationNumber, err := r.generator.Generate(ctx, req)
	if err != nil {
		r.l.Error().
			Err(err).
			Str("orgID", orgID.String()).
			Str("buID", buID.String()).
			Msg("failed to generate consolidation number")
		return "", eris.Wrap(err, "generate consolidation number")
	}

	return consolidationNumber, nil
}

// GetNextConsolidationNumberBatch generates a batch of consolidation numbers
func (r *consolidationRepository) GetNextConsolidationNumberBatch(
	ctx context.Context,
	orgID, buID pulid.ID,
	count int,
) ([]string, error) {
	if count <= 0 {
		return []string{}, nil
	}

	req := &seqgen.GenerateRequest{
		Type:           sequencestore.SequenceTypeConsolidation,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Count:          count,
	}

	consolidationNumbers, err := r.generator.GenerateBatch(ctx, req)
	if err != nil {
		r.l.Error().
			Err(err).
			Str("orgID", orgID.String()).
			Str("buID", buID.String()).
			Int("count", count).
			Msg("failed to generate consolidation number batch")
		return nil, eris.Wrap(err, "generate consolidation number batch")
	}

	return consolidationNumbers, nil
}
