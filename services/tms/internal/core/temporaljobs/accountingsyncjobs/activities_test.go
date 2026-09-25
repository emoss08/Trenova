package accountingsyncjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

type fakeMappingService struct {
	services.AccountingMappingService
	modelErr error
	applied  int
}

func (f *fakeMappingService) ModelPass(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	[]pulid.ID,
) (int, error) {
	return f.applied, f.modelErr
}

func TestReferenceErrorStopsRetryingWhatAPersonMustFix(t *testing.T) {
	t.Parallel()

	var appErr *temporal.ApplicationError
	business := referenceError(errortypes.NewBusinessError("QuickBooks Online is not connected"))
	require.ErrorAs(t, business, &appErr)
	assert.True(t, appErr.NonRetryable())

	transient := errors.New("connection reset by peer")
	assert.Same(t, transient, referenceError(transient))
	assert.NoError(t, referenceError(nil))
}

func TestModelPassActivityFallsBackOnTheFinalAttempt(t *testing.T) {
	t.Parallel()

	fake := &fakeMappingService{modelErr: errors.New("model overloaded"), applied: 1}
	a := NewActivities(ActivitiesParams{Mappings: fake, Logger: zap.NewNop()})

	applied, err := a.AccountingMappingModelPassActivity(t.Context(), refreshPayload(), []pulid.ID{pulid.MustNew("acctm_")})
	require.NoError(t, err, "outside a retrying activity every attempt is the last")
	assert.Equal(t, 1, applied)

	fake.modelErr = nil
	fake.applied = 3
	applied, err = a.AccountingMappingModelPassActivity(t.Context(), refreshPayload(), nil)
	require.NoError(t, err)
	assert.Equal(t, 3, applied)
}
