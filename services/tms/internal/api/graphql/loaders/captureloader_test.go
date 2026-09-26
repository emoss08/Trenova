package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCaptureRecordLabeler struct {
	asked  map[string][]pulid.ID
	labels map[string][]*repositories.CaptureRecordLabel
	fail   map[string]error
}

func (s *stubCaptureRecordLabeler) Labels(
	_ context.Context,
	req *repositories.ListCaptureRecordLabelsRequest,
) ([]*repositories.CaptureRecordLabel, error) {
	s.asked[req.ResourceType] = req.IDs
	if err := s.fail[req.ResourceType]; err != nil {
		return nil, err
	}

	return s.labels[req.ResourceType], nil
}

func TestCaptureRecordLabelsAskOncePerKindAndLeaveAGoneRecordAbsent(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	shipment, goneShipment := pulid.MustNew("shp_"), pulid.MustNew("shp_")
	worker := pulid.MustNew("wrk_")
	repo := &stubCaptureRecordLabeler{
		asked: map[string][]pulid.ID{},
		labels: map[string][]*repositories.CaptureRecordLabel{
			"shipment": {{ResourceType: "shipment", ID: shipment, Title: "PRO-4471"}},
			"worker":   {{ResourceType: "worker", ID: worker, Title: "Dana Reyes"}},
		},
	}
	factory := &CaptureRecordLabelLoaderFactory{records: repo}

	values, errs := factory.batchFunc(tenant)(t.Context(), []string{
		CaptureRecordKey("shipment", shipment),
		CaptureRecordKey("worker", worker),
		CaptureRecordKey("shipment", goneShipment),
		CaptureRecordKey("shipment", shipment),
		"malformed",
	})

	assert.ElementsMatch(t, []pulid.ID{shipment, goneShipment}, repo.asked["shipment"],
		"one query for every shipment on the page, each asked about once")
	assert.Equal(t, []pulid.ID{worker}, repo.asked["worker"])

	require.NoError(t, errs[0])
	assert.Equal(t, "PRO-4471", values[0].Title)
	assert.Equal(t, "Dana Reyes", values[1].Title)
	require.NoError(t, errs[2])
	assert.Nil(t, values[2], "a record that is gone loads as nothing, not as a failure")
	assert.Same(t, values[0], values[3])
	assert.Error(t, errs[4])
}

func TestCaptureRecordLabelFailureStaysWithItsKind(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	tractor, customer := pulid.MustNew("trac_"), pulid.MustNew("cus_")
	boom := errors.New("connection reset")
	repo := &stubCaptureRecordLabeler{
		asked: map[string][]pulid.ID{},
		labels: map[string][]*repositories.CaptureRecordLabel{
			"customer": {{ResourceType: "customer", ID: customer, Title: "Acme Foods"}},
		},
		fail: map[string]error{"tractor": boom},
	}
	factory := &CaptureRecordLabelLoaderFactory{records: repo}

	values, errs := factory.batchFunc(tenant)(t.Context(), []string{
		CaptureRecordKey("tractor", tractor),
		CaptureRecordKey("customer", customer),
	})

	require.ErrorIs(t, errs[0], boom)
	require.NoError(t, errs[1])
	assert.Equal(t, "Acme Foods", values[1].Title)
}
