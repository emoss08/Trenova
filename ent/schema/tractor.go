package schema

import (
	"regexp"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Tractor holds the schema definition for the Tractor entity.
type Tractor struct {
	ent.Schema
}

// Fields of the Tractor.
func (Tractor) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			NotEmpty().
			MaxLen(50).
			StructTag(`json:"code" validate:"required,max=50"`),
		field.Enum("status").
			Values("Available", "OutOfService", "AtMaintenance", "Sold", "Lost").
			Default("Available").
			StructTag(`json:"status" validate:"required,oneof=Available OutOfService AtMaintenance Sold Lost"`),
		field.UUID("equipment_type_id", uuid.UUID{}).
			Optional().
			Unique().
			StructTag(`json:"equipmentTypeId" validate:"required,uuid"`),
		field.String("license_plate_number").
			MaxLen(50).
			Optional().
			StructTag(`json:"licensePlateNumber" validate:"omitempty,max=50"`),
		field.String("vin").
			Match(regexp.MustCompile("^[0-9A-HJ-NPR-Z]{17}$")).
			MaxLen(17).
			Optional().
			StructTag(`json:"vin" validate:"omitempty,alphanum,len=17"`),
		field.UUID("equipment_manufacturer_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"equipmentManufacturerId" validate:"omitempty,uuid"`),
		field.String("model").
			MaxLen(50).
			Optional().
			StructTag(`json:"model" validate:"omitempty,max=50"`),
		field.Int("year").
			Positive().
			Optional().
			StructTag(`json:"year" validate:"omitempty,gt=0"`),
		field.UUID("state_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"stateId" validate:"omitempty,uuid"`),
		field.Bool("leased").
			Default(false).
			StructTag(`json:"leased" validate:"omitempty"`),
		field.Time("leased_date").
			Optional().
			Nillable().
			StructTag(`json:"leasedDate" validate:"omitempty"`),
		field.UUID("primary_worker_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"primaryWorkerId" validate:"omitempty,uuid"`),
		field.UUID("secondary_worker_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"secondaryWorkerId" validate:"omitempty,uuid"`),
	}
}

// Mixin of the Tractor.
func (Tractor) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Indexes of the Tractor.
func (Tractor) Indexes() []ent.Index {
	return []ent.Index{
		// Ensure the code is unique for the organization.
		index.Fields("code", "organization_id").
			Unique(),
	}
}

// Edges of the Tractor.
func (Tractor) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("equipment_type", EquipmentType.Type).
			Field("equipment_type_id").
			StructTag(`json:"equipmentType"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("equipment_manufacturer", EquipmentManufactuer.Type).
			Field("equipment_manufacturer_id").
			StructTag(`json:"equipmentManufacturer"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("state", UsState.Type).
			Field("state_id").
			StructTag(`json:"state"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("primary_worker", Worker.Type).
			Field("primary_worker_id").
			StructTag(`json:"primaryWorker"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("secondary_worker", Worker.Type).
			Field("secondary_worker_id").
			StructTag(`json:"secondaryWorker"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
	}
}
