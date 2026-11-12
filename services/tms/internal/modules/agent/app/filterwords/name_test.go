package filterwords

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	namedCustomer   = "cus_01JW3K18A8TPC7T7SPX7745V1H"
	unnamedCustomer = "cus_01JSV4G0Q80KNYKQ254QWT5T0V"
)

type stubLabeler struct {
	labels agentruntimeports.RecordLabels
	err    error
	asked  map[permission.Resource][]pulid.ID
}

func (s *stubLabeler) Labels(
	_ context.Context,
	_ pagination.TenantInfo,
	refs map[permission.Resource][]pulid.ID,
) (agentruntimeports.RecordLabels, error) {
	s.asked = refs

	return s.labels, s.err
}

func customerFilters() []Filter {
	return Of(
		[]string{"customerId one of " + namedCustomer + ", " + unnamedCustomer},
		map[string]any{"customerId": []any{namedCustomer, unnamedCustomer}},
	)
}

func TestNameRecords_RecordsAreReadOnceAndNamed(t *testing.T) {
	t.Parallel()

	labeler := &stubLabeler{labels: agentruntimeports.RecordLabels{
		permission.ResourceCustomer: {pulid.ID(namedCustomer): "Granite Textiles"},
	}}
	filters := customerFilters()

	require.NoError(t, NameRecords(t.Context(), labeler, pagination.TenantInfo{}, filters))

	assert.Equal(t, map[permission.Resource][]pulid.ID{
		permission.ResourceCustomer: {pulid.ID(namedCustomer), pulid.ID(unnamedCustomer)},
	}, labeler.asked)
	assert.Equal(t, "Customer Granite Textiles and 1 more", Line(filters))
}

func TestNameRecords_AFailedReadStillShowsNoID(t *testing.T) {
	t.Parallel()

	filters := customerFilters()

	err := NameRecords(
		t.Context(),
		&stubLabeler{err: errors.New("read failed")},
		pagination.TenantInfo{},
		filters,
	)

	require.Error(t, err)
	assert.Equal(t, "Customer 2 selected", Line(filters))
}

func TestNameRecords_FiltersWithoutRecordsReadNothing(t *testing.T) {
	t.Parallel()

	labeler := &stubLabeler{}
	filters := Of([]string{"status equals Delayed"}, map[string]any{"status": "Delayed"})

	require.NoError(t, NameRecords(t.Context(), labeler, pagination.TenantInfo{}, filters))

	assert.Nil(t, labeler.asked)
	assert.Equal(t, "Status Delayed", Line(filters))
}
