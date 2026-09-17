package integrationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func newTestControl() *carrierintel.CarrierIntelControl {
	return carrierintel.NewDefaultControl(pulid.MustNew("org_"), pulid.MustNew("bu_"))
}

func TestApplyCarrierIntelRole(t *testing.T) {
	t.Parallel()

	t.Run("enabling a provider with no primary makes it primary", func(t *testing.T) {
		t.Parallel()

		control := newTestControl()
		dirty := applyCarrierIntelRole(control, &carrierIntelRoleChange{
			typ:     integration.TypeCarrierOK,
			enabled: true,
		})
		assert.True(t, dirty)
		provider, ok := control.PrimaryType()
		assert.True(t, ok)
		assert.Equal(t, integration.TypeCarrierOK, provider)
		assert.Equal(t, int64(2), control.PolicyVersion)
	})

	t.Run("FMCSA as fallback never becomes primary", func(t *testing.T) {
		t.Parallel()

		control := newTestControl()
		primary := integration.TypeCarrierOK
		control.PrimaryProvider = &primary

		dirty := applyCarrierIntelRole(control, &carrierIntelRoleChange{
			typ:     integration.TypeFMCSAQCMobile,
			enabled: true,
			role:    integration.CarrierIntelRoleFallback,
		})
		assert.True(t, dirty)
		current, _ := control.PrimaryType()
		fallback, ok := control.FallbackType()
		assert.Equal(t, integration.TypeCarrierOK, current)
		assert.True(t, ok)
		assert.Equal(t, integration.TypeFMCSAQCMobile, fallback)
	})

	t.Run("switching FMCSA from primary to fallback clears primary", func(t *testing.T) {
		t.Parallel()

		control := newTestControl()
		primary := integration.TypeFMCSAQCMobile
		control.PrimaryProvider = &primary

		applyCarrierIntelRole(control, &carrierIntelRoleChange{
			typ:     integration.TypeFMCSAQCMobile,
			enabled: true,
			role:    integration.CarrierIntelRoleFallback,
		})
		_, hasPrimary := control.PrimaryType()
		assert.False(t, hasPrimary)
		fallback, _ := control.FallbackType()
		assert.Equal(t, integration.TypeFMCSAQCMobile, fallback)
	})

	t.Run("disabling the primary clears it", func(t *testing.T) {
		t.Parallel()

		control := newTestControl()
		primary := integration.TypeCarrierOK
		control.PrimaryProvider = &primary

		dirty := applyCarrierIntelRole(control, &carrierIntelRoleChange{
			typ:     integration.TypeCarrierOK,
			enabled: false,
		})
		assert.True(t, dirty)
		_, hasPrimary := control.PrimaryType()
		assert.False(t, hasPrimary)
	})

	t.Run("enabling a second provider leaves the existing primary", func(t *testing.T) {
		t.Parallel()

		control := newTestControl()
		primary := integration.TypeCarrierOK
		control.PrimaryProvider = &primary

		dirty := applyCarrierIntelRole(control, &carrierIntelRoleChange{
			typ:     integration.TypeFMCSAQCMobile,
			enabled: true,
			role:    integration.CarrierIntelRolePrimary,
		})
		assert.False(t, dirty)
		current, _ := control.PrimaryType()
		assert.Equal(t, integration.TypeCarrierOK, current)
	})
}
