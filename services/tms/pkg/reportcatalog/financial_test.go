package reportcatalog

import (
	"strings"
	"testing"
)

func TestMinorUnitFieldsAreNotFormattedAsMoney(t *testing.T) {
	for i := range Default.Entities {
		entity := &Default.Entities[i]
		for j := range entity.Fields {
			field := &entity.Fields[j]
			if !strings.HasSuffix(field.Key, "Minor") {
				continue
			}
			if field.Format != FormatNone {
				t.Errorf(
					"%s.%s is a minor-unit column formatted as %q; the money hint applies currency styling without scaling, so cents would render as whole units",
					entity.Key, field.Key, field.Format,
				)
			}
			if !strings.Contains(field.Label, "Minor") {
				t.Errorf(
					"%s.%s is a minor-unit column labelled %q; the unit belongs in the label because no format hint carries it",
					entity.Key, field.Key, field.Label,
				)
			}
		}
	}
}

func TestInstantsTheColumnHeuristicMissesAreDeclaredEpoch(t *testing.T) {
	declared := map[string][]string{
		"driver_settlement":          {"periodStart", "periodEnd"},
		"driver_settlement_batch":    {"periodStart", "periodEnd"},
		"fiscal_period":              {"adjustmentDeadline"},
		"rate_agreement":             {"effectiveFrom", "effectiveTo"},
		"rate_agreement_rule":        {"effectiveFrom", "effectiveTo"},
		"rate_agreement_accessorial": {"effectiveFrom", "effectiveTo"},
		"rate_quote":                 {"asOf"},
	}

	for entityKey, fieldKeys := range declared {
		entity, ok := Default.Entity(entityKey)
		if !ok {
			t.Errorf("entity %q not found", entityKey)
			continue
		}
		for _, fieldKey := range fieldKeys {
			field, found := entity.Field(fieldKey)
			if !found {
				t.Errorf("%s.%s not found", entityKey, fieldKey)
				continue
			}
			if field.Type != FieldEpoch {
				t.Errorf(
					"%s.%s type = %s, want epoch; its column name ends in none of the suffixes the epoch heuristic reads, so it must be declared",
					entityKey, fieldKey, field.Type,
				)
			}
		}
	}
}

func TestGLAccountBalanceIsKeyedByAccountAndPeriod(t *testing.T) {
	entity, ok := Default.Entity("gl_account_balance")
	if !ok {
		t.Skip("gl_account_balance is not in the catalog yet")
	}

	want := []string{
		"organization_id", "business_unit_id", "gl_account_id",
		"fiscal_year_id", "fiscal_period_id",
	}
	if len(entity.Table.PrimaryKey) != len(want) {
		t.Fatalf("primary key = %v, want the five-column account-period key %v",
			entity.Table.PrimaryKey, want)
	}
	for _, column := range want {
		if !containsString(entity.Table.PrimaryKey, column) {
			t.Errorf("primary key %v is missing %q", entity.Table.PrimaryKey, column)
		}
	}

	if _, found := entity.Field("id"); found {
		t.Error("gl_account_balance has an id field; a count of these rows counts account-periods, not documents, and any measure keyed on id would be wrong")
	}
}

func TestEnumFieldsResolveTheirValues(t *testing.T) {
	for i := range Default.Entities {
		entity := &Default.Entities[i]
		for j := range entity.Fields {
			field := &entity.Fields[j]
			if field.Type == FieldEnum && len(field.EnumValues) == 0 {
				t.Errorf("%s.%s is an enum with no resolved values", entity.Key, field.Key)
			}
		}
	}

	// rounding_mode is backed by ratetypes.RoundingMode, which lives outside the
	// domain directory. Without that package indexed for enum resolution the
	// field degrades to a free-text string behind a warning nobody reads.
	agreement, ok := Default.Entity("rate_agreement")
	if !ok {
		t.Fatal("rate_agreement entity not found")
	}
	rounding, found := agreement.Field("roundingMode")
	if !found {
		t.Fatal("rate_agreement.roundingMode not found")
	}
	if rounding.Type != FieldEnum {
		t.Errorf("rate_agreement.roundingMode type = %s, want enum", rounding.Type)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
