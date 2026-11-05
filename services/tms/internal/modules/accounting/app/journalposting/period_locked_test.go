package journalposting

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type lockedPeriods struct {
	PeriodReader
}

func (lockedPeriods) GetPeriodByDate(
	context.Context,
	fiscalperiodports.GetPeriodByDateRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	return &fiscalperiod.FiscalPeriod{Status: fiscalperiod.StatusLocked}, nil
}

func TestResolvePeriod_LockedPeriodFollowsThePolicy(t *testing.T) {
	t.Parallel()

	allowed, err := ResolvePeriod(t.Context(), lockedPeriods{}, &ResolvePeriodRequest{Date: 100})
	require.NoError(t, err)
	assert.Equal(t, int64(100), allowed.AccountingDate)

	_, err = ResolvePeriod(t.Context(), lockedPeriods{}, &ResolvePeriodRequest{
		Date:         100,
		RefuseLocked: true,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}
