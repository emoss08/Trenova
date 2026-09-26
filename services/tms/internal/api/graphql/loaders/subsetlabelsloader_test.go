package loaders

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingLabeler struct {
	mu      sync.Mutex
	labels  services.RecordLabels
	err     error
	calls   []map[permission.Resource][]pulid.ID
	tenants []pagination.TenantInfo
}

func (l *countingLabeler) Labels(
	_ context.Context,
	tenant pagination.TenantInfo,
	refs map[permission.Resource][]pulid.ID,
) (services.RecordLabels, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, refs)
	l.tenants = append(l.tenants, tenant)
	if l.err != nil {
		return nil, l.err
	}

	out := make(services.RecordLabels, len(refs))
	for resource, ids := range refs {
		out[resource] = make(map[pulid.ID]string, len(ids))
		for _, id := range ids {
			if label := l.labels.Label(resource, id); label != "" {
				out[resource][id] = label
			}
		}
	}

	return out, nil
}

func newSubsetLabels(labeler services.RecordLabeler) *SubsetLabelsLoaderFactory {
	return NewSubsetLabelsLoaderFactory(SubsetLabelsLoaderFactoryParams{Labeler: labeler})
}

func manyShipments(n int) ([]pulid.ID, []string) {
	ids := make([]pulid.ID, 0, n)
	texts := make([]string, 0, n)
	for range n {
		id := pulid.MustNew("shp_")
		ids = append(ids, id)
		texts = append(texts, id.String())
	}

	return ids, texts
}

// The records every proposal on a page offers are named in one read, grouped
// by resource, inside the reader's tenant. Each field gets the labels of its
// own records; a record that is gone, or an id that names no record, has
// none.
func TestSubsetLabelsLoader_NamesEveryFieldsRecordsInOneRead(t *testing.T) {
	t.Parallel()

	shipmentA, shipmentB, gone := pulid.MustNew("shp_"), pulid.MustNew("shp_"),
		pulid.MustNew("shp_")
	worker := pulid.MustNew("wrk_")
	labeler := &countingLabeler{labels: services.RecordLabels{
		permission.ResourceShipment: {shipmentA: "PRO-1", shipmentB: "PRO-2"},
		permission.ResourceWorker:   {worker: "Ada Lovelace"},
	}}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	loader := newSubsetLabels(labeler).NewForTenant(tenant)

	first := SubsetLabelsKey(permission.ResourceShipment,
		[]string{shipmentA.String(), gone.String(), "short"})
	second := SubsetLabelsKey(permission.ResourceShipment,
		[]string{shipmentB.String(), shipmentA.String()})
	third := SubsetLabelsKey(permission.ResourceWorker, []string{worker.String()})

	labels, err := loader.LoadAll(t.Context(), []string{first, second, third})
	require.NoError(t, err)

	assert.Equal(t, []services.RecordLabels{
		{permission.ResourceShipment: {shipmentA: "PRO-1"}},
		{permission.ResourceShipment: {shipmentB: "PRO-2", shipmentA: "PRO-1"}},
		{permission.ResourceWorker: {worker: "Ada Lovelace"}},
	}, labels)
	require.Len(t, labeler.calls, 1, "one read for every field in the batch")
	assert.ElementsMatch(t, []pulid.ID{shipmentA, gone, shipmentB},
		labeler.calls[0][permission.ResourceShipment], "each record is asked for once")
	assert.Equal(t, []pulid.ID{worker}, labeler.calls[0][permission.ResourceWorker])
	assert.Equal(t, tenant, labeler.tenants[0])
}

// A transfer of thousands of shipments is labelled in one read; records past
// what one read names are read in the next, never left unlabelled.
func TestSubsetLabelsLoader_ReadsNoMoreThanOneReadNamesAtATime(t *testing.T) {
	t.Parallel()

	ids, texts := manyShipments(services.MaxRecordLabelsPerResource + 3)
	named := make(map[pulid.ID]string, len(ids))
	for _, id := range ids {
		named[id] = "PRO-" + id.String()
	}
	labeler := &countingLabeler{labels: services.RecordLabels{permission.ResourceShipment: named}}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_")}
	loader := newSubsetLabels(labeler).NewForTenant(tenant)

	longest, err := loader.Load(t.Context(), SubsetLabelsKey(permission.ResourceShipment,
		texts[:services.MaxRecordLabelsPerResource]))
	require.NoError(t, err)
	assert.Len(t, longest[permission.ResourceShipment], services.MaxRecordLabelsPerResource)
	require.Len(t, labeler.calls, 1, "the longest subset is one read")

	loader = newSubsetLabels(labeler).NewForTenant(tenant)
	beyond, err := loader.Load(t.Context(), SubsetLabelsKey(permission.ResourceShipment, texts))
	require.NoError(t, err)
	assert.Len(t, beyond[permission.ResourceShipment], len(ids))
	require.Len(t, labeler.calls, 3)
	assert.Len(t, labeler.calls[1][permission.ResourceShipment], services.MaxRecordLabelsPerResource)
	assert.Len(t, labeler.calls[2][permission.ResourceShipment], 3)
}

func TestSubsetLabelsLoader_ReportsAFailedRead(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_")}
	loader := newSubsetLabels(&countingLabeler{err: errors.New("database is down")}).
		NewForTenant(tenant)

	_, err := loader.Load(t.Context(), SubsetLabelsKey(permission.ResourceShipment,
		[]string{pulid.MustNew("shp_").String()}))
	require.ErrorContains(t, err, "database is down")
}

// Without a labeler no record is named, and that is not an error: the
// records are still listed, by their ids.
func TestSubsetLabelsLoader_WithoutALabelerNamesNothing(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_")}
	loader := newSubsetLabels(nil).NewForTenant(tenant)

	labels, err := loader.Load(t.Context(), SubsetLabelsKey(permission.ResourceShipment,
		[]string{pulid.MustNew("shp_").String()}))
	require.NoError(t, err)
	assert.Empty(t, labels)
}
