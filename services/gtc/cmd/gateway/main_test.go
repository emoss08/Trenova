package main

import (
	"testing"

	"github.com/emoss08/gtc/internal/core/domain"
)

func TestProjectionTablesDeduplicates(t *testing.T) {
	t.Parallel()

	tables := projectionTables([]domain.Projection{
		{Name: "a", SourceSchema: "public", SourceTable: "shipments"},
		{Name: "b", SourceSchema: "public", SourceTable: "shipments"},
		{Name: "c", SourceSchema: "public", SourceTable: "customers"},
	})

	if len(tables) != 2 {
		t.Fatalf("expected 2 unique tables, got %v", tables)
	}
	if tables[0] != "public.shipments" || tables[1] != "public.customers" {
		t.Fatalf("unexpected tables: %v", tables)
	}
}

func TestCSVList(t *testing.T) {
	t.Parallel()

	items := csvList("shipment-search, public.shipments ,,customer-cache")
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %v", items)
	}
}
