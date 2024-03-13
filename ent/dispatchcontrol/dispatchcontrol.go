// Code generated by ent, DO NOT EDIT.

package dispatchcontrol

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the dispatchcontrol type in the database.
	Label = "dispatch_control"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldCreatedAt holds the string denoting the created_at field in the database.
	FieldCreatedAt = "created_at"
	// FieldUpdatedAt holds the string denoting the updated_at field in the database.
	FieldUpdatedAt = "updated_at"
	// FieldRecordServiceIncident holds the string denoting the record_service_incident field in the database.
	FieldRecordServiceIncident = "record_service_incident"
	// FieldDeadheadTarget holds the string denoting the deadhead_target field in the database.
	FieldDeadheadTarget = "deadhead_target"
	// FieldMaxShipmentWeightLimit holds the string denoting the max_shipment_weight_limit field in the database.
	FieldMaxShipmentWeightLimit = "max_shipment_weight_limit"
	// FieldGracePeriod holds the string denoting the grace_period field in the database.
	FieldGracePeriod = "grace_period"
	// FieldEnforceWorkerAssign holds the string denoting the enforce_worker_assign field in the database.
	FieldEnforceWorkerAssign = "enforce_worker_assign"
	// FieldTrailerContinuity holds the string denoting the trailer_continuity field in the database.
	FieldTrailerContinuity = "trailer_continuity"
	// FieldDupeTrailerCheck holds the string denoting the dupe_trailer_check field in the database.
	FieldDupeTrailerCheck = "dupe_trailer_check"
	// FieldMaintenanceCompliance holds the string denoting the maintenance_compliance field in the database.
	FieldMaintenanceCompliance = "maintenance_compliance"
	// FieldRegulatoryCheck holds the string denoting the regulatory_check field in the database.
	FieldRegulatoryCheck = "regulatory_check"
	// FieldPrevShipmentOnHold holds the string denoting the prev_shipment_on_hold field in the database.
	FieldPrevShipmentOnHold = "prev_shipment_on_hold"
	// FieldWorkerTimeAwayRestriction holds the string denoting the worker_time_away_restriction field in the database.
	FieldWorkerTimeAwayRestriction = "worker_time_away_restriction"
	// FieldTractorWorkerFleetConstraint holds the string denoting the tractor_worker_fleet_constraint field in the database.
	FieldTractorWorkerFleetConstraint = "tractor_worker_fleet_constraint"
	// EdgeOrganization holds the string denoting the organization edge name in mutations.
	EdgeOrganization = "organization"
	// EdgeBusinessUnit holds the string denoting the business_unit edge name in mutations.
	EdgeBusinessUnit = "business_unit"
	// Table holds the table name of the dispatchcontrol in the database.
	Table = "dispatch_controls"
	// OrganizationTable is the table that holds the organization relation/edge.
	OrganizationTable = "dispatch_controls"
	// OrganizationInverseTable is the table name for the Organization entity.
	// It exists in this package in order to avoid circular dependency with the "organization" package.
	OrganizationInverseTable = "organizations"
	// OrganizationColumn is the table column denoting the organization relation/edge.
	OrganizationColumn = "organization_id"
	// BusinessUnitTable is the table that holds the business_unit relation/edge.
	BusinessUnitTable = "dispatch_controls"
	// BusinessUnitInverseTable is the table name for the BusinessUnit entity.
	// It exists in this package in order to avoid circular dependency with the "businessunit" package.
	BusinessUnitInverseTable = "business_units"
	// BusinessUnitColumn is the table column denoting the business_unit relation/edge.
	BusinessUnitColumn = "business_unit_id"
)

