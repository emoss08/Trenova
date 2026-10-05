package meilisearch

import (
	"encoding/json"
	"testing"

	meili "github.com/meilisearch/meilisearch-go"
)

func TestDecodeHitReturnsPlainValues(t *testing.T) {
	hit := meili.Hit{
		"id":         json.RawMessage(`"shp_1"`),
		"pro_number": json.RawMessage(`"S26100004036460"`),
		"weight":     json.RawMessage(`12.5`),
		"tags":       json.RawMessage(`["a","b"]`),
		"deleted_at": json.RawMessage(`null`),
	}

	document, err := decodeHit(hit)
	if err != nil {
		t.Fatalf("decodeHit: %v", err)
	}

	if got, ok := document["pro_number"].(string); !ok || got != "S26100004036460" {
		t.Errorf("pro_number = %#v, want the string S26100004036460", document["pro_number"])
	}
	if got, ok := document["id"].(string); !ok || got != "shp_1" {
		t.Errorf("id = %#v, want the string shp_1", document["id"])
	}
	if got, ok := document["weight"].(float64); !ok || got != 12.5 {
		t.Errorf("weight = %#v, want 12.5", document["weight"])
	}
	if tags, ok := document["tags"].([]any); !ok || len(tags) != 2 {
		t.Errorf("tags = %#v, want a two-element list", document["tags"])
	}
	if document["deleted_at"] != nil {
		t.Errorf("deleted_at = %#v, want nil", document["deleted_at"])
	}
}
