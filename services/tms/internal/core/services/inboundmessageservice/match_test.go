package inboundmessageservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubShipments struct {
	byReference map[string]pulid.ID
	err         error
	asked       []string
}

func (s *stubShipments) FindByReference(
	_ context.Context, _ pagination.TenantInfo, reference string,
) (pulid.ID, bool, error) {
	s.asked = append(s.asked, reference)
	if s.err != nil {
		return pulid.Nil, false, s.err
	}
	id, ok := s.byReference[reference]

	return id, ok, nil
}

type stubParties struct {
	customers map[string]pulid.ID
	carriers  map[string]pulid.ID
	err       error
}

func (s *stubParties) FindCustomerByEmail(
	_ context.Context, _ pagination.TenantInfo, address string,
) (pulid.ID, bool, error) {
	if s.err != nil {
		return pulid.Nil, false, s.err
	}
	id, ok := s.customers[address]

	return id, ok, nil
}

func (s *stubParties) FindCarrierByEmail(
	_ context.Context, _ pagination.TenantInfo, address string,
) (pulid.ID, bool, error) {
	if s.err != nil {
		return pulid.Nil, false, s.err
	}
	id, ok := s.carriers[address]

	return id, ok, nil
}

func matcher(shipments *stubShipments, parties *stubParties) *Service {
	svc := &Service{l: zap.NewNop()}
	if shipments != nil {
		svc.shipments = shipments
	}
	if parties != nil {
		svc.parties = parties
	}

	return svc
}

func message(subject, body, from string) *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ID:          pulid.MustNew("imsg_"),
		FromAddress: from,
		Subject:     subject,
		TextBody:    body,
	}
}

// A reference has a digit and is long enough not to be a quantity. Without
// that, every message matches "48" and "2" against whatever shipment happens
// to be numbered that way.
func TestReferenceCandidates_KeepsThingsThatCouldNameARecord(t *testing.T) {
	t.Parallel()

	got := referenceCandidates(
		"Load 88213 tender",
		"Please confirm pickup Thursday for 2 pallets within 48 hours. BOL-2026-0001.",
	)

	assert.Contains(t, got, "88213")
	assert.Contains(t, got, "BOL-2026-0001")
	assert.NotContains(t, got, "Thursday", "a word is not a reference")
	assert.NotContains(t, got, "2", "a pallet count is not a reference")
	assert.NotContains(t, got, "48", "an hour count is not a reference")
	assert.NotContains(t, got, "confirm")
}

// The subject is where people put the number they are writing about, so it is
// looked up before anything buried in a quoted reply.
func TestReferenceCandidates_TakesTheSubjectFirst(t *testing.T) {
	t.Parallel()

	got := referenceCandidates("Re: SHP-0002", "On Monday you wrote about SHP-0001")

	require.NotEmpty(t, got)
	assert.Equal(t, "SHP-0002", got[0])
}

// A quoted thread repeats the same number a dozen times; looking it up a dozen
// times would be a dozen queries for one answer.
func TestReferenceCandidates_AsksForEachReferenceOnce(t *testing.T) {
	t.Parallel()

	got := referenceCandidates("SHP-0001", "SHP-0001 shp-0001 SHP-0001")

	assert.Len(t, got, 1)
}

// A long quoted thread is full of things that look like references. Checking
// all of them turns one message into a hundred queries for no better answer.
func TestReferenceCandidates_IsBounded(t *testing.T) {
	t.Parallel()

	body := ""
	for i := range 100 {
		body += " REF-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10)) + "00"
	}

	assert.LessOrEqual(t, len(referenceCandidates("", body)), maxReferenceCandidates)
}

func TestMatch_FindsTheShipmentAndSaysWhy(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	shipments := &stubShipments{byReference: map[string]pulid.ID{"88213": shipmentID}}
	svc := matcher(shipments, nil)

	got := svc.Match(t.Context(), message("Load 88213", "", ""), pagination.TenantInfo{})

	assert.Equal(t, shipmentID, got.ShipmentID)
	assert.Contains(t, got.Reason, "88213",
		"a match nobody can check is a match nobody will trust")
}

