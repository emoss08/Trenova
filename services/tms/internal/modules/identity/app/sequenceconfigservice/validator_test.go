package sequenceconfigservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestValidateUpdateAcceptsRequiredAccountingSequenceTypes(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	requiredTypes := sequenceconfig.RequiredSequenceTypes()
	configs := make([]*sequenceconfig.SequenceConfig, 0, len(requiredTypes))
	for _, sequenceType := range requiredTypes {
		configs = append(configs, sequenceconfig.DefaultSequenceConfig(orgID, buID, sequenceType))
	}

	validator := NewValidator()
	multiErr := validator.ValidateUpdate(t.Context(), &sequenceconfig.SequenceConfigDocument{
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
	requiredTypes := sequenceconfig.RequiredSequenceTypes()
	configs := make([]*sequenceconfig.SequenceConfig, 0, len(requiredTypes)-1)
	for _, sequenceType := range requiredTypes[:len(requiredTypes)-1] {
		configs = append(configs, sequenceconfig.DefaultSequenceConfig(orgID, buID, sequenceType))
	}

	validator := NewValidator()
	multiErr := validator.ValidateUpdate(t.Context(), &sequenceconfig.SequenceConfigDocument{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Configs:        configs,
	})

	require.NotNil(t, multiErr)
	require.Contains(t, multiErr.Error(), string(requiredTypes[len(requiredTypes)-1]))
}
