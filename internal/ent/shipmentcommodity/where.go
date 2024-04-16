// Code generated by entc, DO NOT EDIT.

package shipmentcommodity

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/emoss08/trenova/internal/ent/predicate"
	"github.com/google/uuid"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldLTE(FieldID, id))
}

// BusinessUnitID applies equality check predicate on the "business_unit_id" field. It's identical to BusinessUnitIDEQ.
func BusinessUnitID(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldBusinessUnitID, v))
}

// OrganizationID applies equality check predicate on the "organization_id" field. It's identical to OrganizationIDEQ.
func OrganizationID(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldOrganizationID, v))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldCreatedAt, v))
}

// UpdatedAt applies equality check predicate on the "updated_at" field. It's identical to UpdatedAtEQ.
func UpdatedAt(v time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldUpdatedAt, v))
}

// Version applies equality check predicate on the "version" field. It's identical to VersionEQ.
func Version(v int) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldVersion, v))
}

// ShipmentID applies equality check predicate on the "shipment_id" field. It's identical to ShipmentIDEQ.
func ShipmentID(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldShipmentID, v))
}

// CommodityID applies equality check predicate on the "commodity_id" field. It's identical to CommodityIDEQ.
func CommodityID(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldCommodityID, v))
}

// HazardousMaterialID applies equality check predicate on the "hazardous_material_id" field. It's identical to HazardousMaterialIDEQ.
func HazardousMaterialID(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldHazardousMaterialID, v))
}

// SubTotal applies equality check predicate on the "sub_total" field. It's identical to SubTotalEQ.
func SubTotal(v float64) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldSubTotal, v))
}

// PlacardNeeded applies equality check predicate on the "placard_needed" field. It's identical to PlacardNeededEQ.
func PlacardNeeded(v bool) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldPlacardNeeded, v))
}

// BusinessUnitIDEQ applies the EQ predicate on the "business_unit_id" field.
func BusinessUnitIDEQ(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldBusinessUnitID, v))
}

// BusinessUnitIDNEQ applies the NEQ predicate on the "business_unit_id" field.
func BusinessUnitIDNEQ(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNEQ(FieldBusinessUnitID, v))
}

// BusinessUnitIDIn applies the In predicate on the "business_unit_id" field.
func BusinessUnitIDIn(vs ...uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldIn(FieldBusinessUnitID, vs...))
}

// BusinessUnitIDNotIn applies the NotIn predicate on the "business_unit_id" field.
func BusinessUnitIDNotIn(vs ...uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNotIn(FieldBusinessUnitID, vs...))
}

// OrganizationIDEQ applies the EQ predicate on the "organization_id" field.
func OrganizationIDEQ(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldOrganizationID, v))
}

// OrganizationIDNEQ applies the NEQ predicate on the "organization_id" field.
func OrganizationIDNEQ(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNEQ(FieldOrganizationID, v))
}

// OrganizationIDIn applies the In predicate on the "organization_id" field.
func OrganizationIDIn(vs ...uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldIn(FieldOrganizationID, vs...))
}

// OrganizationIDNotIn applies the NotIn predicate on the "organization_id" field.
func OrganizationIDNotIn(vs ...uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNotIn(FieldOrganizationID, vs...))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldLTE(FieldCreatedAt, v))
}

// UpdatedAtEQ applies the EQ predicate on the "updated_at" field.
func UpdatedAtEQ(v time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldUpdatedAt, v))
}

// UpdatedAtNEQ applies the NEQ predicate on the "updated_at" field.
func UpdatedAtNEQ(v time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNEQ(FieldUpdatedAt, v))
}

// UpdatedAtIn applies the In predicate on the "updated_at" field.
func UpdatedAtIn(vs ...time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldIn(FieldUpdatedAt, vs...))
}

// UpdatedAtNotIn applies the NotIn predicate on the "updated_at" field.
func UpdatedAtNotIn(vs ...time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNotIn(FieldUpdatedAt, vs...))
}

// UpdatedAtGT applies the GT predicate on the "updated_at" field.
func UpdatedAtGT(v time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldGT(FieldUpdatedAt, v))
}

