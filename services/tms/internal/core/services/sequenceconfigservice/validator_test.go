package sequenceconfigservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestValidateUpdateAcceptsRequiredAccountingSequenceTypes(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	requiredTypes := tenant.RequiredSequenceTypes()
	configs := make([]*tenant.SequenceConfig, 0, len(requiredTypes))
	for _, sequenceType := range requiredTypes {
		configs = append(configs, tenant.DefaultSequenceConfig(orgID, buID, sequenceType))
	}

	validator := NewValidator()
	multiErr := validator.ValidateUpdate(t.Context(), &tenant.SequenceConfigDocument{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Configs:        configs,
	})

	require.Nil(t, multiErr)
}

func TestValidateUpdateRejectsMissingRequiredSequenceType(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	requiredTypes := tenant.RequiredSequenceTypes()
	configs := make([]*tenant.SequenceConfig, 0, len(requiredTypes)-1)
	for _, sequenceType := range requiredTypes[:len(requiredTypes)-1] {
		configs = append(configs, tenant.DefaultSequenceConfig(orgID, buID, sequenceType))
	}

	validator := NewValidator()
	multiErr := validator.ValidateUpdate(t.Context(), &tenant.SequenceConfigDocument{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Configs:        configs,
	})

	require.NotNil(t, multiErr)
	require.Contains(t, multiErr.Error(), string(tenant.SequenceTypeManualJournalRequest))
}
