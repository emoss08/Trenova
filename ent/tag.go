// Code generated by ent, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/ent/businessunit"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/emoss08/trenova/ent/tag"
	"github.com/google/uuid"
)

// Tag is the model entity for the Tag schema.
type Tag struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// BusinessUnitID holds the value of the "business_unit_id" field.
	BusinessUnitID uuid.UUID `json:"businessUnitId"`
	// OrganizationID holds the value of the "organization_id" field.
	OrganizationID uuid.UUID `json:"organizationId"`
	// CreatedAt holds the value of the "created_at" field.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt holds the value of the "updated_at" field.
	UpdatedAt time.Time `json:"updatedAt"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty"`
	// Description holds the value of the "description" field.
	Description *string `json:"description,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the TagQuery when eager-loading is set.
	Edges                       TagEdges `json:"edges"`
	general_ledger_account_tags *uuid.UUID
	selectValues                sql.SelectValues
}

// TagEdges holds the relations/edges for other nodes in the graph.
type TagEdges struct {
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
func (e TagEdges) BusinessUnitOrErr() (*BusinessUnit, error) {
	if e.BusinessUnit != nil {
		return e.BusinessUnit, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: businessunit.Label}
	}
	return nil, &NotLoadedError{edge: "business_unit"}
}

// OrganizationOrErr returns the Organization value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e TagEdges) OrganizationOrErr() (*Organization, error) {
	if e.Organization != nil {
		return e.Organization, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: organization.Label}
	}
	return nil, &NotLoadedError{edge: "organization"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Tag) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case tag.FieldName, tag.FieldDescription:
			values[i] = new(sql.NullString)
		case tag.FieldCreatedAt, tag.FieldUpdatedAt:
			values[i] = new(sql.NullTime)
		case tag.FieldID, tag.FieldBusinessUnitID, tag.FieldOrganizationID:
			values[i] = new(uuid.UUID)
		case tag.ForeignKeys[0]: // general_ledger_account_tags
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Tag fields.
func (t *Tag) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case tag.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				t.ID = *value
			}
		case tag.FieldBusinessUnitID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field business_unit_id", values[i])
			} else if value != nil {
				t.BusinessUnitID = *value
			}
		case tag.FieldOrganizationID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field organization_id", values[i])
			} else if value != nil {
				t.OrganizationID = *value
			}
		case tag.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				t.CreatedAt = value.Time
			}
		case tag.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				t.UpdatedAt = value.Time
			}
		case tag.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				t.Name = value.String
			}
		case tag.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				t.Description = new(string)
				*t.Description = value.String
			}
		case tag.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field general_ledger_account_tags", values[i])
			} else if value.Valid {
				t.general_ledger_account_tags = new(uuid.UUID)
				*t.general_ledger_account_tags = *value.S.(*uuid.UUID)
			}
		default:
			t.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Tag.
// This includes values selected through modifiers, order, etc.
func (t *Tag) Value(name string) (ent.Value, error) {
	return t.selectValues.Get(name)
}

// QueryBusinessUnit queries the "business_unit" edge of the Tag entity.
func (t *Tag) QueryBusinessUnit() *BusinessUnitQuery {
	return NewTagClient(t.config).QueryBusinessUnit(t)
}

// QueryOrganization queries the "organization" edge of the Tag entity.
func (t *Tag) QueryOrganization() *OrganizationQuery {
	return NewTagClient(t.config).QueryOrganization(t)
}

// Update returns a builder for updating this Tag.
// Note that you need to call Tag.Unwrap() before calling this method if this Tag
// was returned from a transaction, and the transaction was committed or rolled back.
func (t *Tag) Update() *TagUpdateOne {
	return NewTagClient(t.config).UpdateOne(t)
}

// Unwrap unwraps the Tag entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (t *Tag) Unwrap() *Tag {
	_tx, ok := t.config.driver.(*txDriver)
	if !ok {
		panic("ent: Tag is not a transactional entity")
	}
	t.config.driver = _tx.drv
	return t
}

// String implements the fmt.Stringer.
func (t *Tag) String() string {
	var builder strings.Builder
	builder.WriteString("Tag(")
	builder.WriteString(fmt.Sprintf("id=%v, ", t.ID))
	builder.WriteString("business_unit_id=")
	builder.WriteString(fmt.Sprintf("%v", t.BusinessUnitID))
	builder.WriteString(", ")
	builder.WriteString("organization_id=")
	builder.WriteString(fmt.Sprintf("%v", t.OrganizationID))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(t.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(t.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("name=")
	builder.WriteString(t.Name)
	builder.WriteString(", ")
	if v := t.Description; v != nil {
		builder.WriteString("description=")
		builder.WriteString(*v)
	}
	builder.WriteByte(')')
	return builder.String()
}

// Tags is a parsable slice of Tag.
type Tags []*Tag
