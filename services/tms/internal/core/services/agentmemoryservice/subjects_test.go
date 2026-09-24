package agentmemoryservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLinks struct {
	links map[agent.MemoryRecordKind][]repositories.MemoryRecordLink
	asked []repositories.ListMemoryRecordLinksRequest
	fail  agent.MemoryRecordKind
}

func (f *fakeLinks) ListRecordLinks(
	_ context.Context,
	req repositories.ListMemoryRecordLinksRequest,
) ([]repositories.MemoryRecordLink, error) {
	f.asked = append(f.asked, req)
	if req.Kind == f.fail {
		return nil, errors.New("links unavailable")
	}

	wanted := make(map[pulid.ID]struct{}, len(req.IDs))
	for _, id := range req.IDs {
		wanted[id] = struct{}{}
	}
	out := make([]repositories.MemoryRecordLink, 0)
	for _, link := range f.links[req.Kind] {
		if _, ok := wanted[link.From]; ok {
			out = append(out, link)
		}
	}

	return out, nil
}

func ref(kind, id string) agent.EntityRef { return agent.EntityRef{Type: kind, ID: id} }

func TestResolve_AMasterRecordIsItsOwnSubject(t *testing.T) {
	t.Parallel()

	customer, location := pulid.MustNew("cus_"), pulid.MustNew("loc_")
	worker, carrier := pulid.MustNew("wrk_"), pulid.MustNew("car_")
	links := &fakeLinks{}

	got, err := NewSubjectResolver(links).Resolve(t.Context(), tenant(), []agent.EntityRef{
		ref("customer", customer.String()),
		ref("Location", location.String()),
		ref(string(agent.SubjectWorker), worker.String()),
		ref("carrier", carrier.String()),
		ref("customer", customer.String()),
	})
	require.NoError(t, err)

	assert.Equal(t, []agent.MemorySubject{
		{Type: agent.MemorySubjectCustomer, ID: customer, Relation: agent.MemoryRelationDirect},
		{Type: agent.MemorySubjectLocation, ID: location, Relation: agent.MemoryRelationDirect},
		{Type: agent.MemorySubjectWorker, ID: worker, Relation: agent.MemoryRelationDirect},
		{Type: agent.MemorySubjectCarrier, ID: carrier, Relation: agent.MemoryRelationDirect},
	}, got)
	assert.Empty(t, links.asked, "a master record names nobody else")
}

func TestResolve_AShipmentNamesItsCustomerStopsWorkersAndCarrier(t *testing.T) {
	t.Parallel()

	shipment := pulid.MustNew("shp_")
	customer := pulid.MustNew("cus_")
	origin, destination := pulid.MustNew("loc_"), pulid.MustNew("loc_")
	driver, carrier := pulid.MustNew("wrk_"), pulid.MustNew("car_")
	links := &fakeLinks{links: map[agent.MemoryRecordKind][]repositories.MemoryRecordLink{
		agent.MemoryRecordShipment: {
			{From: shipment, Kind: agent.MemoryRecordCustomer, ID: customer},
			{From: shipment, Kind: agent.MemoryRecordLocation, ID: origin},
			{From: shipment, Kind: agent.MemoryRecordLocation, ID: destination},
			{From: shipment, Kind: agent.MemoryRecordWorker, ID: driver},
			{From: shipment, Kind: agent.MemoryRecordCarrier, ID: carrier},
		},
	}}

	got, err := NewSubjectResolver(links).Resolve(t.Context(), tenant(), []agent.EntityRef{
		ref(string(agent.SubjectShipment), shipment.String()),
	})
	require.NoError(t, err)

	related := func(kind agent.MemorySubjectType, id pulid.ID) agent.MemorySubject {
		return agent.MemorySubject{Type: kind, ID: id, Relation: agent.MemoryRelationRelated}
	}
	assert.Equal(t, []agent.MemorySubject{
		related(agent.MemorySubjectCustomer, customer),
		related(agent.MemorySubjectLocation, origin),
		related(agent.MemorySubjectLocation, destination),
		related(agent.MemorySubjectWorker, driver),
		related(agent.MemorySubjectCarrier, carrier),
	}, got)
}

