package seqgen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SequenceStoreParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type sequenceStore struct {
	db *postgres.Connection
	l  *zap.Logger
	// * Retry configuration
	maxRetries   int
	retryDelayMs int
}

func NewSequenceStore(p SequenceStoreParams) SequenceStore {
	return &sequenceStore{
		db:           p.DB,
		l:            p.Logger.With(zap.String("component", "sequenceStore")),
		maxRetries:   3,
		retryDelayMs: 50,
	}
}

func (s *sequenceStore) GetNextSequence(ctx context.Context, req *SequenceRequest) (int64, error) {
	var lastErr error

	for attempt := range s.maxRetries {
		if attempt > 0 {
			delay := time.Duration(s.retryDelayMs*(1<<(attempt-1))) * time.Millisecond
			time.Sleep(delay)
		}

		seq, err := s.getNextSequenceAttempt(ctx, req)
		if err == nil {
			return seq, nil
		}

		if !errors.Is(err, ErrSequenceUpdateConflict) {
			return 0, err
		}

		lastErr = err
		s.l.Debug(
			"sequence update conflict, retrying",
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}

	return 0, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (s *sequenceStore) getNextSequenceAttempt(
	ctx context.Context,
	req *SequenceRequest,
) (int64, error) {
	dba, err := s.db.DB(ctx)
	if err != nil {
		return 0, err
	}

	sequence := new(Sequence)
	var nextSeq int64

	err = dba.RunInTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
		func(c context.Context, tx bun.Tx) error {
			var getErr error
			sequence, getErr = s.getOrCreateSequence(c, tx, req)
			if getErr != nil {
				return getErr
			}

			nextSeq = sequence.CurrentSequence + 1

			if incrementErr := s.incrementSequence(c, tx, sequence, 1); incrementErr != nil {
				return incrementErr
			}

			return nil
		},
	)
	if err != nil {
		return 0, err
	}

	return nextSeq, nil
}

func (s *sequenceStore) GetNextSequenceBatch(
	ctx context.Context,
	req *SequenceRequest,
) ([]int64, error) {
	if req.Count <= 0 {
		return []int64{}, nil
	}

	var lastErr error

	for attempt := range s.maxRetries {
		if attempt > 0 {
			delay := time.Duration(s.retryDelayMs*(1<<(attempt-1))) * time.Millisecond
			time.Sleep(delay)
		}

		sequences, err := s.getNextSequenceBatchAttempt(ctx, req)
		if err == nil {
			return sequences, nil
		}

		if !errors.Is(err, ErrSequenceUpdateConflict) {
			return nil, err
		}

		lastErr = err
		s.l.Debug(
			"batch sequence update conflict, retrying",
			zap.Int("attempt", attempt+1),
			zap.Int("count", req.Count),
			zap.Error(err),
		)
	}

	return nil, fmt.Errorf("max retries exceeded for batch: %w", lastErr)
}

func (s *sequenceStore) getNextSequenceBatchAttempt(
	ctx context.Context,
	req *SequenceRequest,
) ([]int64, error) {
	dba, err := s.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	var sequences []int64

	err = dba.RunInTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
		func(c context.Context, tx bun.Tx) error {
			sequence, getErr := s.getOrCreateSequence(c, tx, req)
			if getErr != nil {
				return getErr
			}

			startSeq := sequence.CurrentSequence

			if incrementErr := s.incrementSequence(c, tx, sequence, req.Count); incrementErr != nil {
				return incrementErr
			}

			sequences = make([]int64, req.Count)
			for i := range req.Count {
				sequences[i] = startSeq + int64(i) + 1
			}

			return nil
		},
	)

	return sequences, err
}

func (s *sequenceStore) getOrCreateSequence(
	ctx context.Context,
	tx bun.Tx,
	req *SequenceRequest,
) (*Sequence, error) {
	sequence := new(Sequence)
	err := tx.NewSelect().Model(sequence).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = sq.Where("seq.sequence_type = ?", req.Type).
				Where("seq.organization_id = ?", req.OrgID).
				Where("seq.year = ?", req.Year).
				Where("seq.month = ?", req.Month)

			if !req.BuID.IsNil() {
				sq = sq.Where("seq.business_unit_id = ?", req.BuID)
			} else {
				sq = sq.Where("seq.business_unit_id IS NULL")
			}
			return sq
		}).
		For("UPDATE").
		Scan(ctx)

	if err == nil {
		return sequence, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get sequence: %w", err)
	}

	yearInt16, err := utils.SafeInt16(req.Year)
	if err != nil {
		return nil, fmt.Errorf("invalid year value %d: %w", req.Year, err)
	}

	monthInt16, err := utils.SafeInt16(req.Month)
	if err != nil {
		return nil, fmt.Errorf("invalid month value %d: %w", req.Month, err)
	}

	newSequence := &Sequence{
		SequenceType:    req.Type,
		OrganizationID:  req.OrgID,
		BusinessUnitID:  req.BuID,
		Year:            yearInt16,
		Month:           monthInt16,
		CurrentSequence: 0,
		Version:         0,
	}

	if _, err = tx.NewInsert().Model(newSequence).Exec(ctx); err != nil {
		return nil, err
	}

	err = tx.NewSelect().Model(newSequence).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = sq.Where("seq.sequence_type = ?", req.Type).
				Where("seq.organization_id = ?", req.OrgID).
				Where("seq.year = ?", req.Year).
				Where("seq.month = ?", req.Month)

			if !req.BuID.IsNil() {
				sq = sq.Where("seq.business_unit_id = ?", req.BuID)
			} else {
				sq = sq.Where("seq.business_unit_id IS NULL")
			}

			return sq
		}).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return newSequence, nil
}

func (s *sequenceStore) incrementSequence(
	ctx context.Context,
	tx bun.Tx,
	sequence *Sequence,
	incrementBy int,
) error {
	originalVersion := sequence.Version
	sequence.Version++
	sequence.CurrentSequence += int64(incrementBy)

	result, err := tx.NewUpdate().Model(sequence).
		Where("seq.id = ? AND seq.version = ?", sequence.ID, originalVersion).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrSequenceUpdateConflict
	}

	return nil
}
