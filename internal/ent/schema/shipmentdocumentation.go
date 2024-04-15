package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ShipmentDocumentation holds the schema definition for the ShipmentDocumentation entity.
type ShipmentDocumentation struct {
	ent.Schema
}

// Fields of the ShipmentDocumentation.
func (ShipmentDocumentation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("shipment_id", uuid.UUID{}).
			Immutable().
			StructTag(`json:"shipmentId" validate:"omitempty"`), // Shipment ID will be set by the system.
		field.String("document_url").
			NotEmpty().
			StructTag(`json:"documentUrl" validate:"required"`),
		field.UUID("document_classification_id", uuid.UUID{}).
			StructTag(`json:"documentClassificationId" validate:"omitempty"`),
	}
}

// Mixin of the ShipmentDocumentation.
func (ShipmentDocumentation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the ShipmentDocumentation.
func (ShipmentDocumentation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("shipment", Shipment.Type).
			Ref("shipment_documentation").
			Unique().
			Field("shipment_id").
			Immutable().
			Required(),
		edge.From("document_classification", DocumentClassification.Type).
			Ref("shipment_documentation").
			Field("document_classification_id").
			Unique().
			Required(),
	}
}
