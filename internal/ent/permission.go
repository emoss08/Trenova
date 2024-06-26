// Code generated by entc, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/permission"
	"github.com/emoss08/trenova/internal/ent/resource"
	"github.com/google/uuid"
)

// Permission is the model entity for the Permission schema.
type Permission struct {
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
	// Codename holds the value of the "codename" field.
	Codename string `json:"codename,omitempty"`
	// Action holds the value of the "action" field.
	Action string `json:"action,omitempty"`
	// Label holds the value of the "label" field.
	Label string `json:"label"`
	// ReadDescription holds the value of the "read_description" field.
	ReadDescription string `json:"readDescription"`
	// WriteDescription holds the value of the "write_description" field.
	WriteDescription string `json:"writeDescription"`
	// ResourceID holds the value of the "resource_id" field.
	ResourceID uuid.UUID `json:"resourceId"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the PermissionQuery when eager-loading is set.
	Edges        PermissionEdges `json:"edges"`
	selectValues sql.SelectValues
}

// PermissionEdges holds the relations/edges for other nodes in the graph.
type PermissionEdges struct {
	// BusinessUnit holds the value of the business_unit edge.
	BusinessUnit *BusinessUnit `json:"business_unit,omitempty"`
	// Organization holds the value of the organization edge.
	Organization *Organization `json:"organization,omitempty"`
	// Resource holds the value of the resource edge.
	Resource *Resource `json:"resource,omitempty"`
	// Roles holds the value of the roles edge.
	Roles []*Role `json:"roles,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [4]bool
	namedRoles  map[string][]*Role
}

// BusinessUnitOrErr returns the BusinessUnit value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e PermissionEdges) BusinessUnitOrErr() (*BusinessUnit, error) {
	if e.BusinessUnit != nil {
		return e.BusinessUnit, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: businessunit.Label}
	}
	return nil, &NotLoadedError{edge: "business_unit"}
}

// OrganizationOrErr returns the Organization value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e PermissionEdges) OrganizationOrErr() (*Organization, error) {
	if e.Organization != nil {
		return e.Organization, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: organization.Label}
	}
	return nil, &NotLoadedError{edge: "organization"}
}

// ResourceOrErr returns the Resource value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e PermissionEdges) ResourceOrErr() (*Resource, error) {
	if e.Resource != nil {
		return e.Resource, nil
	} else if e.loadedTypes[2] {
		return nil, &NotFoundError{label: resource.Label}
	}
	return nil, &NotLoadedError{edge: "resource"}
}

// RolesOrErr returns the Roles value or an error if the edge
// was not loaded in eager-loading.
func (e PermissionEdges) RolesOrErr() ([]*Role, error) {
	if e.loadedTypes[3] {
		return e.Roles, nil
	}
	return nil, &NotLoadedError{edge: "roles"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Permission) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case permission.FieldVersion:
			values[i] = new(sql.NullInt64)
		case permission.FieldCodename, permission.FieldAction, permission.FieldLabel, permission.FieldReadDescription, permission.FieldWriteDescription:
			values[i] = new(sql.NullString)
		case permission.FieldCreatedAt, permission.FieldUpdatedAt:
			values[i] = new(sql.NullTime)
		case permission.FieldID, permission.FieldBusinessUnitID, permission.FieldOrganizationID, permission.FieldResourceID:
			values[i] = new(uuid.UUID)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Permission fields.
func (pe *Permission) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case permission.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				pe.ID = *value
			}
		case permission.FieldBusinessUnitID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field business_unit_id", values[i])
			} else if value != nil {
				pe.BusinessUnitID = *value
			}
		case permission.FieldOrganizationID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field organization_id", values[i])
			} else if value != nil {
				pe.OrganizationID = *value
			}
		case permission.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				pe.CreatedAt = value.Time
			}
		case permission.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				pe.UpdatedAt = value.Time
			}
		case permission.FieldVersion:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field version", values[i])
			} else if value.Valid {
				pe.Version = int(value.Int64)
			}
		case permission.FieldCodename:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field codename", values[i])
			} else if value.Valid {
				pe.Codename = value.String
			}
		case permission.FieldAction:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field action", values[i])
			} else if value.Valid {
				pe.Action = value.String
			}
		case permission.FieldLabel:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field label", values[i])
			} else if value.Valid {
				pe.Label = value.String
			}
		case permission.FieldReadDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field read_description", values[i])
			} else if value.Valid {
				pe.ReadDescription = value.String
			}
		case permission.FieldWriteDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field write_description", values[i])
			} else if value.Valid {
				pe.WriteDescription = value.String
			}
		case permission.FieldResourceID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field resource_id", values[i])
			} else if value != nil {
				pe.ResourceID = *value
			}
		default:
			pe.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Permission.
