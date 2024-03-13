package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Organization holds the schema definition for the Organization entity.
type Organization struct {
	ent.Schema
}

// Fields of the Organization.
func (Organization) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100),
		field.String("scac_code").
			MaxLen(4).
			StructTag(`json:"scacCode"`),
		field.String("dot_number").
			MaxLen(12).
			StructTag(`json:"dotNumber"`),
		field.String("logo_url").
			Optional().
			StructTag(`json:"logoUrl"`),
		field.Enum("org_type").
			Values("A", "B", "X").
			Default("A").
			StructTag(`json:"orgType"`),
		field.Enum("timezone").
			Values("TimezoneAmericaLosAngeles", "TimezoneAmericaDenver", "TimezoneAmericaChicago", "TimezoneAmericaNewYork").
			Default("TimezoneAmericaLosAngeles"),
		field.UUID("business_unit_id", uuid.UUID{}).
			StructTag(`json:"businessUnitId"`),
	}
}

// Edges of the Organization.
func (Organization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("business_unit", BusinessUnit.Type).
			Field("business_unit_id").
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
	}
}

// Mixin of the Organization.
func (Organization) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}
