// Code generated by entc, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/customreport"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

// CustomReport is the model entity for the CustomReport schema.
type CustomReport struct {
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
	// Name holds the value of the "name" field.
	Name string `json:"name" validate:"required"`
	// Description holds the value of the "description" field.
	Description string `json:"description" validate:"omitempty"`
	// Table holds the value of the "table" field.
	Table string `json:"table" validate:"omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the CustomReportQuery when eager-loading is set.
	Edges        CustomReportEdges `json:"edges"`
	selectValues sql.SelectValues
}

// CustomReportEdges holds the relations/edges for other nodes in the graph.
type CustomReportEdges struct {
	// BusinessUnit holds the value of the business_unit edge.
	BusinessUnit *BusinessUnit `json:"business_unit,omitempty"`
	// Organization holds the value of the organization edge.
	Organization *Organization `json:"organization,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [2]bool
}

// BusinessUnitOrErr returns the BusinessUnit value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e CustomReportEdges) BusinessUnitOrErr() (*BusinessUnit, error) {
	if e.BusinessUnit != nil {
		return e.BusinessUnit, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: businessunit.Label}
	}
	return nil, &NotLoadedError{edge: "business_unit"}
}

// OrganizationOrErr returns the Organization value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e CustomReportEdges) OrganizationOrErr() (*Organization, error) {
	if e.Organization != nil {
		return e.Organization, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: organization.Label}
	}
	return nil, &NotLoadedError{edge: "organization"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*CustomReport) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case customreport.FieldVersion:
			values[i] = new(sql.NullInt64)
		case customreport.FieldName, customreport.FieldDescription, customreport.FieldTable:
			values[i] = new(sql.NullString)
		case customreport.FieldCreatedAt, customreport.FieldUpdatedAt:
			values[i] = new(sql.NullTime)
		case customreport.FieldID, customreport.FieldBusinessUnitID, customreport.FieldOrganizationID:
			values[i] = new(uuid.UUID)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the CustomReport fields.
func (cr *CustomReport) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case customreport.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				cr.ID = *value
			}
		case customreport.FieldBusinessUnitID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field business_unit_id", values[i])
			} else if value != nil {
				cr.BusinessUnitID = *value
			}
		case customreport.FieldOrganizationID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field organization_id", values[i])
			} else if value != nil {
				cr.OrganizationID = *value
			}
		case customreport.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				cr.CreatedAt = value.Time
			}
		case customreport.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				cr.UpdatedAt = value.Time
			}
		case customreport.FieldVersion:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field version", values[i])
			} else if value.Valid {
				cr.Version = int(value.Int64)
			}
		case customreport.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				cr.Name = value.String
			}
		case customreport.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				cr.Description = value.String
			}
		case customreport.FieldTable:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field table", values[i])
			} else if value.Valid {
				cr.Table = value.String
			}
		default:
			cr.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the CustomReport.
// This includes values selected through modifiers, order, etc.
func (cr *CustomReport) Value(name string) (ent.Value, error) {
	return cr.selectValues.Get(name)
}

// QueryBusinessUnit queries the "business_unit" edge of the CustomReport entity.
func (cr *CustomReport) QueryBusinessUnit() *BusinessUnitQuery {
	return NewCustomReportClient(cr.config).QueryBusinessUnit(cr)
}

// QueryOrganization queries the "organization" edge of the CustomReport entity.
func (cr *CustomReport) QueryOrganization() *OrganizationQuery {
	return NewCustomReportClient(cr.config).QueryOrganization(cr)
}

// Update returns a builder for updating this CustomReport.
// Note that you need to call CustomReport.Unwrap() before calling this method if this CustomReport
// was returned from a transaction, and the transaction was committed or rolled back.
func (cr *CustomReport) Update() *CustomReportUpdateOne {
	return NewCustomReportClient(cr.config).UpdateOne(cr)
}

// Unwrap unwraps the CustomReport entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (cr *CustomReport) Unwrap() *CustomReport {
	_tx, ok := cr.config.driver.(*txDriver)
	if !ok {
		panic("ent: CustomReport is not a transactional entity")
	}
	cr.config.driver = _tx.drv
	return cr
}

// String implements the fmt.Stringer.
func (cr *CustomReport) String() string {
	var builder strings.Builder
	builder.WriteString("CustomReport(")
	builder.WriteString(fmt.Sprintf("id=%v, ", cr.ID))
	builder.WriteString("business_unit_id=")
	builder.WriteString(fmt.Sprintf("%v", cr.BusinessUnitID))
	builder.WriteString(", ")
	builder.WriteString("organization_id=")
	builder.WriteString(fmt.Sprintf("%v", cr.OrganizationID))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(cr.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(cr.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("version=")
	builder.WriteString(fmt.Sprintf("%v", cr.Version))
	builder.WriteString(", ")
	builder.WriteString("name=")
	builder.WriteString(cr.Name)
	builder.WriteString(", ")
	builder.WriteString("description=")
	builder.WriteString(cr.Description)
	builder.WriteString(", ")
	builder.WriteString("table=")
	builder.WriteString(cr.Table)
	builder.WriteByte(')')
	return builder.String()
}

// CustomReports is a parsable slice of CustomReport.
type CustomReports []*CustomReport