// This includes values selected through modifiers, order, etc.
func (pe *Permission) Value(name string) (ent.Value, error) {
	return pe.selectValues.Get(name)
}

// QueryBusinessUnit queries the "business_unit" edge of the Permission entity.
func (pe *Permission) QueryBusinessUnit() *BusinessUnitQuery {
	return NewPermissionClient(pe.config).QueryBusinessUnit(pe)
}

// QueryOrganization queries the "organization" edge of the Permission entity.
func (pe *Permission) QueryOrganization() *OrganizationQuery {
	return NewPermissionClient(pe.config).QueryOrganization(pe)
}

// QueryResource queries the "resource" edge of the Permission entity.
func (pe *Permission) QueryResource() *ResourceQuery {
	return NewPermissionClient(pe.config).QueryResource(pe)
}

// QueryRoles queries the "roles" edge of the Permission entity.
func (pe *Permission) QueryRoles() *RoleQuery {
	return NewPermissionClient(pe.config).QueryRoles(pe)
}

// Update returns a builder for updating this Permission.
// Note that you need to call Permission.Unwrap() before calling this method if this Permission
// was returned from a transaction, and the transaction was committed or rolled back.
func (pe *Permission) Update() *PermissionUpdateOne {
	return NewPermissionClient(pe.config).UpdateOne(pe)
}

// Unwrap unwraps the Permission entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (pe *Permission) Unwrap() *Permission {
	_tx, ok := pe.config.driver.(*txDriver)
	if !ok {
		panic("ent: Permission is not a transactional entity")
	}
	pe.config.driver = _tx.drv
	return pe
}

// String implements the fmt.Stringer.
func (pe *Permission) String() string {
	var builder strings.Builder
	builder.WriteString("Permission(")
	builder.WriteString(fmt.Sprintf("id=%v, ", pe.ID))
	builder.WriteString("business_unit_id=")
	builder.WriteString(fmt.Sprintf("%v", pe.BusinessUnitID))
	builder.WriteString(", ")
	builder.WriteString("organization_id=")
	builder.WriteString(fmt.Sprintf("%v", pe.OrganizationID))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(pe.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(pe.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("version=")
	builder.WriteString(fmt.Sprintf("%v", pe.Version))
	builder.WriteString(", ")
	builder.WriteString("codename=")
	builder.WriteString(pe.Codename)
	builder.WriteString(", ")
	builder.WriteString("action=")
	builder.WriteString(pe.Action)
	builder.WriteString(", ")
	builder.WriteString("label=")
	builder.WriteString(pe.Label)
	builder.WriteString(", ")
	builder.WriteString("read_description=")
	builder.WriteString(pe.ReadDescription)
	builder.WriteString(", ")
	builder.WriteString("write_description=")
	builder.WriteString(pe.WriteDescription)
	builder.WriteString(", ")
	builder.WriteString("resource_id=")
	builder.WriteString(fmt.Sprintf("%v", pe.ResourceID))
	builder.WriteByte(')')
	return builder.String()
}

// NamedRoles returns the Roles named value or an error if the edge was not
// loaded in eager-loading with this name.
func (pe *Permission) NamedRoles(name string) ([]*Role, error) {
	if pe.Edges.namedRoles == nil {
		return nil, &NotLoadedError{edge: name}
	}
	nodes, ok := pe.Edges.namedRoles[name]
	if !ok {
		return nil, &NotLoadedError{edge: name}
	}
	return nodes, nil
}

func (pe *Permission) appendNamedRoles(name string, edges ...*Role) {
	if pe.Edges.namedRoles == nil {
		pe.Edges.namedRoles = make(map[string][]*Role)
	}
	if len(edges) == 0 {
		pe.Edges.namedRoles[name] = []*Role{}
	} else {
		pe.Edges.namedRoles[name] = append(pe.Edges.namedRoles[name], edges...)
	}
}

// Permissions is a parsable slice of Permission.
type Permissions []*Permission
