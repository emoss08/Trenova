package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrganizationFeatureFlag holds the schema definition for the OrganizationFeatureFlag entity.
type OrganizationFeatureFlag struct {
	ent.Schema
}

// Fields of the OrganizationFeatureFlag.
func (OrganizationFeatureFlag) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.UUID{}).
			StructTag(`json:"organizationId"`).
			Immutable(),
		field.UUID("feature_flag_id", uuid.UUID{}).
			StructTag(`json:featureFlagId`).
			Immutable(),
		field.Bool("is_enabled").
			Default(true).
			StructTag(`json:"isEnabled" validate:"omitempty"`),
	}
}

// Mixin of the OrganizationFeatureFlag.
func (OrganizationFeatureFlag) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

// Indexes of the OrganizationFeatureFlag.
func (OrganizationFeatureFlag) Indexes() []ent.Index {
	return []ent.Index{
		// Ensure only 1 feature flag is created per organization.
		index.Fields("organization_id", "feature_flag_id").
			Unique(),
	}
}

// Edges of the OrganizationFeatureFlag.
func (OrganizationFeatureFlag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("feature_flag", FeatureFlag.Type).
			Field("feature_flag_id").
			StructTag(`json:"featureFlag"`).
			Unique().
			Required().
			Immutable(),
		edge.To("organization", Organization.Type).
			Field("organization_id").
			StructTag(`json:"organization"`).
			Unique().
			Required().
			Immutable(),
	}
}
