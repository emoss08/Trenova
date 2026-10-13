package customertracking

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const now = int64(1_800_000_000)

func at(offset int64) *int64 {
	v := now + offset
	return &v
}

func notifyingProfile() *Profile {
	enabled := now - 24*3600
	return &Profile{
		NotifyPickedUp:         true,
		NotifyDelayed:          true,
		NotifyDelivered:        true,
		NotifyPodReady:         true,
		NotificationsEnabledAt: &enabled,
	}
}

func events(occurrences []Occurrence) []Event {
	result := make([]Event, 0, len(occurrences))
	for _, o := range occurrences {
		result = append(result, o.Event)
	}
	return result
}

func TestDetect_PickedUpIsTheEarliestPickupDeparture(t *testing.T) {
	facts := &ShipmentFacts{Stops: []StopFacts{
		{ID: pulid.MustNew("stp_"), Pickup: true, ActualDeparture: at(-3000)},
		{ID: pulid.MustNew("stp_"), Pickup: true, ActualDeparture: at(-6000)},
		{ID: pulid.MustNew("stp_"), Delivery: true},
	}}

	got := Detect(facts, notifyingProfile(), now)

	require.Len(t, got, 1)
	assert.Equal(t, EventPickedUp, got[0].Event)
	assert.Equal(t, now-6000, got[0].OccurredAt)
	assert.Empty(t, got[0].SubjectKey)
}

func TestDetect_DeliveredIsTheDepartureFromTheLastDeliveryStop(t *testing.T) {
	facts := &ShipmentFacts{Stops: []StopFacts{
		{Pickup: true, ActualDeparture: at(-9000)},
		{Delivery: true, ActualArrival: at(-7000), ActualDeparture: at(-6500)},
		{Delivery: true, ActualArrival: at(-2000)},
	}}

	assert.Equal(t, []Event{EventPickedUp}, events(Detect(facts, notifyingProfile(), now)))

	facts.Stops[2].ActualDeparture = at(-1000)
	got := Detect(facts, notifyingProfile(), now)
	assert.Equal(t, []Event{EventPickedUp, EventDelivered}, events(got))
	assert.Equal(t, now-1000, got[1].OccurredAt)
}

func TestDetect_ACompletedShipmentWithoutADepartureIsDeliveredAtArrival(t *testing.T) {
	facts := &ShipmentFacts{
		Completed: true,
		Stops:     []StopFacts{{Delivery: true, ActualArrival: at(-500)}},
	}

	got := Detect(facts, notifyingProfile(), now)

	require.Len(t, got, 1)
	assert.Equal(t, EventDelivered, got[0].Event)
	assert.Equal(t, now-500, got[0].OccurredAt)
}

func TestDetect_DelayedIsOncePerStopThatNeedsAttentionAndHasNotBeenReached(t *testing.T) {
	late, onTime, reached := pulid.MustNew("stp_"), pulid.MustNew("stp_"), pulid.MustNew("stp_")
	facts := &ShipmentFacts{Etas: []EtaFacts{
		{StopID: late, NeedsAttention: true, EstimatedArrival: at(7200), ComputedAt: now - 60},
		{StopID: onTime, ComputedAt: now - 60},
		{StopID: reached, NeedsAttention: true, Arrived: true, ComputedAt: now - 60},
	}}

	got := Detect(facts, notifyingProfile(), now)

	require.Len(t, got, 1)
	assert.Equal(t, EventDelayed, got[0].Event)
	assert.Equal(t, late.String(), got[0].SubjectKey)
	assert.Equal(t, late, got[0].StopID)
	assert.Equal(t, at(7200), got[0].EstimatedAt)
}

func TestDetect_NoDelayOnceTheShipmentIsDelivered(t *testing.T) {
	facts := &ShipmentFacts{
		Stops: []StopFacts{{Delivery: true, ActualDeparture: at(-100)}},
		Etas: []EtaFacts{
			{StopID: pulid.MustNew("stp_"), NeedsAttention: true, ComputedAt: now - 60},
		},
	}

	assert.Equal(t, []Event{EventDelivered}, events(Detect(facts, notifyingProfile(), now)))
}

func TestDetect_PodReadyIsTheFilingOfAPod(t *testing.T) {
	facts := &ShipmentFacts{Pod: &PodFacts{DocumentID: pulid.MustNew("doc_"), FiledAt: now - 30}}

	got := Detect(facts, notifyingProfile(), now)

	require.Len(t, got, 1)
	assert.Equal(t, EventPodReady, got[0].Event)
	assert.Equal(t, now-30, got[0].OccurredAt)
}

func TestDetect_OnlyEventsARuleAsksFor(t *testing.T) {
	profile := notifyingProfile()
	profile.NotifyPickedUp = false
	profile.NotifyPodReady = false
	facts := &ShipmentFacts{
		Stops: []StopFacts{
			{Pickup: true, ActualDeparture: at(-900)},
			{Delivery: true, ActualDeparture: at(-100)},
		},
		Pod: &PodFacts{DocumentID: pulid.MustNew("doc_"), FiledAt: now - 30},
	}

	assert.Equal(t, []Event{EventDelivered}, events(Detect(facts, profile, now)))
}

func TestDetect_NeverReachesBackBeforeTheRuleWasSwitchedOn(t *testing.T) {
	profile := notifyingProfile()
	enabled := now - 600
	profile.NotificationsEnabledAt = &enabled
	facts := &ShipmentFacts{Stops: []StopFacts{
		{Pickup: true, ActualDeparture: at(-3600)},
		{Delivery: true, ActualDeparture: at(-60)},
	}}

	assert.Equal(t, []Event{EventDelivered}, events(Detect(facts, profile, now)))
}

func TestDetect_NeverReachesBackPastTheLookback(t *testing.T) {
	profile := notifyingProfile()
	enabled := now - 30*24*3600
	profile.NotificationsEnabledAt = &enabled
	facts := &ShipmentFacts{Stops: []StopFacts{
		{Pickup: true, ActualDeparture: at(-NotificationLookbackHours*3600 - 1)},
	}}

	assert.Empty(t, Detect(facts, profile, now))
}

func TestDetect_IgnoresFutureTimes(t *testing.T) {
	facts := &ShipmentFacts{Stops: []StopFacts{{Pickup: true, ActualDeparture: at(600)}}}

	assert.Empty(t, Detect(facts, notifyingProfile(), now))
}

func TestDetect_NothingForACanceledShipmentOrWithoutRules(t *testing.T) {
	facts := &ShipmentFacts{Stops: []StopFacts{{Pickup: true, ActualDeparture: at(-60)}}}

	assert.Empty(t, Detect(&ShipmentFacts{Canceled: true, Stops: facts.Stops},
		notifyingProfile(), now))
	assert.Empty(t, Detect(facts, &Profile{}, now))
	assert.Empty(t, Detect(facts, nil, now))
}
