package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// WorkerComment holds the schema definition for the WorkerComment entity.
type WorkerComment struct {
	ent.Schema
}

// Fields of the WorkerComment.
func (WorkerComment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("worker_id", uuid.UUID{}).
			StructTag(`json:"workerId" validate:"required"`),
		field.UUID("comment_type_id", uuid.UUID{}).
			StructTag(`json:"commentTypeId" validate:"required"`),
		field.Text("comment").
			NotEmpty().
			StructTag(`json:"comment" validate:"omitempty"`),
		field.UUID("entered_by", uuid.UUID{}).
			StructTag(`json:"enteredBy" validate:"required"`),
	}
}

// Mixin of the WorkerComment.
func (WorkerComment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the WorkerComment.
func (WorkerComment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("worker", Worker.Type).
			Ref("worker_comments").
			Field("worker_id").
			StructTag(`json:"worker"`).
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("comment_type", CommentType.Type).
			Field("comment_type_id").
			StructTag(`json:"commentType"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
	}
}
