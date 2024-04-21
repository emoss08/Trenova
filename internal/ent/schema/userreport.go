package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// UserReport holds the schema definition for the UserReport entity.
type UserReport struct {
	ent.Schema
}

// Fields of the UserReport.
func (UserReport) Fields() []ent.Field {
	return []ent.Field{
		field.String("report_url").
			NotEmpty().
			StructTag(`json:"reportUrl" validate:"required"`),
		field.UUID("user_id", uuid.UUID{}).
			Immutable().
			StructTag(`json:"userId" validate:"required,uuid"`),
	}
}

// Mixin of the UserReport.
func (UserReport) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the UserReport.
func (UserReport) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("reports").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Field("user_id").
			Immutable().
			Required().
			Unique(),
	}
}
