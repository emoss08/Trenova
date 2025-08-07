/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ConsolidationNumberRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type consolidationNumberRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// func NewConsolidationNumberRepository(
// 	p ConsolidationNumberRepositoryParams,
// ) repositories.ConsolidationNumberRepository {
// 	log := p.Logger.With().Str("repository", "consolidationNumber").Logger()

// 	return &consolidationNumberRepository{
// 		db: p.DB,
// 		l:  &log,
// 	}
// }

// // getOrCreateAndIncrementSequence gets or creates a sequence and increments it in a transaction
// func (r *consolidationNumberRepository) getOrCreateAndIncrementSequence(
// 	ctx context.Context,
// 	dba bun.IDB,
// 	orgID pulid.ID,
// 	year, month int,
// ) (sequence *consolidation.ConsolidationSequence, err error) {
// 	log := r.l.With().
// 		Str("operation", "getOrCreateAndIncrementSequence").
// 		Str("orgID", orgID.String()).
// 		Int("year", year).
// 		Int("month", month).
// 		Logger()

// 	err = dba.RunInTx(
// 		ctx,
// 		&sql.TxOptions{Isolation: sql.LevelSerializable},
// 		func(c context.Context, tx bun.Tx) error {
// 			sequence, err = r.getSequence(c, tx, orgID, year, month)
// 			if err != nil {
// 				return oops.In("consolidation_number_repository.getOrCreateAndIncrementSequence").
// 					With("orgID", orgID.String()).
// 					With("year", year).
// 					With("month", month).
// 					Time(time.Now()).
// 					Wrap(err)
// 			}

// 			if incrementErr := r.incrementSequence(c, tx, sequence); incrementErr != nil {
// 				log.Error().Err(incrementErr).Msg("failed to increment sequence")
// 				return oops.In("consolidation_number_repository.getOrCreateAndIncrementSequence").
// 					With("orgID", orgID.String()).
// 					With("year", year).
// 					With("month", month).
// 					Time(time.Now()).
// 					Wrap(incrementErr)
// 			}

// 			return nil
// 		},
// 	)

// 	return sequence, err
// }

// func (r *consolidationNumberRepository) createNewSequence(
// 	orgID pulid.ID,
// 	year, month int,
// ) (*consolidation.ConsolidationSequence, error) {
// 	log := r.l.With().
// 		Str("operation", "createNewSequence").
// 		Str("orgID", orgID.String()).
// 		Int("year", year).
// 		Int("month", month).
// 		Logger()

// 	yearInt16, err := intutils.SafeInt16(year)
// 	if err != nil {
// 		log.Error().Err(err).Msg("invalid year value for sequence")
// 		return nil, oops.In("consolidation_number_repository.createNewSequence").
// 			With("year", year).
// 			With("month", month).
// 			Time(time.Now()).
// 			Wrapf(err, "invalid year value %d for sequence", year)
// 	}

// 	monthInt16, err := intutils.SafeInt16(month)
// 	if err != nil {
// 		log.Error().Err(err).Msg("invalid month value for sequence")
// 		return nil, oops.In("consolidation_number_repository.createNewSequence").
// 			With("year", year).
// 			With("month", month).
// 			Time(time.Now()).
// 			Wrapf(err, "invalid month value %d for sequence", month)
// 	}

// 	return &consolidation.ConsolidationSequence{
// 		OrganizationID:  orgID,
// 		Year:            yearInt16,
// 		Month:           monthInt16,
// 		CurrentSequence: 0,
// 	}, nil
// }

// // getSequence retrieves an existing sequence or creates a new one if it doesn't exist
// func (r *consolidationNumberRepository) getSequence(
// 	ctx context.Context,
// 	tx bun.Tx,
// 	orgID pulid.ID,
// 	year, month int,
// ) (*consolidation.ConsolidationSequence, error) {
// 	log := r.l.With().
// 		Str("operation", "getSequence").
// 		Str("orgID", orgID.String()).
// 		Int("year", year).
// 		Int("month", month).
// 		Logger()

// 	sequence := new(consolidation.ConsolidationSequence)
// 	err := tx.NewSelect().Model(sequence).
// 		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
// 			return sq.Where("cs.organization_id = ?", orgID).
// 				Where("cs.year = ?", year).
// 				Where("cs.month = ?", month)
// 		}).
// 		For("UPDATE").
// 		Scan(ctx)

// 	if err == nil {
// 		return sequence, nil
// 	}

// 	if !eris.Is(err, sql.ErrNoRows) {
// 		log.Error().Err(err).Msg("failed to get sequence")
// 		return nil, err
// 	}

// 	// Create a new sequence since it doesn't exist
// 	newSequence, err := r.createNewSequence(orgID, year, month)
// 	if err != nil {
// 		log.Error().Err(err).Msg("failed to create new sequence")
// 		return nil, err
// 	}

