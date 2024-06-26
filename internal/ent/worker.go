// Code generated by entc, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/fleetcode"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/tractor"
	"github.com/emoss08/trenova/internal/ent/user"
	"github.com/emoss08/trenova/internal/ent/usstate"
	"github.com/emoss08/trenova/internal/ent/worker"
	"github.com/emoss08/trenova/internal/ent/workerprofile"
	"github.com/google/uuid"
)

// Worker is the model entity for the Worker schema.
type Worker struct {
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
	// Status holds the value of the "status" field.
	Status worker.Status `json:"status" validate:"required,oneof=A I"`
	// Code holds the value of the "code" field.
	Code string `json:"code" validate:"required,max=10"`
	// ProfilePictureURL holds the value of the "profile_picture_url" field.
	ProfilePictureURL string `json:"profilePictureUrl"`
	// WorkerType holds the value of the "worker_type" field.
	WorkerType worker.WorkerType `json:"workerType" validate:"required,oneof=Employee Contractor"`
	// FirstName holds the value of the "first_name" field.
	FirstName string `json:"firstName" validate:"required,max=255"`
	// LastName holds the value of the "last_name" field.
	LastName string `json:"lastName" validate:"required,max=255"`
	// AddressLine1 holds the value of the "address_line_1" field.
	AddressLine1 string `json:"addressLine1" validate:"required,max=150"`
	// AddressLine2 holds the value of the "address_line_2" field.
	AddressLine2 string `json:"addressLine2" validate:"omitempty,max=150"`
	// City holds the value of the "city" field.
	City string `json:"city" validate:"required,max=150"`
	// PostalCode holds the value of the "postal_code" field.
	PostalCode string `json:"postalCode" validate:"omitempty,max=10"`
	// StateID holds the value of the "state_id" field.
	StateID *uuid.UUID `json:"stateId" validate:"omitempty,uuid"`
	// FleetCodeID holds the value of the "fleet_code_id" field.
	FleetCodeID *uuid.UUID `json:"fleetCodeId" validate:"omitempty,uuid"`
	// ManagerID holds the value of the "manager_id" field.
	ManagerID *uuid.UUID `json:"managerId" validate:"omitempty,uuid"`
	// External ID usually from HOS integration.
	ExternalID string `json:"externalId" validate:"omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the WorkerQuery when eager-loading is set.
	Edges        WorkerEdges `json:"edges"`
	selectValues sql.SelectValues
}

// WorkerEdges holds the relations/edges for other nodes in the graph.
type WorkerEdges struct {
	// BusinessUnit holds the value of the business_unit edge.
	BusinessUnit *BusinessUnit `json:"business_unit,omitempty"`
	// Organization holds the value of the organization edge.
	Organization *Organization `json:"organization,omitempty"`
	// State holds the value of the state edge.
	State *UsState `json:"state"`
	// FleetCode holds the value of the fleet_code edge.
	FleetCode *FleetCode `json:"fleetCode"`
	// Manager holds the value of the manager edge.
	Manager *User `json:"manager"`
	// PrimaryTractor holds the value of the primary_tractor edge.
	PrimaryTractor *Tractor `json:"primaryTractor"`
	// SecondaryTractor holds the value of the secondary_tractor edge.
	SecondaryTractor *Tractor `json:"secondaryTractor"`
	// WorkerProfile holds the value of the worker_profile edge.
	WorkerProfile *WorkerProfile `json:"worker_profile,omitempty"`
	// WorkerComments holds the value of the worker_comments edge.
	WorkerComments []*WorkerComment `json:"worker_comments,omitempty"`
	// WorkerContacts holds the value of the worker_contacts edge.
	WorkerContacts []*WorkerContact `json:"worker_contacts,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes         [10]bool
	namedWorkerComments map[string][]*WorkerComment
	namedWorkerContacts map[string][]*WorkerContact
}

// BusinessUnitOrErr returns the BusinessUnit value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e WorkerEdges) BusinessUnitOrErr() (*BusinessUnit, error) {
	if e.BusinessUnit != nil {
		return e.BusinessUnit, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: businessunit.Label}
	}
	return nil, &NotLoadedError{edge: "business_unit"}
}

// OrganizationOrErr returns the Organization value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e WorkerEdges) OrganizationOrErr() (*Organization, error) {
	if e.Organization != nil {
		return e.Organization, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: organization.Label}
	}
	return nil, &NotLoadedError{edge: "organization"}
}

