package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// UsState holds the schema definition for the UsState entity.
type UsState struct {
	ent.Schema
}

// Fields of the UsState.
func (UsState) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(255).
			StructTag(`json:"name"`),
		field.String("abbreviation").
			MaxLen(5).
			StructTag(`json:"abbreviation"`),
		field.String("country_name").
			MaxLen(255).
			Default("United States").
			StructTag(`json:"countryName"`),
		field.String("country_iso3").
			MaxLen(3).
			Default("USA").
			StructTag(`json:"countryIso3"`),
	}
}

// Edges of the UsState.
func (UsState) Edges() []ent.Edge {
	return nil
}

// Mixin of the UsState.
func (UsState) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}
