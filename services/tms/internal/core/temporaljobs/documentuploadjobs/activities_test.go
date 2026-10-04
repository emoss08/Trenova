package documentuploadjobs

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestPlanRefusalRecognisesQuotaAndPlanErrors(t *testing.T) {
	t.Parallel()

	assert.True(t, planRefusal(errortypes.NewQuotaExceededError("documents.uploads", 25, 25, "free_demo")))
	assert.True(t, planRefusal(errortypes.NewPlanRestrictionError("", "subscription_read_only", "free_demo")))
	assert.False(t, planRefusal(errors.New("storage unavailable")))
}
