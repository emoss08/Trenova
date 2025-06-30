package seqgen

import (
	"context"
	"database/sql"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/sequencestore"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/intutils"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

// Errors
var (
	ErrSequenceUpdateConflict = eris.New("sequence update conflict")
	ErrInvalidSequenceType    = eris.New("invalid sequence type")
)

// sequenceStore implements SequenceStore interface
type sequenceStore struct {
	db db.Connection
	l  *zerolog.Logger
	// * Retry configuration
	maxRetries   int
	retryDelayMs int
}

// NewSequenceStore creates a new sequence store
func NewSequenceStore(dbConn db.Connection, log *logger.Logger) SequenceStore {
	l := log.With().
		Str("component", "sequenceStore").
		Logger()

	return &sequenceStore{
		db:           dbConn,
		l:            &l,
		maxRetries:   3,
		retryDelayMs: 50,
	}
}

// GetNextSequence gets the next sequence number with retry logic
func (s *sequenceStore) GetNextSequence(ctx context.Context, req *SequenceRequest) (int64, error) {
	var lastErr error

	for attempt := range s.maxRetries {
		if attempt > 0 {
			// * Exponential backoff with jitter
			delay := time.Duration(s.retryDelayMs*(1<<(attempt-1))) * time.Millisecond
			time.Sleep(delay)
		}

		seq, err := s.getNextSequenceAttempt(ctx, req)
		if err == nil {
			return seq, nil
		}

		if !eris.Is(err, ErrSequenceUpdateConflict) {
			// * Non-retryable error
			return 0, err
		}

		lastErr = err
		s.l.Debug().
			Int("attempt", attempt+1).
			Err(err).
			Msg("sequence update conflict, retrying")
	}

	return 0, eris.Wrap(lastErr, "max retries exceeded")
}

// getNextSequenceAttempt performs a single attempt to get the next sequence
func (s *sequenceStore) getNextSequenceAttempt(
	ctx context.Context,
	req *SequenceRequest,
) (int64, error) {
	dba, err := s.db.DB(ctx)
	if err != nil {
		return 0, eris.Wrap(err, "get database connection")
	}

	sequence := new(sequencestore.Sequence)
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

			// * Store the next sequence number
			nextSeq = sequence.CurrentSequence + 1

			// * Increment with optimistic locking
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

// GetNextSequenceBatch gets a batch of sequence numbers
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

		if !eris.Is(err, ErrSequenceUpdateConflict) {
			return nil, err
		}

		lastErr = err
		s.l.Debug().
			Int("attempt", attempt+1).
			Int("count", req.Count).
			Err(err).
			Msg("batch sequence update conflict, retrying")
	}

	return nil, eris.Wrap(lastErr, "max retries exceeded for batch")
}

// getNextSequenceBatchAttempt performs a single attempt to get a batch of sequences
func (s *sequenceStore) getNextSequenceBatchAttempt(
	ctx context.Context,
	req *SequenceRequest,
) ([]int64, error) {
	dba, err := s.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
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

			// * Store the starting sequence
			startSeq := sequence.CurrentSequence

			// * Increment by count
			if incrementErr := s.incrementSequence(c, tx, sequence, req.Count); incrementErr != nil {
				return incrementErr
			}

			// * Generate the sequence numbers
			sequences = make([]int64, req.Count)
			for i := range req.Count {
				sequences[i] = startSeq + int64(i) + 1
			}

			return nil
		},
	)

	return sequences, err
}

// getOrCreateSequence retrieves or creates a sequence
func (s *sequenceStore) getOrCreateSequence(
	ctx context.Context,
	tx bun.Tx,
	req *SequenceRequest,
) (*sequencestore.Sequence, error) {
	sequence := new(sequencestore.Sequence)
	err := tx.NewSelect().Model(sequence).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = sq.Where("seq.sequence_type = ?", req.Type).
				Where("seq.organization_id = ?", req.OrganizationID).
				Where("seq.year = ?", req.Year).
				Where("seq.month = ?", req.Month)

			if !req.BusinessUnitID.IsNil() {
				sq = sq.Where("seq.business_unit_id = ?", req.BusinessUnitID)
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

	if !eris.Is(err, sql.ErrNoRows) {
		return nil, eris.Wrap(err, "get sequence")
	}

	// * Create new sequence
	yearInt16, err := intutils.SafeInt16(req.Year)
	if err != nil {
		return nil, eris.Wrapf(err, "invalid year value %d", req.Year)
	}

	monthInt16, err := intutils.SafeInt16(req.Month)
	if err != nil {
		return nil, eris.Wrapf(err, "invalid month value %d", req.Month)
	}

	newSequence := &sequencestore.Sequence{
		SequenceType:    req.Type,
		OrganizationID:  req.OrganizationID,
		BusinessUnitID:  req.BusinessUnitID,
		Year:            yearInt16,
		Month:           monthInt16,
		CurrentSequence: 0,
		Version:         0,
	}

	if _, err = tx.NewInsert().Model(newSequence).Exec(ctx); err != nil {
		return nil, eris.Wrap(err, "insert new sequence")
	}

	// * Fetch the inserted sequence
	err = tx.NewSelect().Model(newSequence).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = sq.Where("seq.sequence_type = ?", req.Type).
				Where("seq.organization_id = ?", req.OrganizationID).
				Where("seq.year = ?", req.Year).
				Where("seq.month = ?", req.Month)

			if !req.BusinessUnitID.IsNil() {
				sq = sq.Where("seq.business_unit_id = ?", req.BusinessUnitID)
			} else {
				sq = sq.Where("seq.business_unit_id IS NULL")
			}

			return sq
		}).
		Scan(ctx)

	if err != nil {
		return nil, eris.Wrap(err, "fetch inserted sequence")
	}

	return newSequence, nil
}

// incrementSequence increments the sequence with optimistic locking
func (s *sequenceStore) incrementSequence(
	ctx context.Context,
	tx bun.Tx,
	sequence *sequencestore.Sequence,
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
		return eris.Wrap(err, "update sequence")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return eris.Wrap(err, "get rows affected")
	}

	if rows == 0 {
		return ErrSequenceUpdateConflict
	}

	return nil
}
