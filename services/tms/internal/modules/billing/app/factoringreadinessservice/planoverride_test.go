package factoringreadinessservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPlanOverride_IsRefusedWithoutFactoring(t *testing.T) {
	t.Parallel()

	svc := NewReadinessService(ReadinessParams{
		Features: plantest.Without(platformcatalog.FeatureFactoring),
		Logger:   zap.NewNop(),
	})

	_, err := svc.PlanOverride(t.Context(), &factoringports.SetOverrideRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		InvoiceID: pulid.MustNew("inv_"),
		Override:  invoice.FactoringOverrideExclude,
		Reason:    "customer pays us directly",
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsPlanRestrictionError(err))
}
