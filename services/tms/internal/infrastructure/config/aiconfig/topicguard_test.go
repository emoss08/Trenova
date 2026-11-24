package aiconfig_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/stretchr/testify/assert"
)

func TestTopicGuard_IsOnUnlessTurnedOff(t *testing.T) {
	t.Parallel()

	off := false
	var unset *aiconfig.Config

	assert.True(t, unset.TopicGuardEnabled())
	assert.True(t, (&aiconfig.Config{}).TopicGuardEnabled())
	assert.False(t, (&aiconfig.Config{TopicGuard: &off}).TopicGuardEnabled())
}

func TestTopicGuard_CannotBeOffInProduction(t *testing.T) {
	t.Parallel()

	off := false

	assert.NoError(t, (&aiconfig.Config{}).ValidateProduction())
	assert.Error(t, (&aiconfig.Config{TopicGuard: &off}).ValidateProduction())
}