// Columns holds all SQL columns for dispatchcontrol fields.
var Columns = []string{
	FieldID,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldRecordServiceIncident,
	FieldDeadheadTarget,
	FieldMaxShipmentWeightLimit,
	FieldGracePeriod,
	FieldEnforceWorkerAssign,
	FieldTrailerContinuity,
	FieldDupeTrailerCheck,
	FieldMaintenanceCompliance,
	FieldRegulatoryCheck,
	FieldPrevShipmentOnHold,
	FieldWorkerTimeAwayRestriction,
	FieldTractorWorkerFleetConstraint,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "dispatch_controls"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"business_unit_id",
	"organization_id",
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	for i := range ForeignKeys {
		if column == ForeignKeys[i] {
			return true
		}
	}
	return false
}

var (
	// DefaultCreatedAt holds the default value on creation for the "created_at" field.
	DefaultCreatedAt time.Time
	// DefaultUpdatedAt holds the default value on creation for the "updated_at" field.
	DefaultUpdatedAt time.Time
	// UpdateDefaultUpdatedAt holds the default value on update for the "updated_at" field.
	UpdateDefaultUpdatedAt func() time.Time
	// DefaultDeadheadTarget holds the default value on creation for the "deadhead_target" field.
	DefaultDeadheadTarget float64
	// DefaultMaxShipmentWeightLimit holds the default value on creation for the "max_shipment_weight_limit" field.
	DefaultMaxShipmentWeightLimit int
	// MaxShipmentWeightLimitValidator is a validator for the "max_shipment_weight_limit" field. It is called by the builders before save.
	MaxShipmentWeightLimitValidator func(int) error
	// DefaultGracePeriod holds the default value on creation for the "grace_period" field.
	DefaultGracePeriod uint8
	// DefaultEnforceWorkerAssign holds the default value on creation for the "enforce_worker_assign" field.
	DefaultEnforceWorkerAssign bool
	// DefaultTrailerContinuity holds the default value on creation for the "trailer_continuity" field.
	DefaultTrailerContinuity bool
	// DefaultDupeTrailerCheck holds the default value on creation for the "dupe_trailer_check" field.
	DefaultDupeTrailerCheck bool
	// DefaultMaintenanceCompliance holds the default value on creation for the "maintenance_compliance" field.
	DefaultMaintenanceCompliance bool
	// DefaultRegulatoryCheck holds the default value on creation for the "regulatory_check" field.
	DefaultRegulatoryCheck bool
	// DefaultPrevShipmentOnHold holds the default value on creation for the "prev_shipment_on_hold" field.
	DefaultPrevShipmentOnHold bool
	// DefaultWorkerTimeAwayRestriction holds the default value on creation for the "worker_time_away_restriction" field.
	DefaultWorkerTimeAwayRestriction bool
	// DefaultTractorWorkerFleetConstraint holds the default value on creation for the "tractor_worker_fleet_constraint" field.
	DefaultTractorWorkerFleetConstraint bool
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() uuid.UUID
)

// RecordServiceIncident defines the type for the "record_service_incident" enum field.
type RecordServiceIncident string

// RecordServiceIncidentNever is the default value of the RecordServiceIncident enum.
const DefaultRecordServiceIncident = RecordServiceIncidentNever

// RecordServiceIncident values.
const (
	RecordServiceIncidentNever             RecordServiceIncident = "Never"
	RecordServiceIncidentPickup            RecordServiceIncident = "Pickup"
	RecordServiceIncidentDelivery          RecordServiceIncident = "Delivery"
	RecordServiceIncidentPickupAndDelivery RecordServiceIncident = "PickupAndDelivery"
	RecordServiceIncidentAllExceptShipper  RecordServiceIncident = "AllExceptShipper"
)

func (rsi RecordServiceIncident) String() string {
	return string(rsi)
}

// RecordServiceIncidentValidator is a validator for the "record_service_incident" field enum values. It is called by the builders before save.
func RecordServiceIncidentValidator(rsi RecordServiceIncident) error {
	switch rsi {
	case RecordServiceIncidentNever, RecordServiceIncidentPickup, RecordServiceIncidentDelivery, RecordServiceIncidentPickupAndDelivery, RecordServiceIncidentAllExceptShipper:
		return nil
	default:
		return fmt.Errorf("dispatchcontrol: invalid enum value for record_service_incident field: %q", rsi)
	}
}

