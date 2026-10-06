package editioninfo_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/editioninfo"
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