// StateOrErr returns the State value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e WorkerEdges) StateOrErr() (*UsState, error) {
	if e.State != nil {
		return e.State, nil
	} else if e.loadedTypes[2] {
		return nil, &NotFoundError{label: usstate.Label}
	}
	return nil, &NotLoadedError{edge: "state"}
}

// FleetCodeOrErr returns the FleetCode value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e WorkerEdges) FleetCodeOrErr() (*FleetCode, error) {
	if e.FleetCode != nil {
		return e.FleetCode, nil
	} else if e.loadedTypes[3] {
		return nil, &NotFoundError{label: fleetcode.Label}
	}
	return nil, &NotLoadedError{edge: "fleet_code"}
}

// ManagerOrErr returns the Manager value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e WorkerEdges) ManagerOrErr() (*User, error) {
	if e.Manager != nil {
		return e.Manager, nil
	} else if e.loadedTypes[4] {
		return nil, &NotFoundError{label: user.Label}
	}
	return nil, &NotLoadedError{edge: "manager"}
}

// PrimaryTractorOrErr returns the PrimaryTractor value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e WorkerEdges) PrimaryTractorOrErr() (*Tractor, error) {
	if e.PrimaryTractor != nil {
		return e.PrimaryTractor, nil
	} else if e.loadedTypes[5] {
		return nil, &NotFoundError{label: tractor.Label}
	}
	return nil, &NotLoadedError{edge: "primary_tractor"}
}

// SecondaryTractorOrErr returns the SecondaryTractor value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e WorkerEdges) SecondaryTractorOrErr() (*Tractor, error) {
	if e.SecondaryTractor != nil {
		return e.SecondaryTractor, nil
	} else if e.loadedTypes[6] {
		return nil, &NotFoundError{label: tractor.Label}
	}
	return nil, &NotLoadedError{edge: "secondary_tractor"}
}

// WorkerProfileOrErr returns the WorkerProfile value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e WorkerEdges) WorkerProfileOrErr() (*WorkerProfile, error) {
	if e.WorkerProfile != nil {
		return e.WorkerProfile, nil
	} else if e.loadedTypes[7] {
		return nil, &NotFoundError{label: workerprofile.Label}
	}
	return nil, &NotLoadedError{edge: "worker_profile"}
}

// WorkerCommentsOrErr returns the WorkerComments value or an error if the edge
// was not loaded in eager-loading.
func (e WorkerEdges) WorkerCommentsOrErr() ([]*WorkerComment, error) {
	if e.loadedTypes[8] {
		return e.WorkerComments, nil
	}
	return nil, &NotLoadedError{edge: "worker_comments"}
}

// WorkerContactsOrErr returns the WorkerContacts value or an error if the edge
// was not loaded in eager-loading.
func (e WorkerEdges) WorkerContactsOrErr() ([]*WorkerContact, error) {
	if e.loadedTypes[9] {
		return e.WorkerContacts, nil
	}
	return nil, &NotLoadedError{edge: "worker_contacts"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Worker) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case worker.FieldStateID, worker.FieldFleetCodeID, worker.FieldManagerID:
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case worker.FieldVersion:
			values[i] = new(sql.NullInt64)
		case worker.FieldStatus, worker.FieldCode, worker.FieldProfilePictureURL, worker.FieldWorkerType, worker.FieldFirstName, worker.FieldLastName, worker.FieldAddressLine1, worker.FieldAddressLine2, worker.FieldCity, worker.FieldPostalCode, worker.FieldExternalID:
			values[i] = new(sql.NullString)
		case worker.FieldCreatedAt, worker.FieldUpdatedAt:
			values[i] = new(sql.NullTime)
		case worker.FieldID, worker.FieldBusinessUnitID, worker.FieldOrganizationID:
			values[i] = new(uuid.UUID)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Worker fields.
func (w *Worker) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case worker.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				w.ID = *value
			}
		case worker.FieldBusinessUnitID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field business_unit_id", values[i])
			} else if value != nil {
				w.BusinessUnitID = *value
			}
		case worker.FieldOrganizationID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field organization_id", values[i])
			} else if value != nil {
				w.OrganizationID = *value
			}
		case worker.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				w.CreatedAt = value.Time
			}
		case worker.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				w.UpdatedAt = value.Time
			}
		case worker.FieldVersion:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field version", values[i])
			} else if value.Valid {
				w.Version = int(value.Int64)
			}
		case worker.FieldStatus:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field status", values[i])
			} else if value.Valid {
				w.Status = worker.Status(value.String)
			}
		case worker.FieldCode:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field code", values[i])
			} else if value.Valid {
				w.Code = value.String
			}
		case worker.FieldProfilePictureURL:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field profile_picture_url", values[i])
			} else if value.Valid {
				w.ProfilePictureURL = value.String
			}
		case worker.FieldWorkerType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field worker_type", values[i])
			} else if value.Valid {
				w.WorkerType = worker.WorkerType(value.String)
			}
		case worker.FieldFirstName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field first_name", values[i])
			} else if value.Valid {
				w.FirstName = value.String
			}
		case worker.FieldLastName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field last_name", values[i])
			} else if value.Valid {
				w.LastName = value.String
			}
		case worker.FieldAddressLine1:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field address_line_1", values[i])
			} else if value.Valid {
				w.AddressLine1 = value.String
			}
		case worker.FieldAddressLine2:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field address_line_2", values[i])
			} else if value.Valid {
				w.AddressLine2 = value.String
			}
		case worker.FieldCity:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field city", values[i])
			} else if value.Valid {
				w.City = value.String
			}
		case worker.FieldPostalCode:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field postal_code", values[i])
			} else if value.Valid {
				w.PostalCode = value.String
			}
		case worker.FieldStateID:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field state_id", values[i])
			} else if value.Valid {
				w.StateID = new(uuid.UUID)
				*w.StateID = *value.S.(*uuid.UUID)
			}
		case worker.FieldFleetCodeID:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field fleet_code_id", values[i])
			} else if value.Valid {
				w.FleetCodeID = new(uuid.UUID)
				*w.FleetCodeID = *value.S.(*uuid.UUID)
			}
		case worker.FieldManagerID:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field manager_id", values[i])
			} else if value.Valid {
				w.ManagerID = new(uuid.UUID)
				*w.ManagerID = *value.S.(*uuid.UUID)
			}
		case worker.FieldExternalID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field external_id", values[i])
			} else if value.Valid {
				w.ExternalID = value.String
			}
		default:
			w.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Worker.
