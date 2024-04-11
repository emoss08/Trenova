package schema

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	gen "github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/hook"
	"github.com/emoss08/trenova/tools"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"

	"github.com/jackc/pgx/v5/pgtype"
)

// Trailer holds the schema definition for the Trailer entity.
type Trailer struct {
	ent.Schema
}

// Fields of the Trailer.
func (Trailer) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			NotEmpty().
			MaxLen(50).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(50)",
				dialect.SQLite:   "VARCHAR(50)",
			}).
			StructTag(`json:"code" validate:"required,max=50"`),
		field.Enum("status").
			Values("Available", "OutOfService", "AtMaintenance", "Sold", "Lost").
			Default("Available").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(13)",
				dialect.SQLite:   "VARCHAR(13)",
			}).
			StructTag(`json:"status" validate:"required,oneof=Available OutOfService AtMaintenance Sold Lost"`),
		field.UUID("equipment_type_id", uuid.UUID{}).
			Unique().
			StructTag(`json:"equipmentTypeId" validate:"required,uuid"`),
		field.String("vin").
			// Match(regexp.MustCompile("^[0-9A-HJ-NPR-Z]{17}$")). // VIN regex.
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(17)",
				dialect.SQLite:   "VARCHAR(17)",
			}).
			StructTag(`json:"vin" validate:"omitempty,alphanum,len=17"`),
		field.UUID("equipment_manufacturer_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"equipmentManufacturerId" validate:"omitempty,uuid"`),
		field.String("model").
			MaxLen(50).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(50)",
				dialect.SQLite:   "VARCHAR(50)",
			}).
			StructTag(`json:"model" validate:"omitempty,max=50"`),
		field.Int16("year").
			Positive().
			Nillable().
			Optional().
			StructTag(`json:"year" validate:"omitempty,gt=0"`),
		field.String("license_plate_number").
			MaxLen(50).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(50)",
				dialect.SQLite:   "VARCHAR(50)",
			}).
			StructTag(`json:"licensePlateNumber" validate:"omitempty,max=50"`),
		field.UUID("state_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"stateId" validate:"omitempty,uuid"`),
		field.UUID("fleet_code_id", uuid.UUID{}).
			StructTag(`json:"fleetCodeId" validate:"omitempty,uuid"`),
		field.Other("last_inspection_date", &pgtype.Date{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"lastInspectionDate" validate:"omitempty"`),
		field.String("registration_number").
			Optional().
			StructTag(`json:"registrationNumber" validate:"omitempty"`),
		field.UUID("registration_state_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"registrationStateId" validate:"omitempty,uuid"`),
		field.Other("registration_expiration_date", &pgtype.Date{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"registrationExpirationDate" validate:"omitempty"`),
	}
}

// Mixin of the Trailer.
func (Trailer) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the Trailer.
func (Trailer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("equipment_type", EquipmentType.Type).
			Field("equipment_type_id").
			StructTag(`json:"equipmentType"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
		edge.To("equipment_manufacturer", EquipmentManufactuer.Type).
			Field("equipment_manufacturer_id").
			StructTag(`json:"equipmentManufacturer"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("state", UsState.Type).
			Field("state_id").
			StructTag(`json:"state"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("registration_state", UsState.Type).
			Field("registration_state_id").
			StructTag(`json:"registrationState"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("fleet_code", FleetCode.Type).
			Field("fleet_code_id").
			StructTag(`json:"fleetCode"`).
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
	}
}

func (Trailer) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				// Ensure the equipment type `equipment_class` is equal to `Trailer`.
				return hook.TrailerFunc(func(ctx context.Context, m *gen.TrailerMutation) (ent.Value, error) {
					if !m.Op().Is(ent.OpCreate) && !m.Op().Is(ent.OpUpdate) && !m.Op().Is(ent.OpUpdateOne) {
						return next.Mutate(ctx, m)
					}

					// Get the equipment type.
					equipmentType, exists := m.EquipmentTypeID()
					// If the equipment type ID does not exist just mutate.
					if !exists {
						return next.Mutate(ctx, m)
					}

					// Get the equipment type.
					et, err := m.Client().EquipmentType.Get(ctx, equipmentType)
					if err != nil {
						return nil, eris.Wrap(err, "failed to get equipment type")
					}

					// If the equipment class is not equal to `Trailer` return an error.
					if et.EquipmentClass != "Trailer" {
						return nil, tools.NewValidationError("Cannot assign a non-trailer equipment type to a trailer. Please try again.",
							"invalidEquipmentType",
							"equipmentTypeId")
					}

					return next.Mutate(ctx, m)
				})
			}, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
	}
}
