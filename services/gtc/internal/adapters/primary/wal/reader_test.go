package wal

import (
	"testing"

	"go.uber.org/zap"
)

func TestPublicationTableList(t *testing.T) {
	t.Parallel()

	reader := &Reader{config: Config{
		PublicationTables: []string{"public.shipments", "public.customers"},
	}}

	value, err := reader.publicationTableList()
	if err != nil {
		t.Fatalf("publicationTableList returned error: %v", err)
	}

	if value != `"public"."shipments", "public"."customers"` {
		t.Fatalf("unexpected publication table list: %s", value)
	}
}

func TestAdvanceLSN(t *testing.T) {
	t.Parallel()

	reader := &Reader{}
	if err := reader.AdvanceLSN("0/20"); err != nil {
		t.Fatalf("AdvanceLSN returned error: %v", err)
	}

	if got := reader.CurrentLSN(); got != "0/20" {
		t.Fatalf("expected current lsn 0/20, got %s", got)
	}
}

func TestAdvanceLSNDoesNotMoveBackward(t *testing.T) {
	t.Parallel()

	reader := &Reader{}
	if err := reader.AdvanceLSN("0/20"); err != nil {
		t.Fatalf("AdvanceLSN returned error: %v", err)
	}
	if err := reader.AdvanceLSN("0/10"); err != nil {
		t.Fatalf("AdvanceLSN returned error: %v", err)
	}

	if got := reader.CurrentLSN(); got != "0/20" {
		t.Fatalf("expected current lsn 0/20, got %s", got)
	}
}

func TestAdvanceLSNRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	reader := &Reader{}
	if err := reader.AdvanceLSN("invalid"); err == nil {
		t.Fatalf("expected invalid lsn error")
	}
}

func TestObserveSlotStateStartupFailAction(t *testing.T) {
	t.Parallel()

	reader := &Reader{
		config: Config{
			SlotName:           "gtc_slot",
			InactiveSlotAction: "fail",
			MaxLagBytes:        100,
		},
		logger: zap.NewNop(),
	}

	err := reader.observeSlotState(slotState{Exists: true, Active: false, LagBytes: 10}, true)
	if err == nil {
		t.Fatalf("expected startup failure when slot is inactive")
	}

	statuses := reader.HealthStatuses()
	if statuses["replication_slot"] {
		t.Fatalf("expected replication_slot health to be false")
	}
}

func TestObserveSlotStateStartupWarnAction(t *testing.T) {
	t.Parallel()

	reader := &Reader{
		config: Config{
			SlotName:           "gtc_slot",
			InactiveSlotAction: "warn",
			MaxLagBytes:        100,
		},
		logger: zap.NewNop(),
	}

	if err := reader.observeSlotState(slotState{Exists: true, Active: false, LagBytes: 10}, true); err != nil {
		t.Fatalf("expected warn action to avoid startup failure, got %v", err)
	}

	statuses := reader.HealthStatuses()
	if statuses["replication_slot"] {
		t.Fatalf("expected replication_slot health to remain false when slot is inactive")
	}
}

func TestObserveSlotStateLagThreshold(t *testing.T) {
	t.Parallel()

	reader := &Reader{
		config: Config{
			SlotName:           "gtc_slot",
			InactiveSlotAction: "fail",
			MaxLagBytes:        100,
		},
		logger: zap.NewNop(),
	}

	if err := reader.observeSlotState(slotState{Exists: true, Active: true, LagBytes: 101}, false); err != nil {
		t.Fatalf("expected lag threshold breach to update health without returning error, got %v", err)
	}

	statuses := reader.HealthStatuses()
	if statuses["replication_slot"] {
		t.Fatalf("expected replication_slot health to be false when lag exceeds threshold")
	}
}