// 	// Insert the new sequence
// 	if _, err = tx.NewInsert().Model(newSequence).Exec(ctx); err != nil {
// 		log.Error().Err(err).Msg("failed to insert new sequence")
// 		return nil, err
// 	}

// 	// Fetch the inserted sequence to get the ID and other fields
// 	err = tx.NewSelect().Model(newSequence).
// 		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
// 			return sq.Where("cs.organization_id = ?", orgID).
// 				Where("cs.year = ?", year).
// 				Where("cs.month = ?", month)
// 		}).
// 		Scan(ctx)
// 	if err != nil {
// 		log.Error().Err(err).Msg("failed to fetch inserted sequence")
// 		return nil, err
// 	}

// 	return newSequence, nil
// }

// func (r *consolidationNumberRepository) incrementSequence(
// 	ctx context.Context, tx bun.IDB, sequence *consolidation.ConsolidationSequence,
// ) error {
// 	ov := sequence.Version
// 	sequence.Version++
// 	sequence.CurrentSequence++

// 	result, err := tx.NewUpdate().Model(sequence).
// 		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
// 			return uq.
// 				Where("cs.id = ?", sequence.ID).
// 				Where("cs.version = ?", ov)
// 		}).
// 		Returning("*").
// 		Exec(ctx)

// 	if err != nil {
// 		r.l.Error().Err(err).
// 			Str("operation", "incrementSequence").
// 			Str("sequenceID", sequence.ID.String()).
// 			Int("version", int(ov)).
// 			Msg("failed to update sequence")
// 		return oops.In("consolidation_number_repository.incrementSequence").
// 			With("sequence", sequence).
// 			Time(time.Now()).
// 			Wrap(err)
// 	}

// 	rows, err := result.RowsAffected()
// 	if err != nil {
// 		r.l.Error().Err(err).
// 			Str("operation", "incrementSequence").
// 			Str("sequenceID", sequence.ID.String()).
// 			Int("version", int(ov)).
// 			Msg("failed to get rows affected")
// 		return oops.In("consolidation_number_repository.incrementSequence").
// 			With("sequence", sequence).
// 			Time(time.Now()).
// 			Wrap(err)
// 	}

// 	if rows == 0 {
// 		return consolidation.ErrSequenceUpdateConflict
// 	}

// 	return nil
// }

// func (r *consolidationNumberRepository) getOrCreateAndIncrementSequence(
// 	ctx context.Context,
// 	dba bun.IDB,
// 	orgID pulid.ID,
// 	year, month, count int,
// ) (sequences []int64, err error) {
// 	log := r.l.With().
// 		Str("operation", "getOrCreateAndIncrementSequence").
// 		Str("orgID", orgID.String()).
// 		Int("year", year).
// 		Int("month", month).
// 		Int("count", count).
// 		Logger()

// 	sequence := new(consolidation.ConsolidationSequence)
// 	sequences = make([]int64, count)

// 	err = dba.RunInTx(
// 		ctx,
// 		&sql.TxOptions{Isolation: sql.LevelSerializable},
// 		func(c context.Context, tx bun.Tx) error {
// 			sequence, err = r.getSequence(c, tx, orgID, year, month)
// 			if err != nil {
// 				return oops.In("consolidation_number_repository.getOrCreateAndIncrementSequence").
// 					With("orgID", orgID.String()).
// 					With("year", year).
// 					With("month", month).
// 					With("count", count).
// 					Time(time.Now()).
// 					Wrap(err)
// 			}

// 			startSequence := sequence.CurrentSequence
// 			sequence.CurrentSequence += int64(count)
// 			sequence.Version++

// 			result, rErr := tx.NewUpdate().Model(sequence).
// 				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
// 					return uq.Where("cs.id = ?", sequence.ID).
// 						Where("cs.version = ?", sequence.Version-1)
// 				}).Exec(c)
// 			if rErr != nil {
// 				log.Error().Err(rErr).Msg("failed to update sequence")
// 				return oops.In("consolidation_number_repository.getOrCreateAndIncrementSequence").
// 					With("orgID", orgID.String()).
// 					With("year", year).
// 					With("month", month).
// 					With("count", count).
// 					Time(time.Now()).
// 					Wrap(rErr)
// 			}

// 			rows, roErr := result.RowsAffected()
// 			if roErr != nil {
// 				log.Error().Err(roErr).Msg("failed to get rows affected")
// 				return oops.In("consolidation_number_repository.getOrCreateAndIncrementSequence").
// 					With("orgID", orgID.String()).
// 					With("year", year).
// 					With("month", month).
// 					With("count", count).
// 					Time(time.Now()).
// 					Wrap(roErr)
// 			}

// 			if rows == 0 {
// 				return consolidation.ErrSequenceUpdateConflict
// 			}

// 			for i := range count {
// 				sequences[i] = startSequence + int64(i) + 1
// 			}

// 			return nil
// 		},
// 	)

// 	return sequences, err
// }
