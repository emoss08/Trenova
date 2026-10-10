package carriersettlementservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type periodBatchRepoStub struct {
	repositories.CarrierSettlementBatchRepository
	existing *carriersettlement.CarrierSettlementBatch
}

func (s *periodBatchRepoStub) GetForPeriod(
	context.Context,
	pagination.TenantInfo,
	int64,
	int64,
) (*carriersettlement.CarrierSettlementBatch, error) {
	return s.existing, nil
}

func TestResolveOpenBatchRefusesAClosedPeriodBatch(t *testing.T) {
	t.Parallel()

	for _, status := range []carriersettlement.BatchStatus{
		carriersettlement.BatchStatusCompleted,
		carriersettlement.BatchStatusCanceled,
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			svc := &Service{
				l: zap.NewNop(),
				batchRepo: &periodBatchRepoStub{existing: &carriersettlement.CarrierSettlementBatch{
					ID:     pulid.MustNew("csb_"),
					Status: status,
				}},
			}

			batch, err := svc.resolveOpenBatch(
				t.Context(),
				&GenerateBatchRequest{TenantInfo: pagination.TenantInfo{}},
				PeriodBounds{PeriodStart: 1, PeriodEnd: 2},
				nil,
				3,
			)
			require.Error(t, err)
			assert.Nil(t, batch)
			assert.True(t, errors.Is(err, settlementshared.ErrPeriodBatchClosed))

			var validationErr *errortypes.Error
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, errortypes.ErrInvalidOperation, validationErr.Code)
		})
	}
}

func TestResolveOpenBatchReusesTheOpenPeriodBatch(t *testing.T) {
	t.Parallel()

	open := &carriersettlement.CarrierSettlementBatch{
		ID:     pulid.MustNew("csb_"),
		Status: carriersettlement.BatchStatusOpen,
	}
	svc := &Service{l: zap.NewNop(), batchRepo: &periodBatchRepoStub{existing: open}}

	batch, err := svc.resolveOpenBatch(
		t.Context(),
		&GenerateBatchRequest{},
		PeriodBounds{PeriodStart: 1, PeriodEnd: 2},
		nil,
		3,
	)
	require.NoError(t, err)
	assert.Same(t, open, batch)
}
