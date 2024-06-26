// Code generated by entc, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/customer"
	"github.com/emoss08/trenova/internal/ent/deliveryslot"
	"github.com/emoss08/trenova/internal/ent/location"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/google/uuid"
)

// DeliverySlot is the model entity for the DeliverySlot schema.
type DeliverySlot struct {
	config `json:"-" validate:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// BusinessUnitID holds the value of the "business_unit_id" field.
	BusinessUnitID uuid.UUID `json:"businessUnitId"`
	// OrganizationID holds the value of the "organization_id" field.
	OrganizationID uuid.UUID `json:"organizationId"`
	// The time that this entity was created.
	CreatedAt time.Time `json:"createdAt" validate:"omitempty"`
	// The last time that this entity was updated.
	UpdatedAt time.Time `json:"updatedAt" validate:"omitempty"`
	// The current version of this entity.
	Version int `json:"version" validate:"omitempty"`
	// CustomerID holds the value of the "customer_id" field.
	CustomerID uuid.UUID `json:"customerId" validate:"required"`
	// LocationID holds the value of the "location_id" field.
	LocationID uuid.UUID `json:"locationId" validate:"required"`
	// DayOfWeek holds the value of the "day_of_week" field.
	DayOfWeek deliveryslot.DayOfWeek `json:"dayOfWeek" validate:"required,oneof=SUNDAY MONDAY TUESDAY WEDNESDAY THURSDAY FRIDAY SATURDAY"`
	// StartTime holds the value of the "start_time" field.
	StartTime *types.TimeOnly `json:"startTime" validate:"required"`
	// EndTime holds the value of the "end_time" field.
	EndTime *types.TimeOnly `json:"endTime" validate:"required"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the DeliverySlotQuery when eager-loading is set.
	Edges        DeliverySlotEdges `json:"edges"`
	selectValues sql.SelectValues
}

