package triagecache

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
)

var tenantInfo = pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"}

func TestKeysAreScopedToTheTenant(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "triage:epoch:org_1:bu_1", EpochKey(tenantInfo))
	assert.Equal(t, "triage:rules:org_1:bu_1", RulesKey(tenantInfo))
	assert.Equal(t, "triage:hints:org_1:bu_1:acme.com", HintsKey(tenantInfo, "ACME.com"))
}

func TestSectionKeysCarryTheEpoch(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "triage:briefing:org_1:bu_1:7", SectionKey(&triageports.CacheKey{
		TenantInfo: tenantInfo,
		Section:    triageports.CacheBriefing,
		Epoch:      7,
	}))
	assert.Equal(t, "triage:preview:org_1:bu_1:8:abc", SectionKey(&triageports.CacheKey{
		TenantInfo: tenantInfo,
		Section:    triageports.CachePreview,
		Epoch:      8,
		Variant:    "abc",
	}))
}
