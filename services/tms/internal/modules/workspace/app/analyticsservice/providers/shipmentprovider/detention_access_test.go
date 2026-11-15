package shipmentprovider

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func detentionReads(
	t *testing.T,
	gate platformcatalogports.FeatureGate,
	denied ...permission.Resource,
) bool {
	t.Helper()

	registry := permissioncatalog.Default()
	provider := &Provider{
		views: refview.NewPort(
			refview.New(registry, onplan.New(registry, gate)),
			&readerEngine{denied: denied},
		),
		fieldAccess: groupAccess{},
	}
	opts := &kernelports.AnalyticsRequestOptions{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}

	reads, err := provider.accessFor(t.Context(), opts).readsDetention(t.Context())
	require.NoError(t, err)

	return reads
}

func TestDetentionWatchlist_IsReadForADetentionReaderOnAPlanWithDetention(t *testing.T) {
	t.Parallel()

	assert.True(t, detentionReads(t, plantest.AllFeatures()))
}

func TestDetentionWatchlist_IsNotReadWhenThePlanLeavesDetentionOut(t *testing.T) {
	t.Parallel()

	assert.False(t, detentionReads(t, plantest.Without(platformcatalog.FeatureDetention)))
	assert.False(t, detentionReads(t, plantest.BaseOnly()))
}

func TestDetentionWatchlist_IsNotReadForAReaderWithoutDetentionRead(t *testing.T) {
	t.Parallel()

	assert.False(t, detentionReads(t, plantest.AllFeatures(), permission.ResourceDetention))
}