func TestResolve_FollowsADocumentOrMessageThroughTheRecordItNames(t *testing.T) {
	t.Parallel()

	document, message := pulid.MustNew("doc_"), pulid.MustNew("imsg_")
	move, shipment := pulid.MustNew("smv_"), pulid.MustNew("shp_")
	invoice, queued := pulid.MustNew("inv_"), pulid.MustNew("bqi_")
	customer, billTo := pulid.MustNew("cus_"), pulid.MustNew("cus_")
	matchedCarrier, stop := pulid.MustNew("car_"), pulid.MustNew("loc_")
	links := &fakeLinks{links: map[agent.MemoryRecordKind][]repositories.MemoryRecordLink{
		agent.MemoryRecordDocument: {
			{From: document, Kind: agent.MemoryRecordShipmentMove, ID: move},
		},
		agent.MemoryRecordInboundMessage: {
			{From: message, Kind: agent.MemoryRecordCarrier, ID: matchedCarrier},
			{From: message, Kind: agent.MemoryRecordShipment, ID: shipment},
		},
		agent.MemoryRecordShipmentMove: {
			{From: move, Kind: agent.MemoryRecordShipment, ID: shipment},
		},
		agent.MemoryRecordShipment: {
			{From: shipment, Kind: agent.MemoryRecordCustomer, ID: customer},
			{From: shipment, Kind: agent.MemoryRecordLocation, ID: stop},
		},
		agent.MemoryRecordInvoice: {
			{From: invoice, Kind: agent.MemoryRecordCustomer, ID: customer},
		},
		agent.MemoryRecordBillingQueueItem: {
			{From: queued, Kind: agent.MemoryRecordCustomer, ID: billTo},
		},
	}}

	got, err := NewSubjectResolver(links).Resolve(t.Context(), tenant(), []agent.EntityRef{
		ref("document", document.String()),
		ref(string(agent.SubjectInboundMessage), message.String()),
		ref("invoice", invoice.String()),
		ref(string(agent.SubjectBillingQueueItem), queued.String()),
	})
	require.NoError(t, err)

	ids := make([]pulid.ID, 0, len(got))
	for _, subject := range got {
		assert.Equal(t, agent.MemoryRelationRelated, subject.Relation)
		ids = append(ids, subject.ID)
	}
	assert.Equal(t, []pulid.ID{matchedCarrier, customer, billTo, stop}, ids,
		"the message's carrier and the invoice's customer come a step before the shipment's")

	shipmentReads := 0
	for _, asked := range links.asked {
		if asked.Kind == agent.MemoryRecordShipment {
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
	stops := make([]repositories.MemoryRecordLink, 0, 20)
	for range 20 {
		stops = append(stops, repositories.MemoryRecordLink{
			From: shipment, Kind: agent.MemoryRecordLocation, ID: pulid.MustNew("loc_"),
		})
	}
	links := &fakeLinks{links: map[agent.MemoryRecordKind][]repositories.MemoryRecordLink{
		agent.MemoryRecordShipment: stops,
	}}

	records := []agent.EntityRef{ref("shipment", shipment.String())}
	direct := make([]pulid.ID, 0, 3)
	for range 3 {
		id := pulid.MustNew("cus_")
		direct = append(direct, id)
		records = append(records, ref("customer", id.String()))
	}

	got, err := NewSubjectResolver(links).Resolve(t.Context(), tenant(), records)
	require.NoError(t, err)

	require.Len(t, got, agent.MaxMemorySubjectsPerTurn)
	for idx, id := range direct {
		assert.Equal(t, id, got[idx].ID)
		assert.Equal(t, agent.MemoryRelationDirect, got[idx].Relation,
			"what the person is looking at comes before what it names")
	}
	assert.Equal(t, agent.MemoryRelationRelated, got[len(got)-1].Relation)
}

func TestResolve_IgnoresUnknownKindsAndBadIDsAndKeepsWhatItReadOnFailure(t *testing.T) {
	t.Parallel()

	customer, shipment := pulid.MustNew("cus_"), pulid.MustNew("shp_")
	links := &fakeLinks{fail: agent.MemoryRecordShipment}

	got, err := NewSubjectResolver(links).Resolve(t.Context(), tenant(), []agent.EntityRef{
		ref("tractor", pulid.MustNew("tr_").String()),
		ref("customer", "not-an-id"),
		ref("customer", customer.String()),
		ref("shipment", shipment.String()),
	})
	require.Error(t, err)
	assert.Equal(t, []agent.MemorySubject{
		{Type: agent.MemorySubjectCustomer, ID: customer, Relation: agent.MemoryRelationDirect},
	}, got)
}

func TestResolve_WithoutALinkReaderReadsOnlyDirectRecords(t *testing.T) {
	t.Parallel()

	customer := pulid.MustNew("cus_")
	got, err := NewSubjectResolver(nil).Resolve(t.Context(), tenant(), []agent.EntityRef{
		ref("customer", customer.String()),
		ref("shipment", pulid.MustNew("shp_").String()),
	})
	require.NoError(t, err)
	assert.Len(t, got, 1)
}
