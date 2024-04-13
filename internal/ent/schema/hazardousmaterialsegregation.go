package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// HazardousMaterialSegregation holds the schema definition for the HazardousMaterialSegregation entity.
type HazardousMaterialSegregation struct {
	ent.Schema
}

// Fields of the HazardousMaterialSegregation.
func (HazardousMaterialSegregation) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("class_a").
			Values("HazardClass1And1",
				"HazardClass1And2",
				"HazardClass1And3",
				"HazardClass1And4",
				"HazardClass1And5",
				"HazardClass1And6",
				"HazardClass2And1",
				"HazardClass2And2",
				"HazardClass2And3",
				"HazardClass3",
				"HazardClass4And1",
				"HazardClass4And2",
				"HazardClass4And3",
				"HazardClass5And1",
				"HazardClass5And2",
				"HazardClass6And1",
				"HazardClass6And2",
				"HazardClass7",
				"HazardClass8",
				"HazardClass9").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(16)",
				dialect.SQLite:   "VARCHAR(16)",
			}).
			Default("HazardClass1And1").
			StructTag(`json:"classA" validate:"required"`),
		field.Enum("class_b").
			Values("HazardClass1And1",
				"HazardClass1And2",
				"HazardClass1And3",
				"HazardClass1And4",
				"HazardClass1And5",
				"HazardClass1And6",
				"HazardClass2And1",
				"HazardClass2And2",
				"HazardClass2And3",
				"HazardClass3",
				"HazardClass4And1",
				"HazardClass4And2",
				"HazardClass4And3",
				"HazardClass5And1",
				"HazardClass5And2",
				"HazardClass6And1",
				"HazardClass6And2",
				"HazardClass7",
				"HazardClass8",
				"HazardClass9").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(16)",
				dialect.SQLite:   "VARCHAR(16)",
			}).
			Default("HazardClass1And1").
			StructTag(`json:"classB" validate:"required"`),
		field.Enum("segregation_type").
			Values("NotAllowed", "AllowedWithConditions").
			Default("NotAllowed").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(21)",
				dialect.SQLite:   "VARCHAR(21)",
			}).
			StructTag(`json:"segregationType" validate:"required"`),
	}
}

// Indexes of the HazardousMaterialSegregation.
func (HazardousMaterialSegregation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("class_a", "class_b", "organization_id").
			Unique(),
	}
}

// Mixin of the HazardousMaterialSegregation.
func (HazardousMaterialSegregation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the HazardousMaterialSegregation.
func (HazardousMaterialSegregation) Edges() []ent.Edge {
	return nil
}
