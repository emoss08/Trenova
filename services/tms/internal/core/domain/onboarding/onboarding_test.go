package onboarding_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/onboarding"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func errorFields(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	return fields
}

func TestNewPendingIsValid(t *testing.T) {
	t.Parallel()

	entity := onboarding.NewPending(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	assert.Equal(t, onboarding.StatusPending, entity.Status)
	assert.False(t, entity.IsCompleted())

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}

func TestCompleteRecordsTheWizard(t *testing.T) {
	t.Parallel()

	entity := onboarding.NewPending(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	userID := pulid.MustNew("usr_")
	entity.Complete(onboarding.CompleteParams{
		UserID:           userID,
		OperationType:    onboarding.OperationTypeBoth,
		SampleDataLoaded: true,
		CompletedAt:      1_700,
	})

	assert.True(t, entity.IsCompleted())
	assert.Equal(t, onboarding.OperationTypeBoth, entity.OperationType)
	assert.True(t, entity.SampleDataLoaded)
	require.NotNil(t, entity.CompletedAt)
	assert.Equal(t, int64(1_700), *entity.CompletedAt)
	assert.Equal(t, userID, entity.CompletedByID)

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}

func TestCompletedRequiresOperationType(t *testing.T) {
	t.Parallel()

	entity := onboarding.NewPending(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	entity.Complete(onboarding.CompleteParams{UserID: pulid.MustNew("usr_")})

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	assert.Contains(t, errorFields(multiErr), "operationType")
	require.NotNil(t, entity.CompletedAt, "a zero completion time is filled with now")
}

func TestInvalidOperationTypeAndStatus(t *testing.T) {
	t.Parallel()

	entity := onboarding.NewPending(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	entity.OperationType = "rail"
	entity.Status = "skipped"

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	fields := errorFields(multiErr)
	assert.Contains(t, fields, "operationType")
	assert.Contains(t, fields, "status")
}

func TestOperationTypeCapabilities(t *testing.T) {
	t.Parallel()

	assert.True(t, onboarding.OperationTypeAsset.RunsAssets())
	assert.False(t, onboarding.OperationTypeAsset.RunsBrokerage())
	assert.True(t, onboarding.OperationTypeBrokerage.RunsBrokerage())
	assert.False(t, onboarding.OperationTypeBrokerage.RunsAssets())
	assert.True(t, onboarding.OperationTypeBoth.RunsAssets())
	assert.True(t, onboarding.OperationTypeBoth.RunsBrokerage())
}
