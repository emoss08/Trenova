package seqgen

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SequenceStoreParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type sequenceStore struct {
	db         *postgres.Connection
	l          *zap.Logger
	maxRetries int
}

func NewSequenceStore(p SequenceStoreParams) SequenceStore {
	return &sequenceStore{
		db:         p.DB,
		l:          p.Logger.Named("seq-store"),
		maxRetries: 3,
	}
}

func (s *sequenceStore) GetNextSequence(ctx context.Context, req *SequenceRequest) (int64, error) {
	sequences, err := s.GetNextSequenceBatch(ctx, &SequenceRequest{
		Type:  req.Type,
		OrgID: req.OrgID,
		BuID:  req.BuID,
		Year:  req.Year,
		Month: req.Month,
		Count: 1,
	})
	if err != nil {
		return 0, err
	}
	if len(sequences) == 0 {
		return 0, ErrNoSequencesReturned
	}

	return sequences[0], nil
}

func (s *sequenceStore) GetNextSequenceBatch(
	ctx context.Context,
	req *SequenceRequest,
) ([]int64, error) {
	if req.Count <= 0 {
		return []int64{}, nil
	}

	var lastErr error
	for attempt := 1; attempt <= s.maxRetries; attempt++ {
		sequences, err := s.getNextSequenceBatchAttempt(ctx, req)
		if err == nil {
			return sequences, nil
		}

		lastErr = err
		if !dberror.IsRetryableTransactionError(err) || attempt == s.maxRetries {
			break
		}

		time.Sleep(time.Duration(attempt*25) * time.Millisecond)
	}

	return nil, fmt.Errorf("get next sequence batch: %w", lastErr)
}

func (s *sequenceStore) getNextSequenceBatchAttempt(
	ctx context.Context,
	req *SequenceRequest,
) ([]int64, error) {
	nowUnix := timeutils.NowUnix()
	seqID := pulid.MustNew("seq_")

	type sequenceResult struct {
		CurrentSequence int64 `bun:"current_sequence"`
	}

	result := new(sequenceResult)
	err := s.db.DB().NewRaw(`
		INSERT INTO sequences (
			id,
			sequence_type,
			organization_id,
			business_unit_id,
			year,
			month,
			current_sequence,
			last_generated,
			version,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, '', 0, ?, ?)
		ON CONFLICT (sequence_type, organization_id, business_unit_id, year, month)
		DO UPDATE SET
			current_sequence = sequences.current_sequence + EXCLUDED.current_sequence,
			version = sequences.version + 1,
			updated_at = EXCLUDED.updated_at
		RETURNING current_sequence
	`,
		seqID,
		req.Type,
		req.OrgID,
		req.BuID,
		req.Year,
		req.Month,
		req.Count,
		nowUnix,
		nowUnix,
	).Scan(ctx, result)
	if err != nil {
		return nil, err
	}

	start := result.CurrentSequence - int64(req.Count) + 1
	sequences := make([]int64, req.Count)
	for i := range req.Count {
		sequences[i] = start + int64(i)
	}

	return sequences, nil
}

func (s *sequenceStore) UpdateLastGenerated(ctx context.Context, req *LastGeneratedRequest) error {
	_, err := s.db.DB().NewUpdate().
		Table("sequences").
		Set("last_generated = ?", req.Value).
		Set("updated_at = ?", timeutils.NowUnix()).
		Where("sequence_type = ?", req.Type).
		Where("organization_id = ?", req.OrgID).
		Where("business_unit_id = ?", req.BuID).
		Where("year = ?", req.Year).
		Where("month = ?", req.Month).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
