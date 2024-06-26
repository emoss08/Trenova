// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/predicate"
	"github.com/emoss08/trenova/internal/ent/usstate"
	"github.com/emoss08/trenova/internal/ent/workerprofile"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// WorkerProfileUpdate is the builder for updating WorkerProfile entities.
type WorkerProfileUpdate struct {
	config
	hooks     []Hook
	mutation  *WorkerProfileMutation
	modifiers []func(*sql.UpdateBuilder)
}

// Where appends a list predicates to the WorkerProfileUpdate builder.
func (wpu *WorkerProfileUpdate) Where(ps ...predicate.WorkerProfile) *WorkerProfileUpdate {
	wpu.mutation.Where(ps...)
	return wpu
}

// SetOrganizationID sets the "organization_id" field.
func (wpu *WorkerProfileUpdate) SetOrganizationID(u uuid.UUID) *WorkerProfileUpdate {
	wpu.mutation.SetOrganizationID(u)
	return wpu
}

// SetNillableOrganizationID sets the "organization_id" field if the given value is not nil.
func (wpu *WorkerProfileUpdate) SetNillableOrganizationID(u *uuid.UUID) *WorkerProfileUpdate {
	if u != nil {
		wpu.SetOrganizationID(*u)
	}
	return wpu
}

// SetUpdatedAt sets the "updated_at" field.
func (wpu *WorkerProfileUpdate) SetUpdatedAt(t time.Time) *WorkerProfileUpdate {
	wpu.mutation.SetUpdatedAt(t)
	return wpu
}

// SetVersion sets the "version" field.
func (wpu *WorkerProfileUpdate) SetVersion(i int) *WorkerProfileUpdate {
	wpu.mutation.ResetVersion()
	wpu.mutation.SetVersion(i)
	return wpu
}

// SetNillableVersion sets the "version" field if the given value is not nil.
func (wpu *WorkerProfileUpdate) SetNillableVersion(i *int) *WorkerProfileUpdate {
	if i != nil {
		wpu.SetVersion(*i)
	}
	return wpu
}

// AddVersion adds i to the "version" field.
func (wpu *WorkerProfileUpdate) AddVersion(i int) *WorkerProfileUpdate {
	wpu.mutation.AddVersion(i)
	return wpu
}

// SetRace sets the "race" field.
func (wpu *WorkerProfileUpdate) SetRace(s string) *WorkerProfileUpdate {
	wpu.mutation.SetRace(s)
	return wpu
}

// SetNillableRace sets the "race" field if the given value is not nil.
func (wpu *WorkerProfileUpdate) SetNillableRace(s *string) *WorkerProfileUpdate {
	if s != nil {
		wpu.SetRace(*s)
	}
	return wpu
}

// ClearRace clears the value of the "race" field.
func (wpu *WorkerProfileUpdate) ClearRace() *WorkerProfileUpdate {
	wpu.mutation.ClearRace()
	return wpu
}

// SetSex sets the "sex" field.
func (wpu *WorkerProfileUpdate) SetSex(s string) *WorkerProfileUpdate {
	wpu.mutation.SetSex(s)
	return wpu
}

// SetNillableSex sets the "sex" field if the given value is not nil.
func (wpu *WorkerProfileUpdate) SetNillableSex(s *string) *WorkerProfileUpdate {
	if s != nil {
		wpu.SetSex(*s)
	}
	return wpu
}

// ClearSex clears the value of the "sex" field.
func (wpu *WorkerProfileUpdate) ClearSex() *WorkerProfileUpdate {
	wpu.mutation.ClearSex()
	return wpu
}

// SetDateOfBirth sets the "date_of_birth" field.
func (wpu *WorkerProfileUpdate) SetDateOfBirth(pg *pgtype.Date) *WorkerProfileUpdate {
	wpu.mutation.SetDateOfBirth(pg)
	return wpu
}

// ClearDateOfBirth clears the value of the "date_of_birth" field.
func (wpu *WorkerProfileUpdate) ClearDateOfBirth() *WorkerProfileUpdate {
	wpu.mutation.ClearDateOfBirth()
	return wpu
}

// SetLicenseNumber sets the "license_number" field.
func (wpu *WorkerProfileUpdate) SetLicenseNumber(s string) *WorkerProfileUpdate {
	wpu.mutation.SetLicenseNumber(s)
	return wpu
}

// SetNillableLicenseNumber sets the "license_number" field if the given value is not nil.
func (wpu *WorkerProfileUpdate) SetNillableLicenseNumber(s *string) *WorkerProfileUpdate {
	if s != nil {
		wpu.SetLicenseNumber(*s)
	}
	return wpu
}

// SetLicenseStateID sets the "license_state_id" field.
func (wpu *WorkerProfileUpdate) SetLicenseStateID(u uuid.UUID) *WorkerProfileUpdate {
	wpu.mutation.SetLicenseStateID(u)
	return wpu
}

