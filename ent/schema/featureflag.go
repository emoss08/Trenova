package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// FeatureFlag holds the schema definition for the FeatureFlag entity.
type FeatureFlag struct {
	ent.Schema
}

// Fields of the FeatureFlag.
func (FeatureFlag) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			StructTag(`json:"name" validate:"required"`),
		field.String("code").
			NotEmpty().
			Unique().
			MaxLen(30).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(30)",
				dialect.SQLite:   "VARCHAR(30)",
			}).
			StructTag(`json:"code" validate:"required"`),
		field.Bool("beta").
			Default(false).
			StructTag(`json:"beta" validate:"omitempty"`),
		field.Text("description").
			NotEmpty().
			StructTag(`json:"description" validate:"required"`),
		field.String("preview_picture_url").
			Optional().
			StructTag(`json:"previewPictureUrl" validate:"omitempty"`),
	}
}

// Mixin of the FeatureFlag.
func (FeatureFlag) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

// Annotations for the FeatureFlag.
func (FeatureFlag) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
		schema.Comment("Internal table for storing the feature flags available for Trenova"),
	}
}

// Edges of the FeatureFlag.
func (FeatureFlag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization_feature_flag", OrganizationFeatureFlag.Type).
			Ref("feature_flag"),
	}
}
