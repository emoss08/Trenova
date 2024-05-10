package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Worker holds the schema definition for the Worker entity.
type Worker struct {
	ent.Schema
}

// Fields of the Worker.
func (Worker) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("code").
			NotEmpty().
			MaxLen(10).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(10)",
				dialect.SQLite:   "VARCHAR(10)",
			}).
			StructTag(`json:"code" validate:"required,max=10"`),
		field.String("profile_picture_url").
			Optional().
			StructTag(`json:"profilePictureUrl"`),
		field.Enum("worker_type").
			Values("Employee", "Contractor").
			Default("Employee").
			StructTag(`json:"workerType" validate:"required,oneof=Employee Contractor"`),
		field.String("first_name").
			NotEmpty().
			StructTag(`json:"firstName" validate:"required,max=255"`),
		field.String("last_name").
			NotEmpty().
			StructTag(`json:"lastName" validate:"required,max=255"`),
		field.String("address_line_1").
			Optional().
			MaxLen(150).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(150)",
				dialect.SQLite:   "VARCHAR(150)",
			}).
			StructTag(`json:"addressLine1" validate:"required,max=150"`),
		field.String("address_line_2").
			Optional().
			MaxLen(150).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(150)",
				dialect.SQLite:   "VARCHAR(150)",
			}).
			StructTag(`json:"addressLine2" validate:"omitempty,max=150"`),
		field.String("city").
			Optional().
			MaxLen(150).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(150)",
				dialect.SQLite:   "VARCHAR(150)",
			}).
			StructTag(`json:"city" validate:"required,max=150"`),
		field.String("postal_code").
			Optional().
			MaxLen(10).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(10)",
				dialect.SQLite:   "VARCHAR(10)",
			}).
			StructTag(`json:"postalCode" validate:"omitempty,max=10"`),
		field.UUID("state_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"stateId" validate:"omitempty,uuid"`),
		field.UUID("fleet_code_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"fleetCodeId" validate:"omitempty,uuid"`),
		field.UUID("manager_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"managerId" validate:"omitempty,uuid"`),
		field.String("external_id").
			Optional().
			Comment("External ID usually from HOS integration.").
			StructTag(`json:"externalId" validate:"omitempty"`),
	}
}

// Mixin of the Worker.
func (Worker) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Indexes of the Worker.
func (Worker) Indexes() []ent.Index {
	return []ent.Index{
		// Ensure the code is unique for the organization.
		index.Fields("first_name", "last_name"),
	}
}

// Edges of the Worker.
func (Worker) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("state", UsState.Type).
			Field("state_id").
			StructTag(`json:"state"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("fleet_code", FleetCode.Type).
			Field("fleet_code_id").
			StructTag(`json:"fleetCode"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("manager", User.Type).
			Field("manager_id").
			StructTag(`json:"manager"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("primary_tractor", Tractor.Type).
			Unique().
			StructTag(`json:"primaryTractor"`).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("secondary_tractor", Tractor.Type).
			Unique().
			StructTag(`json:"secondaryTractor"`).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("worker_profile", WorkerProfile.Type).
			Unique(),
		edge.To("worker_comments", WorkerComment.Type),
		edge.To("worker_contacts", WorkerContact.Type),
	}
}
