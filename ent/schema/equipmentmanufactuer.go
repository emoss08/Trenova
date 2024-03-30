package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// EquipmentManufactuer holds the schema definition for the EquipmentManufactuer entity.
type EquipmentManufactuer struct {
	ent.Schema
}

// Fields of the EquipmentManufactuer.
func (EquipmentManufactuer) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(1)",
				dialect.SQLite:   "VARCHAR(1)",
			}).
			Default("A").
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("name").
			NotEmpty().
			StructTag(`json:"name" validate:"required"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
	}
}

// Indexes of the EquipmentManufactuer.
func (EquipmentManufactuer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name", "organization_id").
			Unique(),
	}
}

// Mixin of the EquipmentManufactuer.
func (EquipmentManufactuer) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the EquipmentManufactuer.
func (EquipmentManufactuer) Edges() []ent.Edge {
	return nil
}
