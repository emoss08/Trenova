package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ShipmentComment holds the schema definition for the ShipmentComment entity.
type ShipmentComment struct {
	ent.Schema
}

// Fields of the ShipmentComment.
func (ShipmentComment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("shipment_id", uuid.UUID{}).
			Immutable().
			StructTag(`json:"shipmentId" validate:"omitempty"`), // Shipment ID will be set by the system.
		field.UUID("comment_type_id", uuid.UUID{}).
			StructTag(`json:"commentTypeId" validate:"omitempty"`),
		field.Text("comment").
			NotEmpty().
			StructTag(`json:"comment" validate:"required"`),
		field.UUID("created_by", uuid.UUID{}).
			StructTag(`json:"createdBy" validate:"omitempty"`),
	}
}

// Edges of the ShipmentComment.
func (ShipmentComment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("shipment", Shipment.Type).
			Ref("shipment_comments").
			Unique().
			Field("shipment_id").
			Immutable().
			Required(),
		edge.From("comment_type", CommentType.Type).
			Ref("shipment_comments").
			Field("comment_type_id").
			Unique().
			Required(),
		edge.From("created_by_user", User.Type).
			Ref("shipment_comments").
			Field("created_by").
			Unique().
			Required(),
	}
}
