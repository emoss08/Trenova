package seqgen

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel"

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

	var current int64
	err := s.db.RunDetached(ctx, false, func(ctx context.Context) error {
		var err error
		current, err = keel.QueryValue[int64](ctx, s.db.Keel(ctx), `
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
		) VALUES ($1, $2, $3, $4, $5, $6, $7, '', 0, $8, $8)
		ON CONFLICT (sequence_type, organization_id, business_unit_id, year, month)
		DO UPDATE SET
			current_sequence = sequences.current_sequence + EXCLUDED.current_sequence,
			version = sequences.version + 1,
			updated_at = EXCLUDED.updated_at
		RETURNING current_sequence
	`,
			seqID.String(),
			string(req.Type),
			req.OrgID.String(),
			req.BuID.String(),
			req.Year,
			req.Month,
			req.Count,
			nowUnix,
		)
		return err
	})
	if err != nil {
		return nil, err
	}

	start := current - int64(req.Count) + 1
	sequences := make([]int64, req.Count)
	for i := range req.Count {
		sequences[i] = start + int64(i)
	}

	return sequences, nil
}

func (s *sequenceStore) UpdateLastGenerated(ctx context.Context, req *LastGeneratedRequest) error {
	return s.db.RunDetached(ctx, false, func(ctx context.Context) error {
		_, err := keel.Exec(ctx, s.db.Keel(ctx), `
		UPDATE sequences
		SET last_generated = $1, updated_at = $2
		WHERE sequence_type = $3
			AND organization_id = $4
			AND business_unit_id = $5
			AND year = $6
			AND month = $7`,
			req.Value,
			timeutils.NowUnix(),
			string(req.Type),
			req.OrgID.String(),
			req.BuID.String(),
			req.Year,
			req.Month,
		)
		return err
	})
}
