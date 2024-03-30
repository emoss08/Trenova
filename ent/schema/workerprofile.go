package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
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
			Optional().
			Nillable().
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
	}
}

// Mixin of the WorkerProfile.
func (WorkerProfile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
