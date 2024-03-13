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
			MaxLen(255),
		field.String("state").
			MaxLen(2),
		field.String("country").
			MaxLen(2),
		field.String("postal_code").
			MaxLen(10).
			StructTag(`json:"postalCode"`),
		field.String("tax_id").
			MaxLen(20),
		field.String("subscription_plan").
			NotEmpty().
			StructTag(`json:"subscriptionPlan"`),
		field.String("description").
			Optional(),
		field.String("legal_name").
			NotEmpty().
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
			StructTag(`json:"parentId"`),
	}
}

// Edges of the BusinessUnit.
func (BusinessUnit) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("parent", BusinessUnit.Type).
			Field("parent_id").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique().
			StructTag(`json:"parent_id"`),
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