func TestMatch_FindsTheSenderAndSaysWhy(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	parties := &stubParties{customers: map[string]pulid.ID{"ops@acme.com": customerID}}
	svc := matcher(nil, parties)

	got := svc.Match(t.Context(), message("", "", "ops@acme.com"), pagination.TenantInfo{})

	assert.Equal(t, customerID, got.CustomerID)
	assert.Contains(t, got.Reason, "ops@acme.com")
	assert.Contains(t, got.Reason, "customer")
}

// A sender who is both is treated as a customer, because that is the
// relationship the mail is more likely about; the reason says which was used
// either way.
func TestMatch_PrefersACustomerOverACarrier(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	carrierID := pulid.MustNew("car_")
	parties := &stubParties{
		customers: map[string]pulid.ID{"ops@both.com": customerID},
		carriers:  map[string]pulid.ID{"ops@both.com": carrierID},
	}

	got := matcher(nil, parties).Match(
		t.Context(), message("", "", "ops@both.com"), pagination.TenantInfo{},
	)

	assert.Equal(t, customerID, got.CustomerID)
	assert.True(t, got.CarrierID.IsNil())
}

// Both claims appear in one reason, so a person reading it can weigh them
// separately.
func TestMatch_ReportsBothClaims(t *testing.T) {
	t.Parallel()

	shipments := &stubShipments{byReference: map[string]pulid.ID{"88213": pulid.MustNew("shp_")}}
	parties := &stubParties{customers: map[string]pulid.ID{"ops@acme.com": pulid.MustNew("cus_")}}

	got := matcher(shipments, parties).Match(
		t.Context(), message("Load 88213", "", "ops@acme.com"), pagination.TenantInfo{},
	)

	assert.Contains(t, got.Reason, "ops@acme.com")
	assert.Contains(t, got.Reason, "88213")
}

// An unmatched message is still a message a person can read and link by hand.
func TestMatch_SaysNothingWhenItMatchesNothing(t *testing.T) {
	t.Parallel()

	got := matcher(&stubShipments{}, &stubParties{}).Match(
		t.Context(), message("Hello", "Just checking in", "nobody@nowhere.com"),
		pagination.TenantInfo{},
	)

	assert.True(t, got.ShipmentID.IsNil())
	assert.Empty(t, got.Reason, "no match must not read as an unexplained one")
}

// Losing a lookup must not lose the message. Refusing to store it because one
// query failed would cost the thing the sender believes arrived.
func TestMatch_SurvivesAFailedLookup(t *testing.T) {
	t.Parallel()

	svc := matcher(
		&stubShipments{err: errors.New("database unreachable")},
		&stubParties{err: errors.New("database unreachable")},
	)

	got := svc.Match(t.Context(), message("Load 88213", "", "ops@acme.com"), pagination.TenantInfo{})

	assert.True(t, got.ShipmentID.IsNil())
	assert.Empty(t, got.Reason)
}

// With no finders wired the matcher is a no-op rather than a panic: an
// installation that has not configured them still receives mail.
func TestMatch_WithoutFindersIsQuiet(t *testing.T) {
	t.Parallel()

	got := matcher(nil, nil).Match(
		t.Context(), message("Load 88213", "", "ops@acme.com"), pagination.TenantInfo{},
	)

	assert.Empty(t, got.Reason)
}

// The first candidate that resolves wins, and the ones after it are never
// asked for — a message naming two shipments is one a person should read.
func TestMatch_StopsAtTheFirstShipmentThatResolves(t *testing.T) {
	t.Parallel()

	shipments := &stubShipments{byReference: map[string]pulid.ID{
		"SHP-0001": pulid.MustNew("shp_"),
		"SHP-0002": pulid.MustNew("shp_"),
	}}

	matcher(shipments, nil).Match(
		t.Context(), message("SHP-0001 and SHP-0002", "", ""), pagination.TenantInfo{},
	)

	assert.Equal(t, []string{"SHP-0001"}, shipments.asked)
}
