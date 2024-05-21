package schema

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	gen "github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/hook"
	models "github.com/emoss08/trenova/internal/models"
	"github.com/emoss08/trenova/internal/queries"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/google/uuid"
)

// Stop holds the schema definition for the Stop entity.
type Stop struct {
	ent.Schema
}

// Fields of the Stop.
func (Stop) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("New",
				"InProgress",
				"Completed",
				"Voided").
			Default("New").
			StructTag(`json:"status" validate:"required"`),
		field.UUID("shipment_move_id", uuid.UUID{}).
			Immutable().
			StructTag(`json:"shipmentMoveId"`),
		field.Enum("stop_type").
			Values("Pickup", "SplitPickup", "SplitDrop", "Delivery", "DropOff").
			StructTag(`json:"stopType" validate:"required"`),
		field.Int("sequence").
			Positive().
			Default(1).
			Comment("Current sequence of the stop within the movement.").
			StructTag(`json:"sequence" validate:"required"`),
		field.UUID("location_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"locationId" validate:"omitempty"`),
		field.Float("pieces").
			Positive().
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"pieces" validate:"omitempty"`),
		field.Float("weight").
			Positive().
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"weight" validate:"omitempty"`),
		field.String("address_line").
			Optional().
			StructTag(`json:"addressLine" validate:"omitempty"`),
		field.Time("appointment_start").
			Optional().
			Nillable().
			StructTag(`json:"appointmentStart" validate:"required"`),
		field.Time("appointment_end").
			Optional().
			Nillable().
			StructTag(`json:"appointmentEnd" validate:"required"`),
		field.Time("arrival_time").
			Optional().
			Nillable().
			StructTag(`json:"arrivaltime" validate:"omitempty"`),
		field.Time("departure_time").
			Optional().
			Nillable().
			StructTag(`json:"departureTime" validate:"omitempty"`),
	}
}

// Mixin of the Stop.
func (Stop) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Hooks of the Stop.
func (Stop) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return hook.StopFunc(func(ctx context.Context, m *gen.StopMutation) (ent.Value, error) {
					client := m.Client()

					stopID, stopExists := m.ID()
					if !stopExists {
						return next.Mutate(ctx, m)
					}

					shipmentMove, err := queries.GetShipmentMoveByStop(ctx, client, stopID)
					if err != nil {
						return nil, err
					}

					// Validate the shipment.
					validationErrs, err := models.ValidateStop(m, shipmentMove)
					if err != nil {
						return nil, err
					}

					if len(validationErrs) > 0 {
						return nil, &types.ValidationErrorResponse{
							Type:   "validationError",
							Errors: validationErrs,
						}
					}

					return next.Mutate(ctx, m)
				})
			}, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
	}
}

// Edges of the Stop.
func (Stop) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("shipment_move", ShipmentMove.Type).
			Field("shipment_move_id").
			Ref("move_stops").
			Unique().
			Required().
			Immutable().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"shipmentMove,omitempty"`),
	}
}
