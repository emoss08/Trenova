package objectkey_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/objectkey"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryFamilyKeyIsOwnedByItsOrganizationAlone(t *testing.T) {
	t.Parallel()

	org := pulid.MustNew("org_")
	other := pulid.MustNew("org_")
	for _, family := range objectkey.Families() {
		key := family.Key(org, "batch", "0001.pdf")
		owner, got, ok := objectkey.Owner(key)
		require.True(t, ok, key)
		assert.Equal(t, org, owner, key)
		assert.Equal(t, family.Name, got.Name, key)
		assert.True(t, objectkey.OwnedBy(key, org), key)
		assert.False(t, objectkey.OwnedBy(key, other), key)
		assert.Contains(t, objectkey.Prefixes(org), family.Prefix(org))
		assert.True(t, strings.HasPrefix(key, family.Prefix(org)))
	}
}

func TestOnlyCarriedFamiliesTravelInAnArchive(t *testing.T) {
	t.Parallel()

	org := pulid.MustNew("org_")
	carried := objectkey.CarriedPrefixes(org)
	assert.Contains(t, carried, objectkey.Organization.Prefix(org))
	assert.Contains(t, carried, objectkey.Capture.Prefix(org))
	assert.Contains(t, carried, objectkey.Reports.Prefix(org))
	assert.NotContains(t, carried, objectkey.AuditExports.Prefix(org))
	assert.NotContains(t, carried, objectkey.AIAuditExports.Prefix(org))
	assert.False(t, objectkey.CarriedBy(objectkey.AuditExports.Key(org, "x.csv"), org))
	assert.True(t, objectkey.CarriedBy(objectkey.Capture.Key(org, "b", "1.pdf"), org))
}

func TestRebaseKeepsAKeyOnlyUnderTheTenantsOwnPrefixes(t *testing.T) {
	t.Parallel()

	org := pulid.MustNew("org_")
	other := pulid.MustNew("org_")
	key := objectkey.Capture.Key(org, "bat_1", "0001-abc.pdf")
	rebased, ok := objectkey.Rebase(key, org)
	require.True(t, ok)
	assert.Equal(t, key, rebased)

	for _, hostile := range []string{
		objectkey.Organization.Key(other, "documents", "a.pdf"),
		org.String() + "/../" + other.String() + "/a.pdf",
		org.String() + "//a.pdf",
		"/" + org.String() + "/a.pdf",
		org.String() + "/",
		org.String(),
		"support/logs/bundle.zip",
		"capture/" + other.String() + "/b/1.pdf",
		"reports/" + org.String(),
		org.String() + "/a\x00.pdf",
		org.String() + "\\a.pdf",
		objectkey.AuditExports.Key(org, "x.csv"),
		"",
	} {
		_, ok := objectkey.Rebase(hostile, org)
		assert.False(t, ok, "%q", hostile)
	}
}

func TestANewKeyKeepsTheExtensionAndNeverTheName(t *testing.T) {
	t.Parallel()

	org := pulid.MustNew("org_")
	key := objectkey.Organization.NewKey(org, "shipment", "../../Bill of Lading.PDF")
	assert.True(t, strings.HasPrefix(key, org.String()+"/shipment/"))
	assert.True(t, strings.HasSuffix(key, ".pdf"))
	assert.NotContains(t, key, "Bill")
	assert.True(t, objectkey.OwnedBy(key, org))
}
