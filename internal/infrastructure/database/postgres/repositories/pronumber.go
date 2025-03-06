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

	var sequence *pronumber.Sequence
	err = dba.RunInTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable}, func(c context.Context, tx bun.Tx) error {
		sequence = new(pronumber.Sequence)
		err = tx.NewSelect().Model(sequence).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where("pns.organization_id = ?", orgID).
					Where("pns.year = ?", year).
					Where("pns.month = ?", month)
			}).
			For("UPDATE").
			Scan(ctx)
		if err != nil {
			if eris.Is(err, sql.ErrNoRows) {
				sequence, err = r.createNewSequence(orgID, year, month)
				if err != nil {
					log.Error().Err(err).Msg("failed to create new sequence")
					return err
				}

				// First insert the sequence
				_, err = tx.NewInsert().Model(sequence).Exec(c)
				if err != nil {
					log.Error().Err(err).Msg("failed to insert new sequence")
					return err
				}

				// Then fetch the inserted sequence to get the ID and other fields
				err = tx.NewSelect().Model(sequence).
					Where("pns.organization_id = ?", orgID).
					Where("pns.year = ?", year).
					Where("pns.month = ?", month).
					Scan(c)
				if err != nil {
					log.Error().Err(err).Msg("failed to fetch inserted sequence")
					return err
				}
			} else {
				log.Error().Err(err).Msg("failed to get sequence")
				return err
			}
		}

		if err = r.incrementSequence(c, tx, sequence); err != nil {
			log.Error().Err(err).Msg("failed to increment sequence")
			return err
		}

		return nil
	})
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
