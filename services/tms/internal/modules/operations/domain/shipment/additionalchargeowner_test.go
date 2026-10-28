package shipment

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestAdditionalCharge_OwnerNamesExactlyOneEngine(t *testing.T) {
	t.Parallel()

	programID := pulid.MustNew("fsp_")
	accessorialID := pulid.MustNew("raga_")

	require.Equal(t, SystemOwnerNone, (&AdditionalCharge{}).Owner(),
		"a charge nobody generated has no owner")
	require.Equal(t, SystemOwnerNone,
		(&AdditionalCharge{IsDetention: true}).Owner(),
		"an owner column on a manual charge does not make it system owned")

	require.Equal(t, SystemOwnerFuel, (&AdditionalCharge{
		IsSystemGenerated:      true,
		FuelSurchargeProgramID: &programID,
	}).Owner())
	require.Equal(t, SystemOwnerDetention, (&AdditionalCharge{
		IsSystemGenerated: true,
		IsDetention:       true,
	}).Owner())
	require.Equal(t, SystemOwnerAgreement, (&AdditionalCharge{
		IsSystemGenerated:          true,
		RateAgreementAccessorialID: &accessorialID,
	}).Owner())
}
