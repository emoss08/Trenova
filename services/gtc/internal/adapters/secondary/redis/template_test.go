package redis

import (
	"testing"

	"github.com/emoss08/gtc/internal/core/domain"
)

func TestTemplateExecuteUsesNewAndOldData(t *testing.T) {
	t.Parallel()

	tmpl, err := ParseTemplate(`cache:shipments:{{ value "id" .New .Old }}`)
	if err != nil {
		t.Fatalf("ParseTemplate returned error: %v", err)
	}

	value, err := tmpl.Execute(domain.SourceRecord{
		Schema: "public",
		Table:  "shipments",
		NewData: map[string]any{
			"id": "shp_new",
		},
	}, nil)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if value != "cache:shipments:shp_new" {
		t.Fatalf("unexpected template output: %s", value)
	}

	value, err = tmpl.Execute(domain.SourceRecord{
		Schema: "public",
		Table:  "shipments",
		OldData: map[string]any{
			"id": "shp_old",
		},
	}, nil)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if value != "cache:shipments:shp_old" {
		t.Fatalf("unexpected template output for old data: %s", value)
	}
}

func TestTemplateExecuteUsesPrimaryKeys(t *testing.T) {
	t.Parallel()

	tmpl, err := ParseTemplate(`cache:shipments:{{ key .PrimaryKeys .New .Old }}`)
	if err != nil {
		t.Fatalf("ParseTemplate returned error: %v", err)
	}

	value, err := tmpl.Execute(domain.SourceRecord{
		Schema: "public",
		Table:  "shipments",
		NewData: map[string]any{
			"id":               "shp_1",
			"organization_id":  "org_1",
			"business_unit_id": "bu_1",
		},
	}, []string{"id", "organization_id", "business_unit_id"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if value != "cache:shipments:shp_1:org_1:bu_1" {
		t.Fatalf("unexpected composite key output: %s", value)
	}
}
