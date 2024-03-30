package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// WorkerContact holds the schema definition for the WorkerContact entity.
type WorkerContact struct {
	ent.Schema
}

// Fields of the WorkerContact.
func (WorkerContact) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("worker_id", uuid.UUID{}).
			StructTag(`json:"workerId" validate:"required"`),
		field.String("name").
			NotEmpty().
			StructTag(`json:"name" validate:"required"`),
		field.String("email").
			NotEmpty().
			StructTag(`json:"email" validate:"required"`),
		field.String("phone").
			NotEmpty().
			StructTag(`json:"phone" validate:"required"`),
		field.String("relationship").
			Optional().
			StructTag(`json:"relationship" validate:"omitempty"`),
		field.Bool("is_primary").
			Default(false).
			StructTag(`json:"isPrimary" validate:"omitempty"`),
	}
}

// Mixin of the WorkerContact.
func (WorkerContact) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the WorkerContact.
func (WorkerContact) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("worker", Worker.Type).
			Ref("worker_contacts").
			Field("worker_id").
			StructTag(`json:"worker"`).
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
	}
}
