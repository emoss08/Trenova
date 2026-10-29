package benchsql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfinement_RefusesARoleThatSeesEveryOrganization(t *testing.T) {
	t.Parallel()

	require.NoError(t, confinement{Security: "on"}.err())
	require.ErrorIs(t, confinement{Bypass: true, Security: "on"}.err(), ErrSetupUnconfined)
	require.ErrorIs(t, confinement{Security: "off"}.err(), ErrSetupUnconfined)
}
