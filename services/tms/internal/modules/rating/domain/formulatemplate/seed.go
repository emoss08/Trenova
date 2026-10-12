package formulatemplate

import (
	"maps"

	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const defaultSchemaID = "shipment"

// Seed is everything that makes a template compute a charge, with none of the
// identity or review history a live template accumulates. Fork, duplicate,
// import, and the standard library all start from one, so a new template is
// born the same way whichever door it came through.
type Seed struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID

	Name        string
	Description string
	Type        TemplateType
	Expression  string
	SchemaID    string

	VariableDefinitions  []*formulatypes.VariableDefinition
	BreakdownDefinitions []*formulatypes.BreakdownDefinition
	MinCharge            decimal.NullDecimal
	MaxCharge            decimal.NullDecimal
	RoundingMode         ratetypes.RoundingMode
	RoundingPrecision    int32
	Metadata             map[string]any

	// Status defaults to Draft. Only the vendor catalog sets Active, because
	// its templates are validated before they land.
	Status Status

	SourceTemplateID    *pulid.ID
	SourceVersionNumber *int64
}

// Build materialises the seed as a template at version one, with empty rather
// than nil collections, normalised rounding, and no review stamps.
func (s Seed) Build() *FormulaTemplate {
	status := s.Status
	if status == "" {
		status = StatusDraft
	}
	schemaID := s.SchemaID
	if schemaID == "" {
		schemaID = defaultSchemaID
	}
	variables := s.VariableDefinitions
	if variables == nil {
		variables = []*formulatypes.VariableDefinition{}
	}
	breakdowns := s.BreakdownDefinitions
	if breakdowns == nil {
		breakdowns = []*formulatypes.BreakdownDefinition{}
	}
	var metadata map[string]any
	if s.Metadata != nil {
		metadata = maps.Clone(s.Metadata)
	}

	template := &FormulaTemplate{
		OrganizationID:       s.OrganizationID,
		BusinessUnitID:       s.BusinessUnitID,
		Name:                 s.Name,
		Description:          s.Description,
		Type:                 s.Type,
		Expression:           s.Expression,
		Status:               status,
		SchemaID:             schemaID,
		VariableDefinitions:  variables,
		BreakdownDefinitions: breakdowns,
		MinCharge:            s.MinCharge,
		MaxCharge:            s.MaxCharge,
		RoundingMode:         s.RoundingMode,
		RoundingPrecision:    s.RoundingPrecision,
		Metadata:             metadata,
		SourceTemplateID:     s.SourceTemplateID,
		SourceVersionNumber:  s.SourceVersionNumber,
		CurrentVersionNumber: 1,
	}
	template.NormalizeRounding()

	return template
}

// SeedFromTemplate copies a live template's content and records where it came
// from. The caller names the result.
func SeedFromTemplate(source *FormulaTemplate) Seed {
	sourceID := source.ID
	sourceVersion := source.CurrentVersionNumber

	return Seed{
		OrganizationID:       source.OrganizationID,
		BusinessUnitID:       source.BusinessUnitID,
		Name:                 source.Name,
		Description:          source.Description,
		Type:                 source.Type,
		Expression:           source.Expression,
		SchemaID:             source.SchemaID,
		VariableDefinitions:  source.VariableDefinitions,
		BreakdownDefinitions: source.BreakdownDefinitions,
		MinCharge:            source.MinCharge,
		MaxCharge:            source.MaxCharge,
		RoundingMode:         source.RoundingMode,
		RoundingPrecision:    source.RoundingPrecision,
		Metadata:             source.Metadata,
		SourceTemplateID:     &sourceID,
		SourceVersionNumber:  &sourceVersion,
	}
}

// SeedFromVersion copies a numbered snapshot's content and records the
// snapshot as the source. Tenant and name are the caller's to set.
func SeedFromVersion(snapshot *FormulaTemplateVersion) Seed {
	sourceID := snapshot.TemplateID
	sourceVersion := snapshot.VersionNumber

	return Seed{
		OrganizationID:       snapshot.OrganizationID,
		BusinessUnitID:       snapshot.BusinessUnitID,
		Name:                 snapshot.Name,
		Description:          snapshot.Description,
		Type:                 snapshot.Type,
		Expression:           snapshot.Expression,
		SchemaID:             snapshot.SchemaID,
		VariableDefinitions:  snapshot.VariableDefinitions,
		BreakdownDefinitions: snapshot.BreakdownDefinitions,
		MinCharge:            snapshot.MinCharge,
		MaxCharge:            snapshot.MaxCharge,
		RoundingMode:         snapshot.RoundingMode,
		RoundingPrecision:    snapshot.RoundingPrecision,
		Metadata:             snapshot.Metadata,
		SourceTemplateID:     &sourceID,
		SourceVersionNumber:  &sourceVersion,
	}
}