// UpdatedAtGTE applies the GTE predicate on the "updated_at" field.
func UpdatedAtGTE(v time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldGTE(FieldUpdatedAt, v))
}

// UpdatedAtLT applies the LT predicate on the "updated_at" field.
func UpdatedAtLT(v time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldLT(FieldUpdatedAt, v))
}

// UpdatedAtLTE applies the LTE predicate on the "updated_at" field.
func UpdatedAtLTE(v time.Time) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldLTE(FieldUpdatedAt, v))
}

// VersionEQ applies the EQ predicate on the "version" field.
func VersionEQ(v int) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldVersion, v))
}

// VersionNEQ applies the NEQ predicate on the "version" field.
func VersionNEQ(v int) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNEQ(FieldVersion, v))
}

// VersionIn applies the In predicate on the "version" field.
func VersionIn(vs ...int) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldIn(FieldVersion, vs...))
}

// VersionNotIn applies the NotIn predicate on the "version" field.
func VersionNotIn(vs ...int) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNotIn(FieldVersion, vs...))
}

// VersionGT applies the GT predicate on the "version" field.
func VersionGT(v int) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldGT(FieldVersion, v))
}

// VersionGTE applies the GTE predicate on the "version" field.
func VersionGTE(v int) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldGTE(FieldVersion, v))
}

// VersionLT applies the LT predicate on the "version" field.
func VersionLT(v int) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldLT(FieldVersion, v))
}

// VersionLTE applies the LTE predicate on the "version" field.
func VersionLTE(v int) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldLTE(FieldVersion, v))
}

// ShipmentIDEQ applies the EQ predicate on the "shipment_id" field.
func ShipmentIDEQ(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldShipmentID, v))
}

// ShipmentIDNEQ applies the NEQ predicate on the "shipment_id" field.
func ShipmentIDNEQ(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNEQ(FieldShipmentID, v))
}

// ShipmentIDIn applies the In predicate on the "shipment_id" field.
func ShipmentIDIn(vs ...uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldIn(FieldShipmentID, vs...))
}

// ShipmentIDNotIn applies the NotIn predicate on the "shipment_id" field.
func ShipmentIDNotIn(vs ...uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNotIn(FieldShipmentID, vs...))
}

// CommodityIDEQ applies the EQ predicate on the "commodity_id" field.
func CommodityIDEQ(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldCommodityID, v))
}

// CommodityIDNEQ applies the NEQ predicate on the "commodity_id" field.
func CommodityIDNEQ(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNEQ(FieldCommodityID, v))
}

// CommodityIDIn applies the In predicate on the "commodity_id" field.
func CommodityIDIn(vs ...uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldIn(FieldCommodityID, vs...))
}

// CommodityIDNotIn applies the NotIn predicate on the "commodity_id" field.
func CommodityIDNotIn(vs ...uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNotIn(FieldCommodityID, vs...))
}

// CommodityIDGT applies the GT predicate on the "commodity_id" field.
func CommodityIDGT(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldGT(FieldCommodityID, v))
}

// CommodityIDGTE applies the GTE predicate on the "commodity_id" field.
func CommodityIDGTE(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldGTE(FieldCommodityID, v))
}

// CommodityIDLT applies the LT predicate on the "commodity_id" field.
func CommodityIDLT(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldLT(FieldCommodityID, v))
}

// CommodityIDLTE applies the LTE predicate on the "commodity_id" field.
func CommodityIDLTE(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldLTE(FieldCommodityID, v))
}

// HazardousMaterialIDEQ applies the EQ predicate on the "hazardous_material_id" field.
func HazardousMaterialIDEQ(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldHazardousMaterialID, v))
}

// HazardousMaterialIDNEQ applies the NEQ predicate on the "hazardous_material_id" field.
func HazardousMaterialIDNEQ(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNEQ(FieldHazardousMaterialID, v))
}

// HazardousMaterialIDIn applies the In predicate on the "hazardous_material_id" field.
func HazardousMaterialIDIn(vs ...uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldIn(FieldHazardousMaterialID, vs...))
}

