package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// FeasibilityToolControl holds the schema definition for the FeasibilityToolControl entity.
type FeasibilityToolControl struct {
	ent.Schema
}

// Fields of the FeasibilityToolControl.
func (FeasibilityToolControl) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("otp_operator").
			Values("Eq", "Ne", "Gt", "Gte", "Lt", "Lte").
			Default("Eq").
			StructTag(`json:"otpOperator"`),
		field.Float("otp_value").
			Default(100).
			StructTag(`json:"otpValue"`),
		field.Enum("mpw_operator").
			Values("Eq", "Ne", "Gt", "Gte", "Lt", "Lte").
			Default("Eq").
			StructTag(`json:"mpwOperator"`),
		field.Float("mpw_value").
			Default(100).
			StructTag(`json:"mpwValue"`),
		field.Enum("mpd_operator").
			Values("Eq", "Ne", "Gt", "Gte", "Lt", "Lte").
			Default("Eq").
			StructTag(`json:"mpdOperator"`),
		field.Float("mpd_value").
			Default(100).
			StructTag(`json:"mpdValue"`),
		field.Enum("mpg_operator").
			Values("Eq", "Ne", "Gt", "Gte", "Lt", "Lte").
			Default("Eq").
			StructTag(`json:"mpgOperator"`),
		field.Float("mpg_value").
			Default(100).
			StructTag(`json:"mpgValue"`),
	}
}

func (FeasibilityToolControl) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

func (FeasibilityToolControl) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.QueryField(),
		entgql.Mutations(entgql.MutationCreate()),
	}
}

// Edges of the FeasibilityToolControl.
func (FeasibilityToolControl) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("feasibility_tool_control").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
		edge.To("business_unit", BusinessUnit.Type).
			StorageKey(edge.Column("business_unit_id")).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
	}
}