// This includes values selected through modifiers, order, etc.
func (w *Worker) Value(name string) (ent.Value, error) {
	return w.selectValues.Get(name)
}

// QueryBusinessUnit queries the "business_unit" edge of the Worker entity.
func (w *Worker) QueryBusinessUnit() *BusinessUnitQuery {
	return NewWorkerClient(w.config).QueryBusinessUnit(w)
}

// QueryOrganization queries the "organization" edge of the Worker entity.
func (w *Worker) QueryOrganization() *OrganizationQuery {
	return NewWorkerClient(w.config).QueryOrganization(w)
}

// QueryState queries the "state" edge of the Worker entity.
func (w *Worker) QueryState() *UsStateQuery {
	return NewWorkerClient(w.config).QueryState(w)
}

// QueryFleetCode queries the "fleet_code" edge of the Worker entity.
func (w *Worker) QueryFleetCode() *FleetCodeQuery {
	return NewWorkerClient(w.config).QueryFleetCode(w)
}

// QueryManager queries the "manager" edge of the Worker entity.
func (w *Worker) QueryManager() *UserQuery {
	return NewWorkerClient(w.config).QueryManager(w)
}

// QueryPrimaryTractor queries the "primary_tractor" edge of the Worker entity.
func (w *Worker) QueryPrimaryTractor() *TractorQuery {
	return NewWorkerClient(w.config).QueryPrimaryTractor(w)
}

// QuerySecondaryTractor queries the "secondary_tractor" edge of the Worker entity.
func (w *Worker) QuerySecondaryTractor() *TractorQuery {
	return NewWorkerClient(w.config).QuerySecondaryTractor(w)
}

// QueryWorkerProfile queries the "worker_profile" edge of the Worker entity.
func (w *Worker) QueryWorkerProfile() *WorkerProfileQuery {
	return NewWorkerClient(w.config).QueryWorkerProfile(w)
}

// QueryWorkerComments queries the "worker_comments" edge of the Worker entity.
func (w *Worker) QueryWorkerComments() *WorkerCommentQuery {
	return NewWorkerClient(w.config).QueryWorkerComments(w)
}

// QueryWorkerContacts queries the "worker_contacts" edge of the Worker entity.
func (w *Worker) QueryWorkerContacts() *WorkerContactQuery {
	return NewWorkerClient(w.config).QueryWorkerContacts(w)
}

// Update returns a builder for updating this Worker.
// Note that you need to call Worker.Unwrap() before calling this method if this Worker
// was returned from a transaction, and the transaction was committed or rolled back.
func (w *Worker) Update() *WorkerUpdateOne {
	return NewWorkerClient(w.config).UpdateOne(w)
}

