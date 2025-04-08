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
	"github.com/emoss08/trenova/pkg/types/pulid"
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

func (r *proNumberRepository) GetNextProNumber(ctx context.Context, orgID pulid.ID) (string, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return "", eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetNextProNumber").
		Str("orgID", orgID.String()).
		Logger()

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// Fetch the organization-specific pro number format
	format, err := pronumbergen.GetOrganizationProNumberFormat(ctx, orgID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get pro number format")
		return "", eris.Wrap(err, "get pro number format")
	}

	// Get or create and increment the sequence in a transaction
	sequence, err := r.getOrCreateAndIncrementSequence(ctx, dba, orgID, year, month)
	if err != nil {
		if eris.Is(err, pronumber.ErrSequenceUpdateConflict) {
			return r.GetNextProNumber(ctx, orgID)
		}
		log.Error().Err(err).Msg("failed to get next pro number")
		return "", err
	}

	// Generate pro number using the custom format
	return pronumbergen.GenerateProNumber(format, int(sequence.CurrentSequence), year, month), nil
}

// getOrCreateAndIncrementSequence gets or creates a sequence and increments it in a transaction
func (r *proNumberRepository) getOrCreateAndIncrementSequence(
	ctx context.Context,
	dba bun.IDB,
	orgID pulid.ID,
	year, month int,
) (*pronumber.Sequence, error) {
	log := r.l.With().
		Str("operation", "getOrCreateAndIncrementSequence").
		Str("orgID", orgID.String()).
		Int("year", year).
		Int("month", month).
		Logger()

	var sequence *pronumber.Sequence
	err := dba.RunInTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable}, func(c context.Context, tx bun.Tx) error {
		var getErr error
		sequence, getErr = r.getSequence(c, tx, orgID, year, month)
		if getErr != nil {
			return getErr
		}

		if incrementErr := r.incrementSequence(c, tx, sequence); incrementErr != nil {
			log.Error().Err(incrementErr).Msg("failed to increment sequence")
			return incrementErr
		}

		return nil
	})

	return sequence, err
}

// getSequence retrieves an existing sequence or creates a new one if it doesn't exist
func (r *proNumberRepository) getSequence(
	ctx context.Context,
	tx bun.Tx,
	orgID pulid.ID,
	year, month int,
) (*pronumber.Sequence, error) {
	log := r.l.With().
		Str("operation", "getSequence").
		Str("orgID", orgID.String()).
		Int("year", year).
		Int("month", month).
		Logger()

	sequence := new(pronumber.Sequence)
	err := tx.NewSelect().Model(sequence).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("pns.organization_id = ?", orgID).
				Where("pns.year = ?", year).
				Where("pns.month = ?", month)
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
	newSequence, err := r.createNewSequence(orgID, year, month)
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
		Where("pns.organization_id = ?", orgID).
		Where("pns.year = ?", year).
		Where("pns.month = ?", month).
		Scan(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch inserted sequence")
		return nil, err
	}

	return newSequence, nil
}

// createNewSequence creates a new sequence for the given organization, year, and month
func (r *proNumberRepository) createNewSequence(orgID pulid.ID, year, month int) (*pronumber.Sequence, error) {
	yearInt16, err := pronumber.SafeInt16(year)
	if err != nil {
		return nil, eris.Wrapf(err, "invalid year value %d for sequence", year)
	}

	monthInt16, err := pronumber.SafeInt16(month)
	if err != nil {
		return nil, eris.Wrapf(err, "invalid month value %d for sequence", month)
	}

	return &pronumber.Sequence{
		OrganizationID:  orgID,
		Year:            yearInt16,
		Month:           monthInt16,
		CurrentSequence: 0,
	}, nil
}

// incrementSequence increments the sequence number with optimistic locking
func (r *proNumberRepository) incrementSequence(ctx context.Context, tx bun.Tx, sequence *pronumber.Sequence) error {
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
