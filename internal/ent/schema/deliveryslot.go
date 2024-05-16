package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/google/uuid"
)

// DeliverySlot holds the schema definition for the DeliverySlot entity.
type DeliverySlot struct {
	ent.Schema
}

// Fields of the DeliverySlot.
func (DeliverySlot) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("customer_id", uuid.UUID{}).
			Immutable().
			Unique().
			StructTag(`json:"customerId" validate:"required"`),
		field.UUID("location_id", uuid.UUID{}).
			Unique().
			StructTag(`json:"locationId" validate:"required"`),
		field.Enum("day_of_week").
			Values("SUNDAY", "MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY").
			StructTag(`json:"dayOfWeek" validate:"required,oneof=SUNDAY MONDAY TUESDAY WEDNESDAY THURSDAY FRIDAY SATURDAY"`),
		field.Other("start_time", &types.TimeOnly{}).
			SchemaType(types.TimeOnly{}.SchemaType()).
			StructTag(`json:"startTime" validate:"required"`),
		field.Other("end_time", &types.TimeOnly{}).
			SchemaType(types.TimeOnly{}.SchemaType()).
			StructTag(`json:"endTime" validate:"required"`),
	}
}

// Edges of the DeliverySlot.
func (DeliverySlot) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("customer", Customer.Type).
			Field("customer_id").
			Ref("delivery_slots").
			Required().
			Immutable().
			Unique(),
		edge.To("location", Location.Type).
			Field("location_id").
			StructTag(`json:"location"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
	}
}

// Indexes of the DeliverySlot.
func (DeliverySlot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields(
			"customer_id",
			"location_id",
			"day_of_week",
			"start_time",
			"end_time").
			Unique(),
	}
}

// Annotations of the DeliverySlot.
func (DeliverySlot) Annotations() []schema.Annotation {
	return []schema.Annotation{
		&entsql.Annotation{
			Checks: map[string]string{
				"valid_start_time_end_time": "start_time < end_time",
			},
		},
	}
}

// Mixin of the DeliverySlot.
func (DeliverySlot) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
