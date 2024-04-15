package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// FleetCode holds the schema definition for the FleetCode entity.
type FleetCode struct {
	ent.Schema
}

// Fields of the FleetCode.
func (FleetCode) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("code").
			NotEmpty().
			MaxLen(10).
			StructTag(`json:"code" validate:"required,max=10"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
		field.Float("revenue_goal").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"revenueGoal" validate:"omitempty"`),
		field.Float("deadhead_goal").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"deadheadGoal" validate:"omitempty"`),
		field.Float("mileage_goal").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"mileageGoal" validate:"omitempty"`),
		field.UUID("manager_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"managerId" validate:"omitempty"`),
		field.String("color").
			Optional().
			StructTag(`json:"color" validate:"omitempty"`),
	}
}

// Edges of the FleetCode.
func (FleetCode) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("manager", User.Type).
			Field("manager_id").
			StructTag(`json:"manager"`).
			Unique(),
	}
}

// Mixin of the FleetCode.
func (FleetCode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
