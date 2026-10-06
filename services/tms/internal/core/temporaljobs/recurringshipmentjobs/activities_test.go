package recurringshipmentjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type refusingRepo struct {
	repositories.RecurringShipmentRepository

	failures int
}

func (r *refusingRepo) Generate(
	context.Context,
	*repositories.GenerateRecurringShipmentRequest,
) (*repositories.GenerateRecurringShipmentResult, error) {
	return nil, errortypes.NewPlanRestrictionError("", platformplan.ReasonSubscriptionReadOnly, "free_demo")
}

func (r *refusingRepo) RecordGenerationFailure(
	context.Context,
	*repositories.RecordRecurringGenerationFailureRequest,
) (*recurringshipment.RecurringShipment, error) {
	r.failures++
	return &recurringshipment.RecurringShipment{}, nil
}

func TestDispatchSeriesSkipsAReadOnlyOrganizationWithoutRecordingAFailure(t *testing.T) {
	t.Parallel()

	repo := &refusingRepo{}
	a := &Activities{repo: repo, logger: zap.NewNop()}
	result := &DispatchDueRecurringShipmentsResult{}

	a.dispatchSeries(t.Context(), &recurringshipment.RecurringShipment{
		ID:             pulid.MustNew("rsh_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		EnteredByID:    pulid.MustNew("usr_"),
	}, result)

	assert.Equal(t, 1, result.Skipped)
	assert.Zero(t, result.Failed)
	assert.Zero(t, repo.failures)
}
