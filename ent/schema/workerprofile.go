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

	"github.com/jackc/pgx/v5/pgtype"
)

// WorkerProfile holds the schema definition for the WorkerProfile entity.
type WorkerProfile struct {
	ent.Schema
}

// Fields of the WorkerProfile.
func (WorkerProfile) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("worker_id", uuid.UUID{}).
			Unique().
			Immutable().
			StructTag(`json:"workerId" validate:"required,uuid"`),
		field.String("race").
			Optional().
			StructTag(`json:"race" validate:"omitempty"`),
		field.String("sex").
			Optional().
			StructTag(`validate:"omitempty"`),
		field.Other("date_of_birth", &pgtype.Date{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"dateOfBirth" validate:"omitempty"`),
		field.String("license_number").
			NotEmpty().
			StructTag(`json:"licenseNumber" validate:"required"`),
		field.UUID("license_state_id", uuid.UUID{}).
			StructTag(`json:"licenseStateId" validate:"omitempty,uuid"`),
		field.Other("license_expiration_date", &pgtype.Date{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"licenseExpirationDate" validate:"omitempty"`),
		field.Enum("endorsements").
			Values("None", "Tanker", "Hazmat", "TankerHazmat").
			Default("None").
			Optional().
			StructTag(`json:"endorsements" validate:"omitempty"`),
		field.Other("hazmat_expiration_date", &pgtype.Date{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"hazmatExpirationDate" validate:"omitempty"`),
		field.Other("hire_date", &pgtype.Date{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"hireDate" validate:"omitempty"`),
		field.Other("termination_date", &pgtype.Date{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"terminationDate" validate:"omitempty"`),
		field.Other("physical_due_date", &pgtype.Date{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"physicalDueDate" validate:"omitempty"`),
		field.Other("medical_cert_date", &pgtype.Date{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"medicalCertDate" validate:"omitempty"`),
		field.Other("mvr_due_date", &pgtype.Date{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"mvrDueDate" validate:"omitempty"`),
	}
}

// Edges of the WorkerProfile.
func (WorkerProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("worker", Worker.Type).
			Field("worker_id").
			Ref("worker_profile").
			Immutable().
			Unique().
			Required(),
		edge.To("state", UsState.Type).
			Field("license_state_id").
			StructTag(`json:"state"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
	}
}

// Mixin of the WorkerProfile.
func (WorkerProfile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Hooks for the WorkerProfile.
func (WorkerProfile) Hooks() []ent.Hook {
	return []ent.Hook{
		// Hook that ensures if the worker has an TankerHazmat and/or Hazmat endorsement, the hazmat expiration date is set.
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return hook.WorkerProfileFunc(func(ctx context.Context, m *gen.WorkerProfileMutation) (ent.Value, error) {
					if !m.Op().Is(ent.OpCreate) && !m.Op().Is(ent.OpUpdate) && !m.Op().Is(ent.OpUpdateOne) {
						return next.Mutate(ctx, m)
					}

					// Get the worker endorsement value.
					endorsement, endorsementExists := m.Endorsements()

					// If the worker has a TankerHazmat or Hazmat endorsement, ensure the hazmat expiration date is set.
					if endorsementExists && (endorsement == "TankerHazmat" || endorsement == "Hazmat") {
						_, hazmatExpirationExists := m.HazmatExpirationDate()

						if !hazmatExpirationExists {
							return nil, tools.NewValidationError("Hazmat Expiration date is required for this endorsement. Please try again.",
								"invalidEndorsement",
								"hazmatExpirationDate")
						}
						return next.Mutate(ctx, m)
					}

					return next.Mutate(ctx, m)
				})
			}, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
	}
}