// HazardousMaterialIDNotIn applies the NotIn predicate on the "hazardous_material_id" field.
func HazardousMaterialIDNotIn(vs ...uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNotIn(FieldHazardousMaterialID, vs...))
}

// HazardousMaterialIDGT applies the GT predicate on the "hazardous_material_id" field.
func HazardousMaterialIDGT(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldGT(FieldHazardousMaterialID, v))
}

// HazardousMaterialIDGTE applies the GTE predicate on the "hazardous_material_id" field.
func HazardousMaterialIDGTE(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldGTE(FieldHazardousMaterialID, v))
}

// HazardousMaterialIDLT applies the LT predicate on the "hazardous_material_id" field.
func HazardousMaterialIDLT(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldLT(FieldHazardousMaterialID, v))
}

// HazardousMaterialIDLTE applies the LTE predicate on the "hazardous_material_id" field.
func HazardousMaterialIDLTE(v uuid.UUID) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldLTE(FieldHazardousMaterialID, v))
}

// SubTotalEQ applies the EQ predicate on the "sub_total" field.
func SubTotalEQ(v float64) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldSubTotal, v))
}

// SubTotalNEQ applies the NEQ predicate on the "sub_total" field.
func SubTotalNEQ(v float64) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNEQ(FieldSubTotal, v))
}

// SubTotalIn applies the In predicate on the "sub_total" field.
func SubTotalIn(vs ...float64) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldIn(FieldSubTotal, vs...))
}

// SubTotalNotIn applies the NotIn predicate on the "sub_total" field.
func SubTotalNotIn(vs ...float64) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNotIn(FieldSubTotal, vs...))
}

// SubTotalGT applies the GT predicate on the "sub_total" field.
func SubTotalGT(v float64) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldGT(FieldSubTotal, v))
}

// SubTotalGTE applies the GTE predicate on the "sub_total" field.
func SubTotalGTE(v float64) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldGTE(FieldSubTotal, v))
}

// SubTotalLT applies the LT predicate on the "sub_total" field.
func SubTotalLT(v float64) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldLT(FieldSubTotal, v))
}

// SubTotalLTE applies the LTE predicate on the "sub_total" field.
func SubTotalLTE(v float64) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldLTE(FieldSubTotal, v))
}

// PlacardNeededEQ applies the EQ predicate on the "placard_needed" field.
func PlacardNeededEQ(v bool) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldEQ(FieldPlacardNeeded, v))
}

// PlacardNeededNEQ applies the NEQ predicate on the "placard_needed" field.
func PlacardNeededNEQ(v bool) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.FieldNEQ(FieldPlacardNeeded, v))
}

// HasBusinessUnit applies the HasEdge predicate on the "business_unit" edge.
func HasBusinessUnit() predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, BusinessUnitTable, BusinessUnitColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasBusinessUnitWith applies the HasEdge predicate on the "business_unit" edge with a given conditions (other predicates).
func HasBusinessUnitWith(preds ...predicate.BusinessUnit) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(func(s *sql.Selector) {
		step := newBusinessUnitStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasOrganization applies the HasEdge predicate on the "organization" edge.
func HasOrganization() predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, OrganizationTable, OrganizationColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasOrganizationWith applies the HasEdge predicate on the "organization" edge with a given conditions (other predicates).
func HasOrganizationWith(preds ...predicate.Organization) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(func(s *sql.Selector) {
		step := newOrganizationStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasShipment applies the HasEdge predicate on the "shipment" edge.
func HasShipment() predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, ShipmentTable, ShipmentColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasShipmentWith applies the HasEdge predicate on the "shipment" edge with a given conditions (other predicates).
func HasShipmentWith(preds ...predicate.Shipment) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(func(s *sql.Selector) {
		step := newShipmentStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.ShipmentCommodity) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.ShipmentCommodity) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.ShipmentCommodity) predicate.ShipmentCommodity {
	return predicate.ShipmentCommodity(sql.NotPredicates(p))
}