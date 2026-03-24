package meilisearch

import (
	"testing"

	"github.com/emoss08/gtc/internal/core/domain"
)

func TestDocumentKeyPrefersIDField(t *testing.T) {
	t.Parallel()

	record := domain.SourceRecord{
		Schema: "public",
		Table:  "shipments",
		NewData: map[string]any{
			"id":               "shp_123",
			"organization_id":  "org_1",
			"business_unit_id": "bu_1",
		},
	}

	field, key, err := documentKey(record, []string{"id", "organization_id", "business_unit_id"})
	if err != nil {
		t.Fatalf("documentKey returned error: %v", err)
	}

	if field != "id" {
		t.Fatalf("expected id field, got %q", field)
	}

	if key != "shp_123" {
		t.Fatalf("expected id key, got %q", key)
	}
}

func TestDocumentKeyFallsBackToCompositeKey(t *testing.T) {
	t.Parallel()

	record := domain.SourceRecord{
		Schema: "public",
		Table:  "example",
		NewData: map[string]any{
			"organization_id":  "org_1",
			"business_unit_id": "bu_1",
		},
	}

	field, key, err := documentKey(record, []string{"organization_id", "business_unit_id"})
	if err != nil {
		t.Fatalf("documentKey returned error: %v", err)
	}

	if field != "_pk" {
		t.Fatalf("expected _pk field, got %q", field)
	}

	if key != "org_1|bu_1" {
		t.Fatalf("expected composite key, got %q", key)
	}
}
