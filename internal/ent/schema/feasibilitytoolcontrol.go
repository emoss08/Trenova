package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
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
			StructTag(`json:"otpOperator" validate:"required,oneof=Eq Ne Gt Gte Lt Lte"`),
		field.Float("otp_value").
			Default(100).
			StructTag(`json:"otpValue" validate:"required,gt=0"`),
		field.Enum("mpw_operator").
			Values("Eq", "Ne", "Gt", "Gte", "Lt", "Lte").
			Default("Eq").
			StructTag(`json:"mpwOperator" validate:"required,oneof=Eq Ne Gt Gte Lt Lte"`),
		field.Float("mpw_value").
			Default(100).
			StructTag(`json:"mpwValue" validate:"required,gt=0"`),
		field.Enum("mpd_operator").
			Values("Eq", "Ne", "Gt", "Gte", "Lt", "Lte").
			Default("Eq").
			StructTag(`json:"mpdOperator" validate:"required,oneof=Eq Ne Gt Gte Lt Lte"`),
		field.Float("mpd_value").
			Default(100).
			StructTag(`json:"mpdValue" validate:"required,gt=0"`),
		field.Enum("mpg_operator").
			Values("Eq", "Ne", "Gt", "Gte", "Lt", "Lte").
			Default("Eq").
			StructTag(`json:"mpgOperator" validate:"required,oneof=Eq Ne Gt Gte Lt Lte"`),
		field.Float("mpg_value").
			Default(100).
			StructTag(`json:"mpgValue" validate:"required,gt=0"`),
	}
}

func (FeasibilityToolControl) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
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
