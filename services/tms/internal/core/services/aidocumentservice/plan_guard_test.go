package aidocumentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestRunStructuredRefusesWhenThePlanRestrictsDocumentIntelligence(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityDocumentIntelligence)}

	result, err := svc.runStructured(t.Context(), &structuredCall{tenant: plantest.Tenant()}, nil)

	require.Nil(t, result)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
