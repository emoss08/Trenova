package driversettlementservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type periodBatchRepoStub struct {
	repositories.SettlementBatchRepository
	existing *driversettlement.SettlementBatch
}

func (s *periodBatchRepoStub) GetForPeriod(
	context.Context,
	pagination.TenantInfo,
	int64,
	int64,
) (*driversettlement.SettlementBatch, error) {
	return s.existing, nil
}

func TestOpenBatchForPeriodMarksAClosedBatch(t *testing.T) {
	t.Parallel()

	svc := &Service{batchRepo: &periodBatchRepoStub{existing: &driversettlement.SettlementBatch{
		ID:     pulid.MustNew("sb_"),
		Status: driversettlement.BatchStatusCompleted,
	}}}

	batch, err := svc.openBatchForPeriod(
		t.Context(),
		pagination.TenantInfo{},
		PeriodBounds{PeriodStart: 1, PeriodEnd: 2},
	)
	require.Error(t, err)
	assert.Nil(t, batch)
	assert.True(t, errors.Is(err, settlementshared.ErrPeriodBatchClosed))

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, errortypes.ErrInvalidOperation, validationErr.Code)
}

func TestOpenBatchForPeriodReturnsTheOpenBatch(t *testing.T) {
	t.Parallel()

	open := &driversettlement.SettlementBatch{
		ID:     pulid.MustNew("sb_"),
		Status: driversettlement.BatchStatusOpen,
	}
	svc := &Service{batchRepo: &periodBatchRepoStub{existing: open}}

	batch, err := svc.openBatchForPeriod(
		t.Context(),
		pagination.TenantInfo{},
		PeriodBounds{PeriodStart: 1, PeriodEnd: 2},
	)
	require.NoError(t, err)
	assert.Same(t, open, batch)
}
