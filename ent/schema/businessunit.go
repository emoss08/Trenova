package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
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
			Default(time.Now()),
		field.Time("updated_at").
			Default(time.Now()).
			UpdateDefault(time.Now),
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
			StructTag(`json:"businessUnitId"`),
		field.UUID("organization_id", uuid.UUID{}).
			StructTag(`json:"organizationId"`),
		field.Time("created_at").
			Immutable().
			Default(time.Now()),
		field.Time("updated_at").
			Default(time.Now()).
			UpdateDefault(time.Now),
	}
}

func (BaseMixin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("business_unit", BusinessUnit.Type).
			Field("business_unit_id").
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("organization", Organization.Type).
			Field("organization_id").
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
	}
}

// BusinessUnit holds the schema definition for the BusinessUnit entity.
type BusinessUnit struct {
	ent.Schema
}

// Mixin of the BusinessUnit.
func (BusinessUnit) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

// Fields of the BusinessUnit.
func (BusinessUnit) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A"),
		field.String("name").
			MaxLen(255).
			NotEmpty(),
		field.String("entity_key").
			MaxLen(10).
			NotEmpty(),
		field.String("phone_number").
			MaxLen(15).
			StructTag(`json:"phoneNumber"`),
		field.String("address").
			Optional(),
		field.String("city").
			MaxLen(255).
			Optional(),
		field.String("state").
			MaxLen(2).
			Optional(),
		field.String("country").
			MaxLen(2).
			Optional(),
		field.String("postal_code").
			MaxLen(10).
			Optional().
			StructTag(`json:"postalCode"`),
		field.String("tax_id").
			MaxLen(20).
			Optional().
			StructTag(`json:"taxId"`),
		field.String("subscription_plan").
			Optional().
			StructTag(`json:"subscriptionPlan"`),
		field.String("description").
			Optional(),
		field.String("legal_name").
			Optional().
			StructTag(`json:"legalName"`),
		field.String("contact_name").
			Optional().
			StructTag(`json:"contactName"`),
		field.String("contact_email").
			Optional().
			StructTag(`json:"contactEmail"`),
		field.Time("paid_until").
			Optional().
			StructTag(`json:"-"`),
		field.JSON("settings", map[string]interface{}{}).
			Optional(),
		field.Bool("free_trial").
			Default(false).
			StructTag(`json:"freeTrial"`),
		field.UUID("parent_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"parentId"`),
	}
}

// Edges of the BusinessUnit.
func (BusinessUnit) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("next", BusinessUnit.Type).
			Unique().
			From("prev").
			Unique().
			Field("parent_id").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"parent_id"`),
		edge.To("organizations", Organization.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (BusinessUnit) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").
			Unique().
			Annotations(
				entsql.DefaultExpr("lower(name)"),
			),
		index.Fields("entity_key").
			Unique(),
	}
}
