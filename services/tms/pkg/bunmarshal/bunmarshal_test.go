package bunmarshal_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/bunmarshal"
)

func TestUnmarshalMap(t *testing.T) {
	t.Parallel()

	type testEntity struct {
		ID             string `json:"id"             bun:"id,pk"`
		BusinessUnitID string `json:"businessUnitId" bun:"business_unit_id"`
		OrganizationID string `json:"organizationId" bun:"organization_id"`
		CreatedAt      int64  `json:"createdAt"      bun:"created_at"`
	}

	input := map[string]any{
		"id":               "wrk_123",
		"business_unit_id": "bu_123",
		"organization_id":  "org_123",
		"created_at":       int64(12345),
		"_projection":      "worker-cache",
	}

	entity := new(testEntity)
	if err := bunmarshal.UnmarshalMap(input, entity); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}

	if entity.ID != "wrk_123" {
		t.Fatalf("expected id to match, got %q", entity.ID)
	}
	if entity.BusinessUnitID != "bu_123" {
		t.Fatalf("expected business unit id to match, got %q", entity.BusinessUnitID)
	}
	if entity.OrganizationID != "org_123" {
		t.Fatalf("expected organization id to match, got %q", entity.OrganizationID)
	}
	if entity.CreatedAt != 12345 {
		t.Fatalf("expected created at to match, got %d", entity.CreatedAt)
	}
}