// SetNillableLicenseStateID sets the "license_state_id" field if the given value is not nil.
func (wpu *WorkerProfileUpdate) SetNillableLicenseStateID(u *uuid.UUID) *WorkerProfileUpdate {
	if u != nil {
		wpu.SetLicenseStateID(*u)
	}
	return wpu
}

// SetLicenseExpirationDate sets the "license_expiration_date" field.
func (wpu *WorkerProfileUpdate) SetLicenseExpirationDate(pg *pgtype.Date) *WorkerProfileUpdate {
	wpu.mutation.SetLicenseExpirationDate(pg)
	return wpu
}

// ClearLicenseExpirationDate clears the value of the "license_expiration_date" field.
func (wpu *WorkerProfileUpdate) ClearLicenseExpirationDate() *WorkerProfileUpdate {
	wpu.mutation.ClearLicenseExpirationDate()
	return wpu
}

// SetEndorsements sets the "endorsements" field.
func (wpu *WorkerProfileUpdate) SetEndorsements(w workerprofile.Endorsements) *WorkerProfileUpdate {
	wpu.mutation.SetEndorsements(w)
	return wpu
}

// SetNillableEndorsements sets the "endorsements" field if the given value is not nil.
func (wpu *WorkerProfileUpdate) SetNillableEndorsements(w *workerprofile.Endorsements) *WorkerProfileUpdate {
	if w != nil {
		wpu.SetEndorsements(*w)
	}
	return wpu
}

// ClearEndorsements clears the value of the "endorsements" field.
func (wpu *WorkerProfileUpdate) ClearEndorsements() *WorkerProfileUpdate {
	wpu.mutation.ClearEndorsements()
	return wpu
}

// SetHazmatExpirationDate sets the "hazmat_expiration_date" field.
func (wpu *WorkerProfileUpdate) SetHazmatExpirationDate(pg *pgtype.Date) *WorkerProfileUpdate {
	wpu.mutation.SetHazmatExpirationDate(pg)
	return wpu
}

// ClearHazmatExpirationDate clears the value of the "hazmat_expiration_date" field.
func (wpu *WorkerProfileUpdate) ClearHazmatExpirationDate() *WorkerProfileUpdate {
	wpu.mutation.ClearHazmatExpirationDate()
	return wpu
}

// SetHireDate sets the "hire_date" field.
func (wpu *WorkerProfileUpdate) SetHireDate(pg *pgtype.Date) *WorkerProfileUpdate {
	wpu.mutation.SetHireDate(pg)
	return wpu
}

// ClearHireDate clears the value of the "hire_date" field.
func (wpu *WorkerProfileUpdate) ClearHireDate() *WorkerProfileUpdate {
	wpu.mutation.ClearHireDate()
	return wpu
}

// SetTerminationDate sets the "termination_date" field.
func (wpu *WorkerProfileUpdate) SetTerminationDate(pg *pgtype.Date) *WorkerProfileUpdate {
	wpu.mutation.SetTerminationDate(pg)
	return wpu
}

// ClearTerminationDate clears the value of the "termination_date" field.
func (wpu *WorkerProfileUpdate) ClearTerminationDate() *WorkerProfileUpdate {
	wpu.mutation.ClearTerminationDate()
	return wpu
}

// SetPhysicalDueDate sets the "physical_due_date" field.
func (wpu *WorkerProfileUpdate) SetPhysicalDueDate(pg *pgtype.Date) *WorkerProfileUpdate {
	wpu.mutation.SetPhysicalDueDate(pg)
	return wpu
}

// ClearPhysicalDueDate clears the value of the "physical_due_date" field.
func (wpu *WorkerProfileUpdate) ClearPhysicalDueDate() *WorkerProfileUpdate {
	wpu.mutation.ClearPhysicalDueDate()
	return wpu
}

// SetMedicalCertDate sets the "medical_cert_date" field.
func (wpu *WorkerProfileUpdate) SetMedicalCertDate(pg *pgtype.Date) *WorkerProfileUpdate {
	wpu.mutation.SetMedicalCertDate(pg)
	return wpu
}

// ClearMedicalCertDate clears the value of the "medical_cert_date" field.
func (wpu *WorkerProfileUpdate) ClearMedicalCertDate() *WorkerProfileUpdate {
	wpu.mutation.ClearMedicalCertDate()
	return wpu
}

// SetMvrDueDate sets the "mvr_due_date" field.
func (wpu *WorkerProfileUpdate) SetMvrDueDate(pg *pgtype.Date) *WorkerProfileUpdate {
	wpu.mutation.SetMvrDueDate(pg)
	return wpu
}

// ClearMvrDueDate clears the value of the "mvr_due_date" field.
func (wpu *WorkerProfileUpdate) ClearMvrDueDate() *WorkerProfileUpdate {
	wpu.mutation.ClearMvrDueDate()
	return wpu
}

