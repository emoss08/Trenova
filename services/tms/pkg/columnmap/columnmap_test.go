package columnmap_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/columnmap"
)

func TestUnmarshalMap(t *testing.T) {
	t.Parallel()

	type testEntity struct {
		ID             string `json:"id"             keel:"id,pk"`
		BusinessUnitID string `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
		OrganizationID string `json:"organizationId" keel:"organization_id,pk,tenant"`
		CreatedAt      int64  `json:"createdAt"      keel:"created_at,autocreate"`
	}

	input := map[string]any{
		"id":               "wrk_123",
		"business_unit_id": "bu_123",
		"organization_id":  "org_123",
		"created_at":       int64(12345),
		"_projection":      "worker-cache",
	}

	entity := new(testEntity)
	if err := columnmap.UnmarshalMap(input, entity); err != nil {
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