// Unwrap unwraps the Worker entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (w *Worker) Unwrap() *Worker {
	_tx, ok := w.config.driver.(*txDriver)
	if !ok {
		panic("ent: Worker is not a transactional entity")
	}
	w.config.driver = _tx.drv
	return w
}

// String implements the fmt.Stringer.
func (w *Worker) String() string {
	var builder strings.Builder
	builder.WriteString("Worker(")
	builder.WriteString(fmt.Sprintf("id=%v, ", w.ID))
	builder.WriteString("business_unit_id=")
	builder.WriteString(fmt.Sprintf("%v", w.BusinessUnitID))
	builder.WriteString(", ")
	builder.WriteString("organization_id=")
	builder.WriteString(fmt.Sprintf("%v", w.OrganizationID))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(w.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(w.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("version=")
	builder.WriteString(fmt.Sprintf("%v", w.Version))
	builder.WriteString(", ")
	builder.WriteString("status=")
	builder.WriteString(fmt.Sprintf("%v", w.Status))
	builder.WriteString(", ")
	builder.WriteString("code=")
	builder.WriteString(w.Code)
	builder.WriteString(", ")
	builder.WriteString("profile_picture_url=")
	builder.WriteString(w.ProfilePictureURL)
	builder.WriteString(", ")
	builder.WriteString("worker_type=")
	builder.WriteString(fmt.Sprintf("%v", w.WorkerType))
	builder.WriteString(", ")
	builder.WriteString("first_name=")
	builder.WriteString(w.FirstName)
	builder.WriteString(", ")
	builder.WriteString("last_name=")
	builder.WriteString(w.LastName)
	builder.WriteString(", ")
	builder.WriteString("address_line_1=")
	builder.WriteString(w.AddressLine1)
	builder.WriteString(", ")
	builder.WriteString("address_line_2=")
	builder.WriteString(w.AddressLine2)
	builder.WriteString(", ")
	builder.WriteString("city=")
	builder.WriteString(w.City)
	builder.WriteString(", ")
	builder.WriteString("postal_code=")
	builder.WriteString(w.PostalCode)
	builder.WriteString(", ")
	if v := w.StateID; v != nil {
		builder.WriteString("state_id=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := w.FleetCodeID; v != nil {
		builder.WriteString("fleet_code_id=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := w.ManagerID; v != nil {
		builder.WriteString("manager_id=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	builder.WriteString("external_id=")
	builder.WriteString(w.ExternalID)
	builder.WriteByte(')')
	return builder.String()
}

// NamedWorkerComments returns the WorkerComments named value or an error if the edge was not
// loaded in eager-loading with this name.
func (w *Worker) NamedWorkerComments(name string) ([]*WorkerComment, error) {
	if w.Edges.namedWorkerComments == nil {
		return nil, &NotLoadedError{edge: name}
	}
	nodes, ok := w.Edges.namedWorkerComments[name]
	if !ok {
		return nil, &NotLoadedError{edge: name}
	}
	return nodes, nil
}

func (w *Worker) appendNamedWorkerComments(name string, edges ...*WorkerComment) {
	if w.Edges.namedWorkerComments == nil {
		w.Edges.namedWorkerComments = make(map[string][]*WorkerComment)
	}
	if len(edges) == 0 {
		w.Edges.namedWorkerComments[name] = []*WorkerComment{}
	} else {
		w.Edges.namedWorkerComments[name] = append(w.Edges.namedWorkerComments[name], edges...)
	}
}

// NamedWorkerContacts returns the WorkerContacts named value or an error if the edge was not
// loaded in eager-loading with this name.
func (w *Worker) NamedWorkerContacts(name string) ([]*WorkerContact, error) {
	if w.Edges.namedWorkerContacts == nil {
		return nil, &NotLoadedError{edge: name}
	}
	nodes, ok := w.Edges.namedWorkerContacts[name]
	if !ok {
		return nil, &NotLoadedError{edge: name}
	}
	return nodes, nil
}

func (w *Worker) appendNamedWorkerContacts(name string, edges ...*WorkerContact) {
	if w.Edges.namedWorkerContacts == nil {
		w.Edges.namedWorkerContacts = make(map[string][]*WorkerContact)
	}
	if len(edges) == 0 {
		w.Edges.namedWorkerContacts[name] = []*WorkerContact{}
	} else {
		w.Edges.namedWorkerContacts[name] = append(w.Edges.namedWorkerContacts[name], edges...)
	}
}

// Workers is a parsable slice of Worker.
type Workers []*Worker
