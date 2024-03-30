package schema

import "entgo.io/ent"

// WorkerComment holds the schema definition for the WorkerComment entity.
type WorkerComment struct {
	ent.Schema
}

// Fields of the WorkerComment.
func (WorkerComment) Fields() []ent.Field {
	return nil
}

// Mixin of the WorkerComment.
func (WorkerComment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the WorkerComment.
func (WorkerComment) Edges() []ent.Edge {
	return nil
}
