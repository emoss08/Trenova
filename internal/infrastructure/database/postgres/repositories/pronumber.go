package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/pronumber"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/pronumbergen"
	"github.com/emoss08/trenova/internal/pkg/utils/intutils"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ProNumberRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type proNumberRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewProNumberRepository(p ProNumberRepositoryParams) repositories.ProNumberRepository {
	log := p.Logger.With().
		Str("repository", "proNumber").
		Logger()

	return &proNumberRepository{
		db: p.DB,
		l:  &log,
	}
}

// GetNextProNumberWithBusinessUnit gets the next pro number for an organization and business unit
func (r *proNumberRepository) GetNextProNumber(
	ctx context.Context,
	req *repositories.GetProNumberRequest,
) (string, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return "", eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetNextProNumberWithBusinessUnit").
		Str("orgID", req.OrgID.String()).
		Logger()

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// Fetch the pro number format for this organization and business unit
	format, err := r.getProNumberFormat(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get pro number format")
		return "", eris.Wrap(err, "get pro number format")
	}

	// Get or create and increment the sequence in a transaction
	sequence, err := r.getOrCreateAndIncrementSequence(ctx, dba, req)
	if err != nil {
		if eris.Is(err, pronumber.ErrSequenceUpdateConflict) {
			// ! Recursively call the function to get the next pro number
			return r.GetNextProNumber(ctx, req)
		}
		log.Error().Err(err).Msg("failed to get next pro number")
		return "", err
	}

	// Generate pro number using the custom format
	return pronumbergen.GenerateProNumber(format, int(sequence.CurrentSequence), year, month), nil
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

	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetNextProNumberBatchWithBusinessUnit").
		Str("orgID", req.OrgID.String()).
		Int("count", req.Count).
		Logger()

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// Fetch the pro number format for this organization and business unit
	format, err := r.getProNumberFormat(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get pro number format")
		return nil, eris.Wrap(err, "get pro number format")
	}

	// Results slice
	results := make([]string, 0, req.Count)

	// Get or create and increment the sequence by 'count' in a transaction
	sequences, err := r.getOrCreateAndIncrementSequenceBatch(ctx, dba, req)
	if err != nil {
		if eris.Is(err, pronumber.ErrSequenceUpdateConflict) {
			return r.GetNextProNumberBatchWithBusinessUnit(ctx, req)
		}
		log.Error().Err(err).Msg("failed to get batch of pro numbers")
		return nil, err
	}

	// Generate pro numbers using the returned sequences
	for _, seq := range sequences {
		proNumber := pronumbergen.GenerateProNumber(format, int(seq), year, month)
		results = append(results, proNumber)
	}

	return results, nil
}

// getProNumberFormat fetches the organization or business unit specific pro number format
func (r *proNumberRepository) getProNumberFormat(
	ctx context.Context,
	req *repositories.GetProNumberRequest,
) (*pronumbergen.ProNumberFormat, error) {
	// If business unit ID is provided, try to get business unit specific format first
	if !req.BuID.IsNil() {
		format, err := pronumbergen.GetProNumberFormatForBusinessUnit(ctx, req.OrgID, req.BuID)
		if err == nil {
			return format, nil
		}
		// If not found or error, fall back to organization format
	}

	// Get organization format
	return pronumbergen.GetOrganizationProNumberFormat(ctx, req.OrgID)
}

// getOrCreateAndIncrementSequence gets or creates a sequence and increments it in a transaction
func (r *proNumberRepository) getOrCreateAndIncrementSequence(
	ctx context.Context,
	dba bun.IDB,
	req *repositories.GetProNumberRequest,
) (*pronumber.Sequence, error) {
	log := r.l.With().
		Str("operation", "getOrCreateAndIncrementSequence").
		Str("orgID", req.OrgID.String()).
		Int("year", req.Year).
		Int("month", req.Month).
		Logger()

	var sequence *pronumber.Sequence
	err := dba.RunInTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
		func(c context.Context, tx bun.Tx) error {
			var getErr error
			sequence, getErr = r.getSequence(c, tx, req)
			if getErr != nil {
				return getErr
			}

			if incrementErr := r.incrementSequence(c, tx, sequence); incrementErr != nil {
				log.Error().Err(incrementErr).Msg("failed to increment sequence")
				return incrementErr
			}

			return nil
		},
	)

	return sequence, err
}