// OrderOption defines the ordering options for the DispatchControl queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByCreatedAt orders the results by the created_at field.
func ByCreatedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCreatedAt, opts...).ToFunc()
}

// ByUpdatedAt orders the results by the updated_at field.
func ByUpdatedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldUpdatedAt, opts...).ToFunc()
}

// ByRecordServiceIncident orders the results by the record_service_incident field.
func ByRecordServiceIncident(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldRecordServiceIncident, opts...).ToFunc()
}

// ByDeadheadTarget orders the results by the deadhead_target field.
func ByDeadheadTarget(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDeadheadTarget, opts...).ToFunc()
}

// ByMaxShipmentWeightLimit orders the results by the max_shipment_weight_limit field.
func ByMaxShipmentWeightLimit(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldMaxShipmentWeightLimit, opts...).ToFunc()
}

// ByGracePeriod orders the results by the grace_period field.
func ByGracePeriod(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldGracePeriod, opts...).ToFunc()
}

// ByEnforceWorkerAssign orders the results by the enforce_worker_assign field.
func ByEnforceWorkerAssign(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldEnforceWorkerAssign, opts...).ToFunc()
}

// ByTrailerContinuity orders the results by the trailer_continuity field.
func ByTrailerContinuity(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTrailerContinuity, opts...).ToFunc()
}

// ByDupeTrailerCheck orders the results by the dupe_trailer_check field.
func ByDupeTrailerCheck(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDupeTrailerCheck, opts...).ToFunc()
}

// ByMaintenanceCompliance orders the results by the maintenance_compliance field.
func ByMaintenanceCompliance(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldMaintenanceCompliance, opts...).ToFunc()
}

// ByRegulatoryCheck orders the results by the regulatory_check field.
func ByRegulatoryCheck(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldRegulatoryCheck, opts...).ToFunc()
}

// ByPrevShipmentOnHold orders the results by the prev_shipment_on_hold field.
func ByPrevShipmentOnHold(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPrevShipmentOnHold, opts...).ToFunc()
}

// ByWorkerTimeAwayRestriction orders the results by the worker_time_away_restriction field.
func ByWorkerTimeAwayRestriction(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldWorkerTimeAwayRestriction, opts...).ToFunc()
}

// ByTractorWorkerFleetConstraint orders the results by the tractor_worker_fleet_constraint field.
func ByTractorWorkerFleetConstraint(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTractorWorkerFleetConstraint, opts...).ToFunc()
}

// ByOrganizationField orders the results by organization field.
func ByOrganizationField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newOrganizationStep(), sql.OrderByField(field, opts...))
	}
}

// ByBusinessUnitField orders the results by business_unit field.
func ByBusinessUnitField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBusinessUnitStep(), sql.OrderByField(field, opts...))
	}
}
func newOrganizationStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(OrganizationInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2O, true, OrganizationTable, OrganizationColumn),
	)
}
func newBusinessUnitStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BusinessUnitInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, BusinessUnitTable, BusinessUnitColumn),
	)
}

// MarshalGQL implements graphql.Marshaler interface.
func (e RecordServiceIncident) MarshalGQL(w io.Writer) {
	io.WriteString(w, strconv.Quote(e.String()))
}

// UnmarshalGQL implements graphql.Unmarshaler interface.
func (e *RecordServiceIncident) UnmarshalGQL(val interface{}) error {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("enum %T must be a string", val)
	}
	*e = RecordServiceIncident(str)
	if err := RecordServiceIncidentValidator(*e); err != nil {
		return fmt.Errorf("%s is not a valid RecordServiceIncident", str)
	}
	return nil
}