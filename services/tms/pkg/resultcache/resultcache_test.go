package resultcache

import (
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/stretchr/testify/assert"
)

func TestEveryClassKeepsItsTTLAndTheCallersOptions(t *testing.T) {
	t.Parallel()

	for name, class := range map[string]func(...keel.CacheOption) []keel.CacheOption{
		"settings":  Settings,
		"access":    Access,
		"reference": Reference,
		"catalog":   Catalog,
	} {
		assert.Lenf(t, class(), 1, "%s carries its TTL", name)
		assert.Lenf(t, class(keel.AlsoReads("us_states")), 2, "%s keeps AlsoReads", name)
	}
}

func TestSettingsExpireBeforeReferenceData(t *testing.T) {
	t.Parallel()

	assert.Less(t, SettingsTTL, ReferenceTTL)
	assert.Less(t, ReferenceTTL, CatalogTTL)
	assert.LessOrEqual(t, AccessTTL, 10*time.Minute)
}
