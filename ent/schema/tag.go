package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Tag holds the schema definition for the Tag entity.
type Tag struct {
	ent.Schema
}

// Fields of the Tag.
func (Tag) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(50),
		field.Text("description").
			Nillable().
			Optional(),
	}
}

// Edges of the Tag.
func (Tag) Edges() []ent.Edge {
	return nil
}

// Mixin of the Tag.
func (Tag) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Indexes of the Tag.
func (Tag) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name", "organization_id").
			Unique(),
	}
}
