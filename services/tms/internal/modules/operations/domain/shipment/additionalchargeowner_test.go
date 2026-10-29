package shipment

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestAdditionalCharge_OwnerNamesExactlyOneEngine(t *testing.T) {
	t.Parallel()

	programID := pulid.MustNew("fsp_")
	accessorialID := pulid.MustNew("raga_")

	require.Equal(t, shipmentvocab.SystemOwnerNone, (&AdditionalCharge{}).Owner(),
		"a charge nobody generated has no owner")
	require.Equal(t, shipmentvocab.SystemOwnerNone,
		(&AdditionalCharge{IsDetention: true}).Owner(),
		"an owner column on a manual charge does not make it system owned")

	require.Equal(t, shipmentvocab.SystemOwnerFuel, (&AdditionalCharge{
		IsSystemGenerated:      true,
		FuelSurchargeProgramID: &programID,
	}).Owner())
	require.Equal(t, shipmentvocab.SystemOwnerDetention, (&AdditionalCharge{
		IsSystemGenerated: true,
		IsDetention:       true,
	}).Owner())
	require.Equal(t, shipmentvocab.SystemOwnerAgreement, (&AdditionalCharge{
		IsSystemGenerated:          true,
		RateAgreementAccessorialID: &accessorialID,
	}).Owner())
}
