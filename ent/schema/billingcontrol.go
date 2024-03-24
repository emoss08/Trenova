package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// BillingControl holds the schema definition for the BillingControl entity.
type BillingControl struct {
	ent.Schema
}

// Fields of the BillingControl.
func (BillingControl) Fields() []ent.Field {
	return []ent.Field{
		field.Bool("remove_billing_history").
			Default(false).
			StructTag(`json:"removeBillingHistory" validate:"omitempty"`),
		field.Bool("auto_bill_shipment").
			Default(false).
			StructTag(`json:"autoBillShipment" validate:"omitempty"`),
		field.Bool("auto_mark_ready_to_bill").
			Default(false).
			StructTag(`json:"autoMarkReadyToBill" validate:"omitempty"`),
		field.Bool("validate_customer_rates").
			Default(false).
			StructTag(`json:"validateCustomerRates" validate:"omitempty"`),
		field.Enum("auto_bill_criteria").
			Values("Delivered", "TransferredToBilling", "MarkedReadyToBill").
			Default("MarkedReadyToBill").
			StructTag(`json:"autoBillCriteria" validate:"omitempty"`),
		field.Enum("shipment_transfer_criteria").
			Values("ReadyAndCompleted", "Completed", "ReadyToBill").
			Default("ReadyToBill").
			StructTag(`json:"shipmentTransferCriteria" validate:"omitempty"`),
		field.Bool("enforce_customer_billing").
			Default(false).
			StructTag(`json:"enforceCustomerBilling" validate:"omitempty"`),
	}
}

// Mixin for the BillingControl.
func (BillingControl) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

// Annotations of the BillingControl.
func (BillingControl) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.QueryField(),
		entgql.Mutations(entgql.MutationCreate()),
	}
}

// Edges of the BillingControl.
func (BillingControl) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("billing_control").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
		edge.To("business_unit", BusinessUnit.Type).
			StorageKey(edge.Column("business_unit_id")).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
	}
}