// getSequence retrieves an existing sequence or creates a new one if it doesn't exist
func (r *proNumberRepository) getSequence(
	ctx context.Context,
	tx bun.Tx,
	req *repositories.GetProNumberRequest,
) (*pronumber.Sequence, error) {
	log := r.l.With().
		Str("operation", "getSequence").
		Str("orgID", req.OrgID.String()).
		Int("year", req.Year).
		Int("month", req.Month).
		Logger()

	sequence := new(pronumber.Sequence)
	err := tx.NewSelect().Model(sequence).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("pns.organization_id = ?", req.OrgID).
				Where("pns.year = ?", req.Year).
				Where("pns.month = ?", req.Month)
		}).
		For("UPDATE").
		Scan(ctx)

	if err == nil {
		return sequence, nil
	}

	if !eris.Is(err, sql.ErrNoRows) {
		log.Error().Err(err).Msg("failed to get sequence")
		return nil, err
	}

	// Create a new sequence since it doesn't exist
	newSequence, err := r.createNewSequence(req)
	if err != nil {
		log.Error().Err(err).Msg("failed to create new sequence")
		return nil, err
	}

	// Insert the new sequence
	if _, err = tx.NewInsert().Model(newSequence).Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to insert new sequence")
		return nil, err
	}

	// Fetch the inserted sequence to get the ID and other fields
	err = tx.NewSelect().Model(newSequence).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("pns.organization_id = ?", req.OrgID).
				Where("pns.year = ?", req.Year).
				Where("pns.month = ?", req.Month)
		}).
		Scan(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch inserted sequence")
		return nil, err
	}

	return newSequence, nil
}

// createNewSequence creates a new sequence for the given organization, year, and month
func (r *proNumberRepository) createNewSequence(
	req *repositories.GetProNumberRequest,
) (*pronumber.Sequence, error) {
	yearInt16, err := intutils.SafeInt16(req.Year)
	if err != nil {
		return nil, eris.Wrapf(err, "invalid year value %d for sequence", req.Year)
	}

	monthInt16, err := intutils.SafeInt16(req.Month)
	if err != nil {
		return nil, eris.Wrapf(err, "invalid month value %d for sequence", req.Month)
	}

	return &pronumber.Sequence{
		OrganizationID:  req.OrgID,
		BusinessUnitID:  req.BuID,
		Year:            yearInt16,
		Month:           monthInt16,
		CurrentSequence: 0,
	}, nil
}

// incrementSequence increments the sequence number with optimistic locking
func (r *proNumberRepository) incrementSequence(
	ctx context.Context,
	tx bun.Tx,
	sequence *pronumber.Sequence,
) error {
	originalVersion := sequence.Version
	sequence.Version++
	sequence.CurrentSequence++

	result, err := tx.NewUpdate().Model(sequence).
		Where("pns.id = ? AND pns.version = ?", sequence.ID, originalVersion).
		Returning("*").Exec(ctx)
	if err != nil {
		return eris.Wrap(err, "failed to update sequence")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return eris.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return pronumber.ErrSequenceUpdateConflict
	}

	return nil
}

// getOrCreateAndIncrementSequenceBatch gets or creates a sequence and increments it by count in a transaction
// Returns a slice of generated sequence numbers
func (r *proNumberRepository) getOrCreateAndIncrementSequenceBatch(
	ctx context.Context,
	dba bun.IDB,
	req *repositories.GetProNumberRequest,
) ([]int64, error) {
	log := r.l.With().
		Str("operation", "getOrCreateAndIncrementSequenceBatch").
		Str("orgID", req.OrgID.String()).
		Int("year", req.Year).
		Int("month", req.Month).
		Int("count", req.Count).
		Logger()

	var sequence *pronumber.Sequence
	var sequences []int64

	err := dba.RunInTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
		func(c context.Context, tx bun.Tx) error {
			var getErr error
			sequence, getErr = r.getSequence(c, tx, req)
			if getErr != nil {
				return getErr
			}

			// Store the starting sequence
			startSequence := sequence.CurrentSequence

			// Increment the sequence by count
			sequence.CurrentSequence += int64(req.Count)
			sequence.Version++

			// Update the sequence in the database
			result, updateErr := tx.NewUpdate().Model(sequence).
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return uq.Where("pns.id = ?", sequence.ID).
						Where("pns.version = ?", sequence.Version-1)
				}).
				Returning("*").Exec(c)
			if updateErr != nil {
				log.Error().Err(updateErr).Msg("failed to update sequence batch")
				return updateErr
			}

			rows, rowErr := result.RowsAffected()
			if rowErr != nil {
				return eris.Wrap(rowErr, "failed to get rows affected")
			}

			if rows == 0 {
				return pronumber.ErrSequenceUpdateConflict
			}

			for i := range req.Count {
				sequences[i] = startSequence + int64(i) + 1
			}

			return nil
		},
	)

	return sequences, err
}
