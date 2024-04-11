package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
)

// EquipmentType holds the schema definition for the EquipmentType entity.
type EquipmentType struct {
	ent.Schema
}

// Fields of the EquipmentType.
func (EquipmentType) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("code").
			NotEmpty().
			MaxLen(10).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(10)",
				dialect.SQLite:   "VARCHAR(10)",
			}).
			StructTag(`json:"code" validate:"required,max=50"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
		field.Float("cost_per_mile").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"costPerMile" validate:"omitempty"`),
		field.Enum("equipment_class").
			Values("Undefined",
				"Car",
				"Van",
				"Pickup",
				"Straight",
				"Tractor",
				"Trailer",
				"Container",
				"Chassis",
				"Other").
			Default("Undefined").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(10)",
				dialect.SQLite:   "VARCHAR(10)",
			}).
			StructTag(`json:"equipmentClass" validate:"required,oneof=Undefined Car Van Pickup Straight Tractor Trailer Container Chassis Other"`),
		field.Float("fixed_cost").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"fixedCost" validate:"omitempty"`),
		field.Float("variable_cost").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"variableCost" validate:"omitempty"`),
		field.Float("height").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"height" validate:"omitempty"`),
		field.Float("length").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"length" validate:"omitempty"`),
		field.Float("width").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"width" validate:"omitempty"`),
		field.Float("weight").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"weight" validate:"omitempty"`),
		field.Float("idling_fuel_usage").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"idlingFuelUsage" validate:"omitempty"`),
		field.Bool("exempt_from_tolls").
			Default(false).
			StructTag(`json:"exemptFromTolls" validate:"omitempty"`),
		field.String("color").
			Optional().
			StructTag(`json:"color" validate:"omitempty"`),
	}
}

// Mixin of the EquipmentType.
func (EquipmentType) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the EquipmentType.
func (EquipmentType) Edges() []ent.Edge {
	return nil
}
