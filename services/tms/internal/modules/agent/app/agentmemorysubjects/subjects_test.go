package agentmemorysubjects

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLinks struct {
	links map[agentmemory.MemoryRecordKind][]agentmemoryports.MemoryRecordLink
	asked []agentmemoryports.ListMemoryRecordLinksRequest
	fail  agentmemory.MemoryRecordKind
}

func (f *fakeLinks) ListRecordLinks(
	_ context.Context,
	req *agentmemoryports.ListMemoryRecordLinksRequest,
) ([]agentmemoryports.MemoryRecordLink, error) {
	f.asked = append(f.asked, *req)
	if req.Kind == f.fail {
		return nil, errors.New("links unavailable")
	}

	wanted := make(map[pulid.ID]struct{}, len(req.IDs))
	for _, id := range req.IDs {
		wanted[id] = struct{}{}
	}
	out := make([]agentmemoryports.MemoryRecordLink, 0)
	for _, link := range f.links[req.Kind] {
		if _, ok := wanted[link.From]; ok {
			out = append(out, link)
		}
	}

	return out, nil
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func ref(kind, id string) agentpagecontext.EntityRef {
	return agentpagecontext.EntityRef{Type: kind, ID: id}
}

func TestResolve_AMasterRecordIsItsOwnSubject(t *testing.T) {
	t.Parallel()

	customer, location := pulid.MustNew("cus_"), pulid.MustNew("loc_")
	worker, carrier := pulid.MustNew("wrk_"), pulid.MustNew("car_")
	links := &fakeLinks{}

	got, err := NewSubjectResolver(
		links,
	).Resolve(t.Context(), tenant(), []agentpagecontext.EntityRef{
		ref("customer", customer.String()),
		ref("Location", location.String()),
		ref(string(agent.SubjectWorker), worker.String()),
		ref("carrier", carrier.String()),
		ref("customer", customer.String()),
	})
	require.NoError(t, err)

	assert.Equal(t, []agentmemory.MemorySubject{
		{
			Type:     agentmemory.MemorySubjectCustomer,
			ID:       customer,
			Relation: agentmemory.MemoryRelationDirect,
		},
		{
			Type:     agentmemory.MemorySubjectLocation,
			ID:       location,
			Relation: agentmemory.MemoryRelationDirect,
		},
		{
			Type:     agentmemory.MemorySubjectWorker,
			ID:       worker,
			Relation: agentmemory.MemoryRelationDirect,
		},
		{
			Type:     agentmemory.MemorySubjectCarrier,
			ID:       carrier,
			Relation: agentmemory.MemoryRelationDirect,
		},
	}, got)
	assert.Empty(t, links.asked, "a master record names nobody else")
}

func TestResolve_AShipmentNamesItsCustomerStopsWorkersAndCarrier(t *testing.T) {
	t.Parallel()

	shipment := pulid.MustNew("shp_")
	customer := pulid.MustNew("cus_")
	origin, destination := pulid.MustNew("loc_"), pulid.MustNew("loc_")
	driver, carrier := pulid.MustNew("wrk_"), pulid.MustNew("car_")
	links := &fakeLinks{links: map[agentmemory.MemoryRecordKind][]agentmemoryports.MemoryRecordLink{
		agentmemory.MemoryRecordShipment: {
			{From: shipment, Kind: agentmemory.MemoryRecordCustomer, ID: customer},
			{From: shipment, Kind: agentmemory.MemoryRecordLocation, ID: origin},
			{From: shipment, Kind: agentmemory.MemoryRecordLocation, ID: destination},
			{From: shipment, Kind: agentmemory.MemoryRecordWorker, ID: driver},
			{From: shipment, Kind: agentmemory.MemoryRecordCarrier, ID: carrier},
		},
	}}

	got, err := NewSubjectResolver(
		links,
	).Resolve(t.Context(), tenant(), []agentpagecontext.EntityRef{
		ref(string(agent.SubjectShipment), shipment.String()),
	})
	require.NoError(t, err)

	related := func(kind agentmemory.MemorySubjectType, id pulid.ID) agentmemory.MemorySubject {
		return agentmemory.MemorySubject{
			Type:     kind,
			ID:       id,
			Relation: agentmemory.MemoryRelationRelated,
		}
	}
	assert.Equal(t, []agentmemory.MemorySubject{
		related(agentmemory.MemorySubjectCustomer, customer),
		related(agentmemory.MemorySubjectLocation, origin),
		related(agentmemory.MemorySubjectLocation, destination),
		related(agentmemory.MemorySubjectWorker, driver),
		related(agentmemory.MemorySubjectCarrier, carrier),
	}, got)
}

func TestResolve_FollowsADocumentOrMessageThroughTheRecordItNames(t *testing.T) {
	t.Parallel()

	document, message := pulid.MustNew("doc_"), pulid.MustNew("imsg_")
	move, shipment := pulid.MustNew("smv_"), pulid.MustNew("shp_")
	invoice, queued := pulid.MustNew("inv_"), pulid.MustNew("bqi_")
	customer, billTo := pulid.MustNew("cus_"), pulid.MustNew("cus_")
	matchedCarrier, stop := pulid.MustNew("car_"), pulid.MustNew("loc_")
	links := &fakeLinks{links: map[agentmemory.MemoryRecordKind][]agentmemoryports.MemoryRecordLink{
		agentmemory.MemoryRecordDocument: {
			{From: document, Kind: agentmemory.MemoryRecordShipmentMove, ID: move},
		},
		agentmemory.MemoryRecordInboundMessage: {
			{From: message, Kind: agentmemory.MemoryRecordCarrier, ID: matchedCarrier},
			{From: message, Kind: agentmemory.MemoryRecordShipment, ID: shipment},
		},
		agentmemory.MemoryRecordShipmentMove: {
			{From: move, Kind: agentmemory.MemoryRecordShipment, ID: shipment},
		},
		agentmemory.MemoryRecordShipment: {
			{From: shipment, Kind: agentmemory.MemoryRecordCustomer, ID: customer},
			{From: shipment, Kind: agentmemory.MemoryRecordLocation, ID: stop},
		},
		agentmemory.MemoryRecordInvoice: {
			{From: invoice, Kind: agentmemory.MemoryRecordCustomer, ID: customer},
		},
		agentmemory.MemoryRecordBillingQueueItem: {
			{From: queued, Kind: agentmemory.MemoryRecordCustomer, ID: billTo},
		},
	}}

	got, err := NewSubjectResolver(
		links,
	).Resolve(t.Context(), tenant(), []agentpagecontext.EntityRef{
		ref("document", document.String()),
		ref(string(agent.SubjectInboundMessage), message.String()),
		ref("invoice", invoice.String()),
		ref(string(agent.SubjectBillingQueueItem), queued.String()),
	})
	require.NoError(t, err)

	ids := make([]pulid.ID, 0, len(got))
	for _, subject := range got {
		assert.Equal(t, agentmemory.MemoryRelationRelated, subject.Relation)
		ids = append(ids, subject.ID)
	}
	assert.Equal(t, []pulid.ID{matchedCarrier, customer, billTo, stop}, ids,
		"the message's carrier and the invoice's customer come a step before the shipment's")

	shipmentReads := 0
	for _, asked := range links.asked {
		if asked.Kind == agentmemory.MemoryRecordShipment {
			shipmentReads++
			assert.Equal(t, []pulid.ID{shipment}, asked.IDs,
				"a shipment reached two ways is read once")
		}
	}
	assert.Equal(t, 1, shipmentReads)
}

func TestResolve_StopsAtTwelveDirectRecordsFirst(t *testing.T) {
	t.Parallel()

	shipment := pulid.MustNew("shp_")
	stops := make([]agentmemoryports.MemoryRecordLink, 0, 20)
	for range 20 {
		stops = append(stops, agentmemoryports.MemoryRecordLink{
			From: shipment, Kind: agentmemory.MemoryRecordLocation, ID: pulid.MustNew("loc_"),
		})
	}
	links := &fakeLinks{links: map[agentmemory.MemoryRecordKind][]agentmemoryports.MemoryRecordLink{
		agentmemory.MemoryRecordShipment: stops,
	}}

	records := []agentpagecontext.EntityRef{ref("shipment", shipment.String())}
	direct := make([]pulid.ID, 0, 3)
	for range 3 {
		id := pulid.MustNew("cus_")
		direct = append(direct, id)
		records = append(records, ref("customer", id.String()))
	}

	got, err := NewSubjectResolver(links).Resolve(t.Context(), tenant(), records)
	require.NoError(t, err)

	require.Len(t, got, agentmemory.MaxMemorySubjectsPerTurn)
	for idx, id := range direct {
		assert.Equal(t, id, got[idx].ID)
		assert.Equal(t, agentmemory.MemoryRelationDirect, got[idx].Relation,
			"what the person is looking at comes before what it names")
	}
	assert.Equal(t, agentmemory.MemoryRelationRelated, got[len(got)-1].Relation)
}

func TestResolve_IgnoresUnknownKindsAndBadIDsAndKeepsWhatItReadOnFailure(t *testing.T) {
	t.Parallel()

	customer, shipment := pulid.MustNew("cus_"), pulid.MustNew("shp_")
	links := &fakeLinks{fail: agentmemory.MemoryRecordShipment}

	got, err := NewSubjectResolver(
		links,
	).Resolve(t.Context(), tenant(), []agentpagecontext.EntityRef{
		ref("tractor", pulid.MustNew("tr_").String()),
		ref("customer", "not-an-id"),
		ref("customer", customer.String()),
		ref("shipment", shipment.String()),
	})
	require.Error(t, err)
	assert.Equal(t, []agentmemory.MemorySubject{
		{
			Type:     agentmemory.MemorySubjectCustomer,
			ID:       customer,
			Relation: agentmemory.MemoryRelationDirect,
		},
	}, got)
}

func TestResolve_WithoutALinkReaderReadsOnlyDirectRecords(t *testing.T) {
	t.Parallel()

	customer := pulid.MustNew("cus_")
	got, err := NewSubjectResolver(nil).Resolve(t.Context(), tenant(), []agentpagecontext.EntityRef{
		ref("customer", customer.String()),
		ref("shipment", pulid.MustNew("shp_").String()),
	})
	require.NoError(t, err)
	assert.Len(t, got, 1)
}
