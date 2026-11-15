package driverportalservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func withoutDriverPay() *Service {
	return New(Params{
		Logger:   zap.NewNop(),
		Features: plantest.Without(platformcatalog.FeatureDriverPortal),
	})
}

func TestTheDriverAppIsRefusedWithoutDriverPay(t *testing.T) {
	t.Parallel()

	_, err := withoutDriverPay().ResolveWorker(t.Context(), plantest.Tenant())

	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.True(t, ok, "expected PLAN_RESTRICTED, got %v", err)
	assert.Equal(t, string(platformcatalog.FeatureDriverPortal), restriction.Feature)
}

func TestADriverCannotAcknowledgeAnAssignmentWithoutDriverPay(t *testing.T) {
	t.Parallel()

	err := withoutDriverPay().RespondToMyAssignment(
		t.Context(),
		plantest.Tenant(),
		&RespondToAssignmentRequest{AssignmentID: pulid.MustNew("asn_"), Accept: true},
	)

	require.True(t, errortypes.IsPlanRestrictionError(err), "got %v", err)
}
