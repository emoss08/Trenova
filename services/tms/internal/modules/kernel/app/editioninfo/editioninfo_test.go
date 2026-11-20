package editioninfo_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/app/editioninfo"
	"github.com/stretchr/testify/assert"
)

func TestSelfHostedDoesNotShareTenancy(t *testing.T) {
	t.Parallel()

	info := editioninfo.SelfHosted()
	assert.Equal(t, editioninfo.SelfHostedName, info.Name())
	assert.False(t, info.SharedTenancy())
}

func TestStaticReportsWhatItWasGiven(t *testing.T) {
	t.Parallel()

	info := editioninfo.NewStatic("cloud", true)
	assert.Equal(t, "cloud", info.Name())
	assert.True(t, info.SharedTenancy())
}

func TestTheEditionFollowsTheManagedTenancy(t *testing.T) {
	t.Parallel()

	selfHosted := editioninfo.New(&managedconfig.Config{})
	assert.Equal(t, editioninfo.SelfHostedName, selfHosted.Name())
	assert.False(t, selfHosted.SharedTenancy())

	dedicated := editioninfo.New(&managedconfig.Config{
		Enabled: true,
		Tenancy: managedconfig.TenancyDedicated,
	})
	assert.Equal(t, editioninfo.ManagedName, dedicated.Name())
	assert.False(t, dedicated.SharedTenancy())

	shared := editioninfo.New(&managedconfig.Config{
		Enabled: true,
		Tenancy: managedconfig.TenancyShared,
	})
	assert.Equal(t, editioninfo.SharedName, shared.Name())
	assert.True(t, shared.SharedTenancy())
}
