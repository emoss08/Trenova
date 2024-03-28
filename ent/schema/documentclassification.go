package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// DocumentClassification holds the schema definition for the DocumentClassification entity.
type DocumentClassification struct {
	ent.Schema
}

// Fields of the DocumentClassification.
func (DocumentClassification) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(10).
			StructTag(`json:"name" validate:"required,max=10"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
	}
}

// Mixin of the DocumentClassification.
func (DocumentClassification) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the DocumentClassification.
func (DocumentClassification) Edges() []ent.Edge {
	return nil
}
