package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/gtc/internal/core/domain"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("CDC_AUTO_CREATE_SLOT", "")
	t.Setenv("CDC_INACTIVE_SLOT_ACTION", "")
	t.Setenv("CDC_MAX_LAG_BYTES", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.AutoCreateSlot {
		t.Fatalf("expected auto create slot default to be false")
	}
	if cfg.InactiveSlotAction != "fail" {
		t.Fatalf("expected inactive slot action default to be fail, got %s", cfg.InactiveSlotAction)
	}
	if cfg.MaxLagBytes != 5*1024*1024*1024 {
		t.Fatalf("expected max lag bytes default to be 5GB, got %d", cfg.MaxLagBytes)
	}
}

func TestLoadRejectsInvalidInactiveSlotAction(t *testing.T) {
	t.Setenv("CDC_INACTIVE_SLOT_ACTION", "invalid")

	if _, err := Load(); err == nil {
		t.Fatalf("expected validation error for invalid inactive slot action")
	}
}

func TestLoadRejectsInvalidMaxLagBytes(t *testing.T) {
	t.Setenv("CDC_MAX_LAG_BYTES", "abc")

	if _, err := Load(); err == nil {
		t.Fatalf("expected validation error for invalid max lag bytes")
	}
}

func TestLoadProjections(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "gtc.yaml")
	content := `
projections:
  - name: shipment-search
    source_table: public.shipments
    primary_keys: [id, organization_id, business_unit_id]
    searchable_fields: [pro_number, bol]
    filterable_fields: [organization_id, business_unit_id]
    destination:
      kind: meilisearch
      index: shipments
  - name: shipment-cache
    source_table: public.shipments
    primary_keys: [id]
    destination:
      kind: redis_json
      key_template: 'cache:shipments:{{ field "id" .New }}'
`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	projections, err := LoadProjections(path)
	if err != nil {
		t.Fatalf("LoadProjections returned error: %v", err)
	}

	if len(projections) != 2 {
		t.Fatalf("expected 2 projections, got %d", len(projections))
	}

	if projections[0].Destination.Kind != domain.DestinationMeilisearch {
		t.Fatalf("expected meilisearch destination, got %s", projections[0].Destination.Kind)
	}

	if projections[1].Destination.KeyTemplate == "" {
		t.Fatalf("expected redis json key template to be populated")
	}

	if len(projections[0].PrimaryKeys) != 3 {
		t.Fatalf("expected composite keys to load, got %v", projections[0].PrimaryKeys)
	}

	if len(projections[0].FilterableFields) != 2 {
		t.Fatalf("expected filterable fields to load, got %v", projections[0].FilterableFields)
	}
}

func TestProjectionConfigValidateRejectsInvalidDestination(t *testing.T) {
	t.Parallel()

	cfg := ProjectionConfig{
		Name:        "invalid",
		SourceTable: "public.shipments",
		Destination: DestinationConfig{
			Kind: domain.DestinationMeilisearch,
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}
