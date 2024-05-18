package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/google/uuid"
)

// DefaultMixin implements the ent.Mixin for sharing time fields with package schemas.
type DefaultMixin struct {
	mixin.Schema
}

// Fields of the DefaultMixin.
func (DefaultMixin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			Comment("The time that this entity was created.").
			StructTag(`json:"createdAt" validate:"omitempty"`),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("The last time that this entity was updated.").
			StructTag(`json:"updatedAt" validate:"omitempty"`),
	}
}

// BaseMixin implements the ent.Mixin for sharing time fields with package schemas.
//
// This mixin is used to add the common fields to all entities.
type BaseMixin struct {
	mixin.Schema
}

// Fields of the BaseMixin.
func (BaseMixin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.UUID("business_unit_id", uuid.UUID{}).
			StructTag(`json:"businessUnitId"`).
			Immutable(),
		field.UUID("organization_id", uuid.UUID{}).
			StructTag(`json:"organizationId"`),
		field.Time("created_at").
			Immutable().
			Comment("The time that this entity was created.").
			Default(time.Now).
			StructTag(`json:"createdAt" validate:"omitempty"`),
		field.Time("updated_at").
			Default(time.Now).
			Comment("The last time that this entity was updated.").
			UpdateDefault(time.Now).
			StructTag(`json:"updatedAt" validate:"omitempty"`),
		field.Int("version").
			Default(1).
			Comment("The current version of this entity.").
			StructTag(`json:"version" validate:"omitempty"`),
	}
}

// Edges of the BaseMixin.
func (BaseMixin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("business_unit", BusinessUnit.Type).
			Field("business_unit_id").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique().
			Required().
			Immutable(),
		edge.To("organization", Organization.Type).
			Field("organization_id").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique().
			Required(),
	}
}

func (BaseMixin) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
	}
}