// DeliverySlotEdges holds the relations/edges for other nodes in the graph.
type DeliverySlotEdges struct {
	// BusinessUnit holds the value of the business_unit edge.
	BusinessUnit *BusinessUnit `json:"business_unit,omitempty"`
	// Organization holds the value of the organization edge.
	Organization *Organization `json:"organization,omitempty"`
	// Customer holds the value of the customer edge.
	Customer *Customer `json:"customer,omitempty"`
	// Location holds the value of the location edge.
	Location *Location `json:"location"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [4]bool
}

// BusinessUnitOrErr returns the BusinessUnit value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e DeliverySlotEdges) BusinessUnitOrErr() (*BusinessUnit, error) {
	if e.BusinessUnit != nil {
		return e.BusinessUnit, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: businessunit.Label}
	}
	return nil, &NotLoadedError{edge: "business_unit"}
}

// OrganizationOrErr returns the Organization value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e DeliverySlotEdges) OrganizationOrErr() (*Organization, error) {
	if e.Organization != nil {
		return e.Organization, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: organization.Label}
	}
	return nil, &NotLoadedError{edge: "organization"}
}

// CustomerOrErr returns the Customer value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e DeliverySlotEdges) CustomerOrErr() (*Customer, error) {
	if e.Customer != nil {
		return e.Customer, nil
	} else if e.loadedTypes[2] {
		return nil, &NotFoundError{label: customer.Label}
	}
	return nil, &NotLoadedError{edge: "customer"}
}

// LocationOrErr returns the Location value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e DeliverySlotEdges) LocationOrErr() (*Location, error) {
	if e.Location != nil {
		return e.Location, nil
	} else if e.loadedTypes[3] {
		return nil, &NotFoundError{label: location.Label}
	}
	return nil, &NotLoadedError{edge: "location"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*DeliverySlot) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case deliveryslot.FieldVersion:
			values[i] = new(sql.NullInt64)
		case deliveryslot.FieldDayOfWeek:
			values[i] = new(sql.NullString)
		case deliveryslot.FieldCreatedAt, deliveryslot.FieldUpdatedAt:
			values[i] = new(sql.NullTime)
		case deliveryslot.FieldStartTime, deliveryslot.FieldEndTime:
			values[i] = new(types.TimeOnly)
		case deliveryslot.FieldID, deliveryslot.FieldBusinessUnitID, deliveryslot.FieldOrganizationID, deliveryslot.FieldCustomerID, deliveryslot.FieldLocationID:
			values[i] = new(uuid.UUID)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the DeliverySlot fields.
func (ds *DeliverySlot) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case deliveryslot.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				ds.ID = *value
			}
		case deliveryslot.FieldBusinessUnitID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field business_unit_id", values[i])
			} else if value != nil {
				ds.BusinessUnitID = *value
			}
		case deliveryslot.FieldOrganizationID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field organization_id", values[i])
			} else if value != nil {
				ds.OrganizationID = *value
			}
		case deliveryslot.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				ds.CreatedAt = value.Time
			}
		case deliveryslot.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				ds.UpdatedAt = value.Time
			}
		case deliveryslot.FieldVersion:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field version", values[i])
			} else if value.Valid {
				ds.Version = int(value.Int64)
			}
		case deliveryslot.FieldCustomerID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field customer_id", values[i])
			} else if value != nil {
				ds.CustomerID = *value
			}
		case deliveryslot.FieldLocationID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field location_id", values[i])
			} else if value != nil {
				ds.LocationID = *value
			}
		case deliveryslot.FieldDayOfWeek:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field day_of_week", values[i])
			} else if value.Valid {
				ds.DayOfWeek = deliveryslot.DayOfWeek(value.String)
			}
		case deliveryslot.FieldStartTime:
			if value, ok := values[i].(*types.TimeOnly); !ok {
				return fmt.Errorf("unexpected type %T for field start_time", values[i])
			} else if value != nil {
				ds.StartTime = value
			}
		case deliveryslot.FieldEndTime:
			if value, ok := values[i].(*types.TimeOnly); !ok {
				return fmt.Errorf("unexpected type %T for field end_time", values[i])
			} else if value != nil {
				ds.EndTime = value
			}
		default:
			ds.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the DeliverySlot.
// This includes values selected through modifiers, order, etc.
func (ds *DeliverySlot) Value(name string) (ent.Value, error) {
	return ds.selectValues.Get(name)
}

// QueryBusinessUnit queries the "business_unit" edge of the DeliverySlot entity.
func (ds *DeliverySlot) QueryBusinessUnit() *BusinessUnitQuery {
	return NewDeliverySlotClient(ds.config).QueryBusinessUnit(ds)
}

// QueryOrganization queries the "organization" edge of the DeliverySlot entity.
func (ds *DeliverySlot) QueryOrganization() *OrganizationQuery {
	return NewDeliverySlotClient(ds.config).QueryOrganization(ds)
}

// QueryCustomer queries the "customer" edge of the DeliverySlot entity.
func (ds *DeliverySlot) QueryCustomer() *CustomerQuery {
	return NewDeliverySlotClient(ds.config).QueryCustomer(ds)
}

// QueryLocation queries the "location" edge of the DeliverySlot entity.
func (ds *DeliverySlot) QueryLocation() *LocationQuery {
	return NewDeliverySlotClient(ds.config).QueryLocation(ds)
}

// Update returns a builder for updating this DeliverySlot.
// Note that you need to call DeliverySlot.Unwrap() before calling this method if this DeliverySlot
// was returned from a transaction, and the transaction was committed or rolled back.
func (ds *DeliverySlot) Update() *DeliverySlotUpdateOne {
	return NewDeliverySlotClient(ds.config).UpdateOne(ds)
}

// Unwrap unwraps the DeliverySlot entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (ds *DeliverySlot) Unwrap() *DeliverySlot {
	_tx, ok := ds.config.driver.(*txDriver)
	if !ok {
		panic("ent: DeliverySlot is not a transactional entity")
	}
	ds.config.driver = _tx.drv
	return ds
}

// String implements the fmt.Stringer.
func (ds *DeliverySlot) String() string {
	var builder strings.Builder
	builder.WriteString("DeliverySlot(")
	builder.WriteString(fmt.Sprintf("id=%v, ", ds.ID))
	builder.WriteString("business_unit_id=")
	builder.WriteString(fmt.Sprintf("%v", ds.BusinessUnitID))
	builder.WriteString(", ")
	builder.WriteString("organization_id=")
	builder.WriteString(fmt.Sprintf("%v", ds.OrganizationID))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(ds.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(ds.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("version=")
	builder.WriteString(fmt.Sprintf("%v", ds.Version))
	builder.WriteString(", ")
	builder.WriteString("customer_id=")
	builder.WriteString(fmt.Sprintf("%v", ds.CustomerID))
	builder.WriteString(", ")
	builder.WriteString("location_id=")
	builder.WriteString(fmt.Sprintf("%v", ds.LocationID))
	builder.WriteString(", ")
	builder.WriteString("day_of_week=")
	builder.WriteString(fmt.Sprintf("%v", ds.DayOfWeek))
	builder.WriteString(", ")
	builder.WriteString("start_time=")
	builder.WriteString(fmt.Sprintf("%v", ds.StartTime))
	builder.WriteString(", ")
	builder.WriteString("end_time=")
	builder.WriteString(fmt.Sprintf("%v", ds.EndTime))
	builder.WriteByte(')')
	return builder.String()
}

// DeliverySlots is a parsable slice of DeliverySlot.
type DeliverySlots []*DeliverySlot