// SetOrganization sets the "organization" edge to the Organization entity.
func (wpu *WorkerProfileUpdate) SetOrganization(o *Organization) *WorkerProfileUpdate {
	return wpu.SetOrganizationID(o.ID)
}

// SetStateID sets the "state" edge to the UsState entity by ID.
func (wpu *WorkerProfileUpdate) SetStateID(id uuid.UUID) *WorkerProfileUpdate {
	wpu.mutation.SetStateID(id)
	return wpu
}

// SetState sets the "state" edge to the UsState entity.
func (wpu *WorkerProfileUpdate) SetState(u *UsState) *WorkerProfileUpdate {
	return wpu.SetStateID(u.ID)
}

// Mutation returns the WorkerProfileMutation object of the builder.
func (wpu *WorkerProfileUpdate) Mutation() *WorkerProfileMutation {
	return wpu.mutation
}

// ClearOrganization clears the "organization" edge to the Organization entity.
func (wpu *WorkerProfileUpdate) ClearOrganization() *WorkerProfileUpdate {
	wpu.mutation.ClearOrganization()
	return wpu
}

// ClearState clears the "state" edge to the UsState entity.
func (wpu *WorkerProfileUpdate) ClearState() *WorkerProfileUpdate {
	wpu.mutation.ClearState()
	return wpu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (wpu *WorkerProfileUpdate) Save(ctx context.Context) (int, error) {
	if err := wpu.defaults(); err != nil {
		return 0, err
	}
	return withHooks(ctx, wpu.sqlSave, wpu.mutation, wpu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (wpu *WorkerProfileUpdate) SaveX(ctx context.Context) int {
	affected, err := wpu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (wpu *WorkerProfileUpdate) Exec(ctx context.Context) error {
	_, err := wpu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (wpu *WorkerProfileUpdate) ExecX(ctx context.Context) {
	if err := wpu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (wpu *WorkerProfileUpdate) defaults() error {
	if _, ok := wpu.mutation.UpdatedAt(); !ok {
		if workerprofile.UpdateDefaultUpdatedAt == nil {
			return fmt.Errorf("ent: uninitialized workerprofile.UpdateDefaultUpdatedAt (forgotten import ent/runtime?)")
		}
		v := workerprofile.UpdateDefaultUpdatedAt()
		wpu.mutation.SetUpdatedAt(v)
	}
	return nil
}

// check runs all checks and user-defined validators on the builder.
func (wpu *WorkerProfileUpdate) check() error {
	if v, ok := wpu.mutation.LicenseNumber(); ok {
		if err := workerprofile.LicenseNumberValidator(v); err != nil {
			return &ValidationError{Name: "license_number", err: fmt.Errorf(`ent: validator failed for field "WorkerProfile.license_number": %w`, err)}
		}
	}
	if v, ok := wpu.mutation.Endorsements(); ok {
		if err := workerprofile.EndorsementsValidator(v); err != nil {
			return &ValidationError{Name: "endorsements", err: fmt.Errorf(`ent: validator failed for field "WorkerProfile.endorsements": %w`, err)}
		}
	}
	if _, ok := wpu.mutation.BusinessUnitID(); wpu.mutation.BusinessUnitCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "WorkerProfile.business_unit"`)
	}
	if _, ok := wpu.mutation.OrganizationID(); wpu.mutation.OrganizationCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "WorkerProfile.organization"`)
	}
	if _, ok := wpu.mutation.WorkerID(); wpu.mutation.WorkerCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "WorkerProfile.worker"`)
	}
	if _, ok := wpu.mutation.StateID(); wpu.mutation.StateCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "WorkerProfile.state"`)
	}
	return nil
}

// Modify adds a statement modifier for attaching custom logic to the UPDATE statement.
func (wpu *WorkerProfileUpdate) Modify(modifiers ...func(u *sql.UpdateBuilder)) *WorkerProfileUpdate {
	wpu.modifiers = append(wpu.modifiers, modifiers...)
	return wpu
}

func (wpu *WorkerProfileUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := wpu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(workerprofile.Table, workerprofile.Columns, sqlgraph.NewFieldSpec(workerprofile.FieldID, field.TypeUUID))
	if ps := wpu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := wpu.mutation.UpdatedAt(); ok {
		_spec.SetField(workerprofile.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := wpu.mutation.Version(); ok {
		_spec.SetField(workerprofile.FieldVersion, field.TypeInt, value)
	}
	if value, ok := wpu.mutation.AddedVersion(); ok {
		_spec.AddField(workerprofile.FieldVersion, field.TypeInt, value)
	}
	if value, ok := wpu.mutation.Race(); ok {
		_spec.SetField(workerprofile.FieldRace, field.TypeString, value)
	}
	if wpu.mutation.RaceCleared() {
		_spec.ClearField(workerprofile.FieldRace, field.TypeString)
	}
	if value, ok := wpu.mutation.Sex(); ok {
		_spec.SetField(workerprofile.FieldSex, field.TypeString, value)
	}
	if wpu.mutation.SexCleared() {
		_spec.ClearField(workerprofile.FieldSex, field.TypeString)
	}
	if value, ok := wpu.mutation.DateOfBirth(); ok {
		_spec.SetField(workerprofile.FieldDateOfBirth, field.TypeOther, value)
	}
	if wpu.mutation.DateOfBirthCleared() {
		_spec.ClearField(workerprofile.FieldDateOfBirth, field.TypeOther)
	}
	if value, ok := wpu.mutation.LicenseNumber(); ok {
		_spec.SetField(workerprofile.FieldLicenseNumber, field.TypeString, value)
	}
	if value, ok := wpu.mutation.LicenseExpirationDate(); ok {
		_spec.SetField(workerprofile.FieldLicenseExpirationDate, field.TypeOther, value)
	}
	if wpu.mutation.LicenseExpirationDateCleared() {
		_spec.ClearField(workerprofile.FieldLicenseExpirationDate, field.TypeOther)
	}
	if value, ok := wpu.mutation.Endorsements(); ok {
		_spec.SetField(workerprofile.FieldEndorsements, field.TypeEnum, value)
	}
	if wpu.mutation.EndorsementsCleared() {
		_spec.ClearField(workerprofile.FieldEndorsements, field.TypeEnum)
	}
	if value, ok := wpu.mutation.HazmatExpirationDate(); ok {
		_spec.SetField(workerprofile.FieldHazmatExpirationDate, field.TypeOther, value)
	}
	if wpu.mutation.HazmatExpirationDateCleared() {
		_spec.ClearField(workerprofile.FieldHazmatExpirationDate, field.TypeOther)
	}
	if value, ok := wpu.mutation.HireDate(); ok {
		_spec.SetField(workerprofile.FieldHireDate, field.TypeOther, value)
	}
	if wpu.mutation.HireDateCleared() {
		_spec.ClearField(workerprofile.FieldHireDate, field.TypeOther)
	}
	if value, ok := wpu.mutation.TerminationDate(); ok {
		_spec.SetField(workerprofile.FieldTerminationDate, field.TypeOther, value)
	}
	if wpu.mutation.TerminationDateCleared() {
		_spec.ClearField(workerprofile.FieldTerminationDate, field.TypeOther)
	}
	if value, ok := wpu.mutation.PhysicalDueDate(); ok {
		_spec.SetField(workerprofile.FieldPhysicalDueDate, field.TypeOther, value)
	}
	if wpu.mutation.PhysicalDueDateCleared() {
		_spec.ClearField(workerprofile.FieldPhysicalDueDate, field.TypeOther)
	}
	if value, ok := wpu.mutation.MedicalCertDate(); ok {
		_spec.SetField(workerprofile.FieldMedicalCertDate, field.TypeOther, value)
	}
	if wpu.mutation.MedicalCertDateCleared() {
		_spec.ClearField(workerprofile.FieldMedicalCertDate, field.TypeOther)
	}
	if value, ok := wpu.mutation.MvrDueDate(); ok {
		_spec.SetField(workerprofile.FieldMvrDueDate, field.TypeOther, value)
	}
	if wpu.mutation.MvrDueDateCleared() {
		_spec.ClearField(workerprofile.FieldMvrDueDate, field.TypeOther)
	}
	if wpu.mutation.OrganizationCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   workerprofile.OrganizationTable,
			Columns: []string{workerprofile.OrganizationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(organization.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := wpu.mutation.OrganizationIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   workerprofile.OrganizationTable,
			Columns: []string{workerprofile.OrganizationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(organization.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if wpu.mutation.StateCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   workerprofile.StateTable,
			Columns: []string{workerprofile.StateColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(usstate.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := wpu.mutation.StateIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   workerprofile.StateTable,
			Columns: []string{workerprofile.StateColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(usstate.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_spec.AddModifiers(wpu.modifiers...)
	if n, err = sqlgraph.UpdateNodes(ctx, wpu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{workerprofile.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	wpu.mutation.done = true
	return n, nil
}

// WorkerProfileUpdateOne is the builder for updating a single WorkerProfile entity.
type WorkerProfileUpdateOne struct {
	config
	fields    []string
	hooks     []Hook
	mutation  *WorkerProfileMutation
	modifiers []func(*sql.UpdateBuilder)
}

// SetOrganizationID sets the "organization_id" field.
func (wpuo *WorkerProfileUpdateOne) SetOrganizationID(u uuid.UUID) *WorkerProfileUpdateOne {
	wpuo.mutation.SetOrganizationID(u)
	return wpuo
}

// SetNillableOrganizationID sets the "organization_id" field if the given value is not nil.
func (wpuo *WorkerProfileUpdateOne) SetNillableOrganizationID(u *uuid.UUID) *WorkerProfileUpdateOne {
	if u != nil {
		wpuo.SetOrganizationID(*u)
	}
	return wpuo
}

// SetUpdatedAt sets the "updated_at" field.
func (wpuo *WorkerProfileUpdateOne) SetUpdatedAt(t time.Time) *WorkerProfileUpdateOne {
	wpuo.mutation.SetUpdatedAt(t)
	return wpuo
}

// SetVersion sets the "version" field.
func (wpuo *WorkerProfileUpdateOne) SetVersion(i int) *WorkerProfileUpdateOne {
	wpuo.mutation.ResetVersion()
	wpuo.mutation.SetVersion(i)
	return wpuo
}

// SetNillableVersion sets the "version" field if the given value is not nil.
func (wpuo *WorkerProfileUpdateOne) SetNillableVersion(i *int) *WorkerProfileUpdateOne {
	if i != nil {
		wpuo.SetVersion(*i)
	}
	return wpuo
}

// AddVersion adds i to the "version" field.
func (wpuo *WorkerProfileUpdateOne) AddVersion(i int) *WorkerProfileUpdateOne {
	wpuo.mutation.AddVersion(i)
	return wpuo
}

// SetRace sets the "race" field.
func (wpuo *WorkerProfileUpdateOne) SetRace(s string) *WorkerProfileUpdateOne {
	wpuo.mutation.SetRace(s)
	return wpuo
}

// SetNillableRace sets the "race" field if the given value is not nil.
func (wpuo *WorkerProfileUpdateOne) SetNillableRace(s *string) *WorkerProfileUpdateOne {
	if s != nil {
		wpuo.SetRace(*s)
	}
	return wpuo
}

// ClearRace clears the value of the "race" field.
func (wpuo *WorkerProfileUpdateOne) ClearRace() *WorkerProfileUpdateOne {
	wpuo.mutation.ClearRace()
	return wpuo
}

// SetSex sets the "sex" field.
func (wpuo *WorkerProfileUpdateOne) SetSex(s string) *WorkerProfileUpdateOne {
	wpuo.mutation.SetSex(s)
	return wpuo
}

// SetNillableSex sets the "sex" field if the given value is not nil.
func (wpuo *WorkerProfileUpdateOne) SetNillableSex(s *string) *WorkerProfileUpdateOne {
	if s != nil {
		wpuo.SetSex(*s)
	}
	return wpuo
}

// ClearSex clears the value of the "sex" field.
func (wpuo *WorkerProfileUpdateOne) ClearSex() *WorkerProfileUpdateOne {
	wpuo.mutation.ClearSex()
	return wpuo
}

// SetDateOfBirth sets the "date_of_birth" field.
func (wpuo *WorkerProfileUpdateOne) SetDateOfBirth(pg *pgtype.Date) *WorkerProfileUpdateOne {
	wpuo.mutation.SetDateOfBirth(pg)
	return wpuo
}

// ClearDateOfBirth clears the value of the "date_of_birth" field.
func (wpuo *WorkerProfileUpdateOne) ClearDateOfBirth() *WorkerProfileUpdateOne {
	wpuo.mutation.ClearDateOfBirth()
	return wpuo
}

// SetLicenseNumber sets the "license_number" field.
func (wpuo *WorkerProfileUpdateOne) SetLicenseNumber(s string) *WorkerProfileUpdateOne {
	wpuo.mutation.SetLicenseNumber(s)
	return wpuo
}

// SetNillableLicenseNumber sets the "license_number" field if the given value is not nil.
func (wpuo *WorkerProfileUpdateOne) SetNillableLicenseNumber(s *string) *WorkerProfileUpdateOne {
	if s != nil {
		wpuo.SetLicenseNumber(*s)
	}
	return wpuo
}

// SetLicenseStateID sets the "license_state_id" field.
func (wpuo *WorkerProfileUpdateOne) SetLicenseStateID(u uuid.UUID) *WorkerProfileUpdateOne {
	wpuo.mutation.SetLicenseStateID(u)
	return wpuo
}

// SetNillableLicenseStateID sets the "license_state_id" field if the given value is not nil.
func (wpuo *WorkerProfileUpdateOne) SetNillableLicenseStateID(u *uuid.UUID) *WorkerProfileUpdateOne {
	if u != nil {
		wpuo.SetLicenseStateID(*u)
	}
	return wpuo
}

// SetLicenseExpirationDate sets the "license_expiration_date" field.
func (wpuo *WorkerProfileUpdateOne) SetLicenseExpirationDate(pg *pgtype.Date) *WorkerProfileUpdateOne {
	wpuo.mutation.SetLicenseExpirationDate(pg)
	return wpuo
}

// ClearLicenseExpirationDate clears the value of the "license_expiration_date" field.
func (wpuo *WorkerProfileUpdateOne) ClearLicenseExpirationDate() *WorkerProfileUpdateOne {
	wpuo.mutation.ClearLicenseExpirationDate()
	return wpuo
}

// SetEndorsements sets the "endorsements" field.
func (wpuo *WorkerProfileUpdateOne) SetEndorsements(w workerprofile.Endorsements) *WorkerProfileUpdateOne {
	wpuo.mutation.SetEndorsements(w)
	return wpuo
}

// SetNillableEndorsements sets the "endorsements" field if the given value is not nil.
func (wpuo *WorkerProfileUpdateOne) SetNillableEndorsements(w *workerprofile.Endorsements) *WorkerProfileUpdateOne {
	if w != nil {
		wpuo.SetEndorsements(*w)
	}
	return wpuo
}

// ClearEndorsements clears the value of the "endorsements" field.
func (wpuo *WorkerProfileUpdateOne) ClearEndorsements() *WorkerProfileUpdateOne {
	wpuo.mutation.ClearEndorsements()
	return wpuo
}

// SetHazmatExpirationDate sets the "hazmat_expiration_date" field.
func (wpuo *WorkerProfileUpdateOne) SetHazmatExpirationDate(pg *pgtype.Date) *WorkerProfileUpdateOne {
	wpuo.mutation.SetHazmatExpirationDate(pg)
	return wpuo
}

// ClearHazmatExpirationDate clears the value of the "hazmat_expiration_date" field.
func (wpuo *WorkerProfileUpdateOne) ClearHazmatExpirationDate() *WorkerProfileUpdateOne {
	wpuo.mutation.ClearHazmatExpirationDate()
	return wpuo
}

// SetHireDate sets the "hire_date" field.
func (wpuo *WorkerProfileUpdateOne) SetHireDate(pg *pgtype.Date) *WorkerProfileUpdateOne {
	wpuo.mutation.SetHireDate(pg)
	return wpuo
}

// ClearHireDate clears the value of the "hire_date" field.
func (wpuo *WorkerProfileUpdateOne) ClearHireDate() *WorkerProfileUpdateOne {
	wpuo.mutation.ClearHireDate()
	return wpuo
}

// SetTerminationDate sets the "termination_date" field.
func (wpuo *WorkerProfileUpdateOne) SetTerminationDate(pg *pgtype.Date) *WorkerProfileUpdateOne {
	wpuo.mutation.SetTerminationDate(pg)
	return wpuo
}

// ClearTerminationDate clears the value of the "termination_date" field.
func (wpuo *WorkerProfileUpdateOne) ClearTerminationDate() *WorkerProfileUpdateOne {
	wpuo.mutation.ClearTerminationDate()
	return wpuo
}

// SetPhysicalDueDate sets the "physical_due_date" field.
func (wpuo *WorkerProfileUpdateOne) SetPhysicalDueDate(pg *pgtype.Date) *WorkerProfileUpdateOne {
	wpuo.mutation.SetPhysicalDueDate(pg)
	return wpuo
}

// ClearPhysicalDueDate clears the value of the "physical_due_date" field.
func (wpuo *WorkerProfileUpdateOne) ClearPhysicalDueDate() *WorkerProfileUpdateOne {
	wpuo.mutation.ClearPhysicalDueDate()
	return wpuo
}

// SetMedicalCertDate sets the "medical_cert_date" field.
func (wpuo *WorkerProfileUpdateOne) SetMedicalCertDate(pg *pgtype.Date) *WorkerProfileUpdateOne {
	wpuo.mutation.SetMedicalCertDate(pg)
	return wpuo
}

// ClearMedicalCertDate clears the value of the "medical_cert_date" field.
func (wpuo *WorkerProfileUpdateOne) ClearMedicalCertDate() *WorkerProfileUpdateOne {
	wpuo.mutation.ClearMedicalCertDate()
	return wpuo
}

// SetMvrDueDate sets the "mvr_due_date" field.
func (wpuo *WorkerProfileUpdateOne) SetMvrDueDate(pg *pgtype.Date) *WorkerProfileUpdateOne {
	wpuo.mutation.SetMvrDueDate(pg)
	return wpuo
}

// ClearMvrDueDate clears the value of the "mvr_due_date" field.
func (wpuo *WorkerProfileUpdateOne) ClearMvrDueDate() *WorkerProfileUpdateOne {
	wpuo.mutation.ClearMvrDueDate()
	return wpuo
}

// SetOrganization sets the "organization" edge to the Organization entity.
func (wpuo *WorkerProfileUpdateOne) SetOrganization(o *Organization) *WorkerProfileUpdateOne {
	return wpuo.SetOrganizationID(o.ID)
}

// SetStateID sets the "state" edge to the UsState entity by ID.
func (wpuo *WorkerProfileUpdateOne) SetStateID(id uuid.UUID) *WorkerProfileUpdateOne {
	wpuo.mutation.SetStateID(id)
	return wpuo
}

// SetState sets the "state" edge to the UsState entity.
func (wpuo *WorkerProfileUpdateOne) SetState(u *UsState) *WorkerProfileUpdateOne {
	return wpuo.SetStateID(u.ID)
}

// Mutation returns the WorkerProfileMutation object of the builder.
func (wpuo *WorkerProfileUpdateOne) Mutation() *WorkerProfileMutation {
	return wpuo.mutation
}

// ClearOrganization clears the "organization" edge to the Organization entity.
func (wpuo *WorkerProfileUpdateOne) ClearOrganization() *WorkerProfileUpdateOne {
	wpuo.mutation.ClearOrganization()
	return wpuo
}

// ClearState clears the "state" edge to the UsState entity.
func (wpuo *WorkerProfileUpdateOne) ClearState() *WorkerProfileUpdateOne {
	wpuo.mutation.ClearState()
	return wpuo
}

// Where appends a list predicates to the WorkerProfileUpdate builder.
func (wpuo *WorkerProfileUpdateOne) Where(ps ...predicate.WorkerProfile) *WorkerProfileUpdateOne {
	wpuo.mutation.Where(ps...)
	return wpuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (wpuo *WorkerProfileUpdateOne) Select(field string, fields ...string) *WorkerProfileUpdateOne {
	wpuo.fields = append([]string{field}, fields...)
	return wpuo
}

// Save executes the query and returns the updated WorkerProfile entity.
func (wpuo *WorkerProfileUpdateOne) Save(ctx context.Context) (*WorkerProfile, error) {
	if err := wpuo.defaults(); err != nil {
		return nil, err
	}
	return withHooks(ctx, wpuo.sqlSave, wpuo.mutation, wpuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (wpuo *WorkerProfileUpdateOne) SaveX(ctx context.Context) *WorkerProfile {
	node, err := wpuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (wpuo *WorkerProfileUpdateOne) Exec(ctx context.Context) error {
	_, err := wpuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (wpuo *WorkerProfileUpdateOne) ExecX(ctx context.Context) {
	if err := wpuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (wpuo *WorkerProfileUpdateOne) defaults() error {
	if _, ok := wpuo.mutation.UpdatedAt(); !ok {
		if workerprofile.UpdateDefaultUpdatedAt == nil {
			return fmt.Errorf("ent: uninitialized workerprofile.UpdateDefaultUpdatedAt (forgotten import ent/runtime?)")
		}
		v := workerprofile.UpdateDefaultUpdatedAt()
		wpuo.mutation.SetUpdatedAt(v)
	}
	return nil
}

// check runs all checks and user-defined validators on the builder.
func (wpuo *WorkerProfileUpdateOne) check() error {
	if v, ok := wpuo.mutation.LicenseNumber(); ok {
		if err := workerprofile.LicenseNumberValidator(v); err != nil {
			return &ValidationError{Name: "license_number", err: fmt.Errorf(`ent: validator failed for field "WorkerProfile.license_number": %w`, err)}
		}
	}
	if v, ok := wpuo.mutation.Endorsements(); ok {
		if err := workerprofile.EndorsementsValidator(v); err != nil {
			return &ValidationError{Name: "endorsements", err: fmt.Errorf(`ent: validator failed for field "WorkerProfile.endorsements": %w`, err)}
		}
	}
	if _, ok := wpuo.mutation.BusinessUnitID(); wpuo.mutation.BusinessUnitCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "WorkerProfile.business_unit"`)
	}
	if _, ok := wpuo.mutation.OrganizationID(); wpuo.mutation.OrganizationCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "WorkerProfile.organization"`)
	}
	if _, ok := wpuo.mutation.WorkerID(); wpuo.mutation.WorkerCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "WorkerProfile.worker"`)
	}
	if _, ok := wpuo.mutation.StateID(); wpuo.mutation.StateCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "WorkerProfile.state"`)
	}
	return nil
}

// Modify adds a statement modifier for attaching custom logic to the UPDATE statement.
func (wpuo *WorkerProfileUpdateOne) Modify(modifiers ...func(u *sql.UpdateBuilder)) *WorkerProfileUpdateOne {
	wpuo.modifiers = append(wpuo.modifiers, modifiers...)
	return wpuo
}

func (wpuo *WorkerProfileUpdateOne) sqlSave(ctx context.Context) (_node *WorkerProfile, err error) {
	if err := wpuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(workerprofile.Table, workerprofile.Columns, sqlgraph.NewFieldSpec(workerprofile.FieldID, field.TypeUUID))
	id, ok := wpuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "WorkerProfile.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := wpuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, workerprofile.FieldID)
		for _, f := range fields {
			if !workerprofile.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != workerprofile.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := wpuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := wpuo.mutation.UpdatedAt(); ok {
		_spec.SetField(workerprofile.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := wpuo.mutation.Version(); ok {
		_spec.SetField(workerprofile.FieldVersion, field.TypeInt, value)
	}
	if value, ok := wpuo.mutation.AddedVersion(); ok {
		_spec.AddField(workerprofile.FieldVersion, field.TypeInt, value)
	}
	if value, ok := wpuo.mutation.Race(); ok {
		_spec.SetField(workerprofile.FieldRace, field.TypeString, value)
	}
	if wpuo.mutation.RaceCleared() {
		_spec.ClearField(workerprofile.FieldRace, field.TypeString)
	}
	if value, ok := wpuo.mutation.Sex(); ok {
		_spec.SetField(workerprofile.FieldSex, field.TypeString, value)
	}
	if wpuo.mutation.SexCleared() {
		_spec.ClearField(workerprofile.FieldSex, field.TypeString)
	}
	if value, ok := wpuo.mutation.DateOfBirth(); ok {
		_spec.SetField(workerprofile.FieldDateOfBirth, field.TypeOther, value)
	}
	if wpuo.mutation.DateOfBirthCleared() {
		_spec.ClearField(workerprofile.FieldDateOfBirth, field.TypeOther)
	}
	if value, ok := wpuo.mutation.LicenseNumber(); ok {
		_spec.SetField(workerprofile.FieldLicenseNumber, field.TypeString, value)
	}
	if value, ok := wpuo.mutation.LicenseExpirationDate(); ok {
		_spec.SetField(workerprofile.FieldLicenseExpirationDate, field.TypeOther, value)
	}
	if wpuo.mutation.LicenseExpirationDateCleared() {
		_spec.ClearField(workerprofile.FieldLicenseExpirationDate, field.TypeOther)
	}
	if value, ok := wpuo.mutation.Endorsements(); ok {
		_spec.SetField(workerprofile.FieldEndorsements, field.TypeEnum, value)
	}
	if wpuo.mutation.EndorsementsCleared() {
		_spec.ClearField(workerprofile.FieldEndorsements, field.TypeEnum)
	}
	if value, ok := wpuo.mutation.HazmatExpirationDate(); ok {
		_spec.SetField(workerprofile.FieldHazmatExpirationDate, field.TypeOther, value)
	}
	if wpuo.mutation.HazmatExpirationDateCleared() {
		_spec.ClearField(workerprofile.FieldHazmatExpirationDate, field.TypeOther)
	}
	if value, ok := wpuo.mutation.HireDate(); ok {
		_spec.SetField(workerprofile.FieldHireDate, field.TypeOther, value)
	}
	if wpuo.mutation.HireDateCleared() {
		_spec.ClearField(workerprofile.FieldHireDate, field.TypeOther)
	}
	if value, ok := wpuo.mutation.TerminationDate(); ok {
		_spec.SetField(workerprofile.FieldTerminationDate, field.TypeOther, value)
	}
	if wpuo.mutation.TerminationDateCleared() {
		_spec.ClearField(workerprofile.FieldTerminationDate, field.TypeOther)
	}
	if value, ok := wpuo.mutation.PhysicalDueDate(); ok {
		_spec.SetField(workerprofile.FieldPhysicalDueDate, field.TypeOther, value)
	}
	if wpuo.mutation.PhysicalDueDateCleared() {
		_spec.ClearField(workerprofile.FieldPhysicalDueDate, field.TypeOther)
	}
	if value, ok := wpuo.mutation.MedicalCertDate(); ok {
		_spec.SetField(workerprofile.FieldMedicalCertDate, field.TypeOther, value)
	}
	if wpuo.mutation.MedicalCertDateCleared() {
		_spec.ClearField(workerprofile.FieldMedicalCertDate, field.TypeOther)
	}
	if value, ok := wpuo.mutation.MvrDueDate(); ok {
		_spec.SetField(workerprofile.FieldMvrDueDate, field.TypeOther, value)
	}
	if wpuo.mutation.MvrDueDateCleared() {
		_spec.ClearField(workerprofile.FieldMvrDueDate, field.TypeOther)
	}
	if wpuo.mutation.OrganizationCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   workerprofile.OrganizationTable,
			Columns: []string{workerprofile.OrganizationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(organization.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := wpuo.mutation.OrganizationIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   workerprofile.OrganizationTable,
			Columns: []string{workerprofile.OrganizationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(organization.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if wpuo.mutation.StateCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   workerprofile.StateTable,
			Columns: []string{workerprofile.StateColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(usstate.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := wpuo.mutation.StateIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   workerprofile.StateTable,
			Columns: []string{workerprofile.StateColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(usstate.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_spec.AddModifiers(wpuo.modifiers...)
	_node = &WorkerProfile{config: wpuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, wpuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{workerprofile.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	wpuo.mutation.done = true
	return _node, nil
}
