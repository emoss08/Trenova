package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// FormulaTemplate holds the schema definition for the FormulaTemplate entity.
type FormulaTemplate struct {
	ent.Schema
}

// Fields of the FormulaTemplate.
func (FormulaTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			StructTag(`json:"name" validate:"required"`),
		field.Text("formula_text").
			NotEmpty().
			StructTag(`json:"formulaText" validate:"required"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
		field.Enum("template_type").
			Values("Refrigerated", "Hazardous", "General").
			Default("General").
			StructTag(`json:"templateType" validate:"required,oneof=Refrigerated Hazardous General"`),
		field.UUID("customer_id", uuid.UUID{}).
			Nillable().
			Optional().
			StructTag(`json:"customerId" validate:"required"`),
		field.UUID("shipment_type_id", uuid.UUID{}).
			Nillable().
			Optional().
			StructTag(`json:"shipmentTypeId" validate:"required"`),
		field.Bool("auto_apply").
			Default(false).
			StructTag(`json:"autoApply" validate:"required"`),
	}
}

// Mixin of the FormulaTemplate.
func (FormulaTemplate) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the FormulaTemplate.
func (FormulaTemplate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("customer", Customer.Type).
			Field("customer_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"customer,omitempty"`),
		edge.To("shipment_type", ShipmentType.Type).
			Field("shipment_type_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"shipment_type,omitempty"`),
	}
}
