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
	"github.com/emoss08/trenova/internal/ent/worker"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Tractor holds the schema definition for the Tractor entity.
type Tractor struct {
	ent.Schema
}

// Fields of the Tractor.
func (Tractor) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			NotEmpty().
			MaxLen(50).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(50)",
				dialect.SQLite:   "VARCHAR(50)",
			}).
			Comment("The unique code assigned to each tractor for identification purposes.").
			StructTag(`json:"code" validate:"required,max=50"`),
		field.Enum("status").
			Values("Available", "OutOfService", "AtMaintenance", "Sold", "Lost").
			Default("Available").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(13)",
				dialect.SQLite:   "VARCHAR(13)",
			}).
			Comment("The operational status of the tractor, indicating availability, maintenance, or other conditions.").
			StructTag(`json:"status" validate:"required,oneof=Available OutOfService AtMaintenance Sold Lost"`),
		field.UUID("equipment_type_id", uuid.UUID{}).
			Optional().
			Unique().
			Comment("Identifier for the type of equipment the tractor is classified under.").
			StructTag(`json:"equipmentTypeId" validate:"required,uuid"`),
		field.String("license_plate_number").
			MaxLen(50).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(50)",
				dialect.SQLite:   "VARCHAR(50)",
			}).
			Comment("The license plate number of the tractor, used for legal identification on roads.").
			StructTag(`json:"licensePlateNumber" validate:"omitempty,max=50"`),
		field.String("vin").
			// Match(regexp.MustCompile("^[0-9A-HJ-NPR-Z]{17}$")). // VIN regex.
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(17)",
				dialect.SQLite:   "VARCHAR(17)",
			}).
			Comment("The Vehicle Identification Number, a unique code used to identify individual motor vehicles.").
			StructTag(`json:"vin" validate:"omitempty,alphanum,len=17"`),
		field.UUID("equipment_manufacturer_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("The UUID of the manufacturer of the tractor's equipment, linking to specific company details.").
			StructTag(`json:"equipmentManufacturerId" validate:"omitempty,uuid"`),
		field.String("model").
			MaxLen(50).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(50)",
				dialect.SQLite:   "VARCHAR(50)",
			}).
			Comment("The model of the tractor, which indicates the design and technical specifications.").
			StructTag(`json:"model" validate:"omitempty,max=50"`),
		field.Int16("year").
			Positive().
			Nillable().
			Optional().
			Comment("The year the tractor was manufactured, reflecting its age and potentially its technology level.").
			StructTag(`json:"year" validate:"omitempty,gt=0"`),
		field.UUID("state_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("A UUID representing the state in which the tractor is registered, for jurisdiction purposes.").
			StructTag(`json:"stateId" validate:"omitempty,uuid"`),
		field.Bool("leased").
			Default(false).
			Comment("Indicates whether the tractor is currently leased or owned outright.").
			StructTag(`json:"leased" validate:"omitempty"`),
		field.Other("leased_date", &pgtype.Date{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			Comment("The date on which the tractor was leased, if applicable.").
			StructTag(`json:"leasedDate" validate:"omitempty"`),
		field.UUID("primary_worker_id", uuid.UUID{}).
			Comment("The primary worker assigned to operate the tractor, identified by UUID.").
			StructTag(`json:"primaryWorkerId" validate:"omitempty,uuid"`),
		field.UUID("secondary_worker_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("An optional secondary worker who can also operate the tractor, identified by UUID.").
			StructTag(`json:"secondaryWorkerId" validate:"omitempty,uuid"`),
		field.UUID("fleet_code_id", uuid.UUID{}).
			Comment("A UUID linking the tractor to a specific fleet within an organization.").
			StructTag(`json:"fleetCodeId" validate:"omitempty,uuid"`),
	}
}

// Mixin of the Tractor.
func (Tractor) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the Tractor.
func (Tractor) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("equipment_type", EquipmentType.Type).
			Field("equipment_type_id").
			StructTag(`json:"equipmentType"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
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
		edge.From("primary_worker", Worker.Type).
			Ref("primary_tractor").
			Field("primary_worker_id").
			StructTag(`json:"primaryWorker"`).
			Required().
			Unique(),
		edge.From("secondary_worker", Worker.Type).
			Ref("secondary_tractor").
			Field("secondary_worker_id").
			StructTag(`json:"secondaryWorker"`).
			Unique(),
		edge.To("fleet_code", FleetCode.Type).
			Field("fleet_code_id").
			StructTag(`json:"fleetCode"`).
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
	}
}

// Hooks for the Tractor.
func (Tractor) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(workerDifferentiationHook, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
		hook.On(leasedDateVerificationHook, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
		hook.On(fleetCodeConsistencyHook, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
	}
}

// workerDifferentiationHook ensures that the primary and secondary workers are not the same.
func workerDifferentiationHook(next ent.Mutator) ent.Mutator {
	return hook.TractorFunc(func(ctx context.Context, m *gen.TractorMutation) (ent.Value, error) {
		if !m.Op().Is(ent.OpCreate) && !m.Op().Is(ent.OpUpdate) && !m.Op().Is(ent.OpUpdateOne) {
			return next.Mutate(ctx, m)
		}

		primaryWorkerID, primaryWorkerIDExists := m.PrimaryWorkerID()
		secondaryWorkerID, secondaryWorkerIDExists := m.SecondaryWorkerID()

		if primaryWorkerIDExists && secondaryWorkerIDExists && primaryWorkerID == secondaryWorkerID {
			return nil, util.NewValidationError("The primary and secondary workers cannot be the same. Please try again.",
				"invalidWorkers", "primaryWorkerId")
		}

		return next.Mutate(ctx, m)
	})
}

// leasedDateVerificationHook ensures that the leased date is set if the tractor is leased.
func leasedDateVerificationHook(next ent.Mutator) ent.Mutator {
	return hook.TractorFunc(func(ctx context.Context, m *gen.TractorMutation) (ent.Value, error) {
		if !m.Op().Is(ent.OpCreate) && !m.Op().Is(ent.OpUpdate) && !m.Op().Is(ent.OpUpdateOne) {
			return next.Mutate(ctx, m)
		}

		leased, leasedExists := m.Leased()
		_, leasedDateExists := m.LeasedDate()

		if leasedExists && leased && !leasedDateExists {
			return nil, util.NewValidationError("The leased date must be set if the tractor is leased. Please try again.",
				"invalidLeasedDate", "leasedDate")
		}

		return next.Mutate(ctx, m)
	})
}

// fleetCodeConsistencyHook validates that the primary worker and tractor have the same fleet code.
func fleetCodeConsistencyHook(next ent.Mutator) ent.Mutator {
	return hook.TractorFunc(func(ctx context.Context, m *gen.TractorMutation) (ent.Value, error) {
		if !m.Op().Is(ent.OpCreate) && !m.Op().Is(ent.OpUpdate) && !m.Op().Is(ent.OpUpdateOne) {
			return next.Mutate(ctx, m)
		}

		fleetCodeID, fleetCodeIDExists := m.FleetCodeID()
		primaryWorkerID, primaryWorkerIDExists := m.PrimaryWorkerID()

		if !fleetCodeIDExists || !primaryWorkerIDExists {
			return nil, util.NewValidationError("Both tractor and primary worker must have valid fleet codes. Please try again.",
				"invalidFleetCode", "fleetCodeId")
		}

		primaryWorkerFleetCode, err := m.Client().Worker.Query().Where(worker.IDEQ(primaryWorkerID)).QueryFleetCode().Only(ctx)
		if err != nil {
			return next.Mutate(ctx, m)
		}

		if primaryWorkerFleetCode.ID != fleetCodeID {
			return nil, util.NewValidationError("The primary worker and tractor must have the same fleet code. Please try again.",
				"invalidFleetCode", "fleetCodeId")
		}

		return next.Mutate(ctx, m)
	})
}
