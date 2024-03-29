package schema

import (
	"entgo.io/ent"
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
			MaxLen(255).
			StructTag(`json:"firstName" validate:"required,max=255"`),
		field.String("last_name").
			NotEmpty().
			MaxLen(255).
			StructTag(`json:"lastName" validate:"required,max=255"`),
		field.String("city").
			Optional().
			MaxLen(255).
			StructTag(`json:"city" validate:"omitempty,max=255"`),
		field.String("postal_code").
			Optional().
			MaxLen(10).
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
		index.Fields("code", "organization_id").
			Unique(),
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
		edge.From("tractor", Tractor.Type).
			Ref("primary_worker").
			StructTag(`json:"primary_tractor"`).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("secondary_tractor", Tractor.Type).
			Ref("secondary_worker").
			StructTag(`json:"secondary_tractor"`).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
