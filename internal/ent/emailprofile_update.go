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
	"github.com/emoss08/trenova/internal/ent/emailprofile"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/predicate"
	"github.com/google/uuid"
)

// EmailProfileUpdate is the builder for updating EmailProfile entities.
type EmailProfileUpdate struct {
	config
	hooks     []Hook
	mutation  *EmailProfileMutation
	modifiers []func(*sql.UpdateBuilder)
}

// Where appends a list predicates to the EmailProfileUpdate builder.
func (epu *EmailProfileUpdate) Where(ps ...predicate.EmailProfile) *EmailProfileUpdate {
	epu.mutation.Where(ps...)
	return epu
}

// SetOrganizationID sets the "organization_id" field.
func (epu *EmailProfileUpdate) SetOrganizationID(u uuid.UUID) *EmailProfileUpdate {
	epu.mutation.SetOrganizationID(u)
	return epu
}

// SetNillableOrganizationID sets the "organization_id" field if the given value is not nil.
func (epu *EmailProfileUpdate) SetNillableOrganizationID(u *uuid.UUID) *EmailProfileUpdate {
	if u != nil {
		epu.SetOrganizationID(*u)
	}
	return epu
}

// SetUpdatedAt sets the "updated_at" field.
func (epu *EmailProfileUpdate) SetUpdatedAt(t time.Time) *EmailProfileUpdate {
	epu.mutation.SetUpdatedAt(t)
	return epu
}

// SetVersion sets the "version" field.
func (epu *EmailProfileUpdate) SetVersion(i int) *EmailProfileUpdate {
	epu.mutation.ResetVersion()
	epu.mutation.SetVersion(i)
	return epu
}

// SetNillableVersion sets the "version" field if the given value is not nil.
func (epu *EmailProfileUpdate) SetNillableVersion(i *int) *EmailProfileUpdate {
	if i != nil {
		epu.SetVersion(*i)
	}
	return epu
}

// AddVersion adds i to the "version" field.
func (epu *EmailProfileUpdate) AddVersion(i int) *EmailProfileUpdate {
	epu.mutation.AddVersion(i)
	return epu
}

// SetName sets the "name" field.
func (epu *EmailProfileUpdate) SetName(s string) *EmailProfileUpdate {
	epu.mutation.SetName(s)
	return epu
}

// SetNillableName sets the "name" field if the given value is not nil.
func (epu *EmailProfileUpdate) SetNillableName(s *string) *EmailProfileUpdate {
	if s != nil {
		epu.SetName(*s)
	}
	return epu
}

// SetEmail sets the "email" field.
func (epu *EmailProfileUpdate) SetEmail(s string) *EmailProfileUpdate {
	epu.mutation.SetEmail(s)
	return epu
}

// SetNillableEmail sets the "email" field if the given value is not nil.
func (epu *EmailProfileUpdate) SetNillableEmail(s *string) *EmailProfileUpdate {
	if s != nil {
		epu.SetEmail(*s)
	}
	return epu
}

// SetProtocol sets the "protocol" field.
func (epu *EmailProfileUpdate) SetProtocol(e emailprofile.Protocol) *EmailProfileUpdate {
	epu.mutation.SetProtocol(e)
	return epu
}

// SetNillableProtocol sets the "protocol" field if the given value is not nil.
func (epu *EmailProfileUpdate) SetNillableProtocol(e *emailprofile.Protocol) *EmailProfileUpdate {
	if e != nil {
		epu.SetProtocol(*e)
	}
	return epu
}

// ClearProtocol clears the value of the "protocol" field.
func (epu *EmailProfileUpdate) ClearProtocol() *EmailProfileUpdate {
	epu.mutation.ClearProtocol()
	return epu
}

// SetHost sets the "host" field.
func (epu *EmailProfileUpdate) SetHost(s string) *EmailProfileUpdate {
	epu.mutation.SetHost(s)
	return epu
}

// SetNillableHost sets the "host" field if the given value is not nil.
func (epu *EmailProfileUpdate) SetNillableHost(s *string) *EmailProfileUpdate {
	if s != nil {
		epu.SetHost(*s)
	}
	return epu
}

// ClearHost clears the value of the "host" field.
func (epu *EmailProfileUpdate) ClearHost() *EmailProfileUpdate {
	epu.mutation.ClearHost()
	return epu
}

// SetPort sets the "port" field.
func (epu *EmailProfileUpdate) SetPort(i int16) *EmailProfileUpdate {
	epu.mutation.ResetPort()
	epu.mutation.SetPort(i)
	return epu
}

// SetNillablePort sets the "port" field if the given value is not nil.
func (epu *EmailProfileUpdate) SetNillablePort(i *int16) *EmailProfileUpdate {
	if i != nil {
		epu.SetPort(*i)
	}
	return epu
}

// AddPort adds i to the "port" field.
func (epu *EmailProfileUpdate) AddPort(i int16) *EmailProfileUpdate {
	epu.mutation.AddPort(i)
	return epu
}

// ClearPort clears the value of the "port" field.
func (epu *EmailProfileUpdate) ClearPort() *EmailProfileUpdate {
	epu.mutation.ClearPort()
	return epu
}

// SetUsername sets the "username" field.
func (epu *EmailProfileUpdate) SetUsername(s string) *EmailProfileUpdate {
	epu.mutation.SetUsername(s)
	return epu
}

// SetNillableUsername sets the "username" field if the given value is not nil.
func (epu *EmailProfileUpdate) SetNillableUsername(s *string) *EmailProfileUpdate {
	if s != nil {
		epu.SetUsername(*s)
	}
	return epu
}

// ClearUsername clears the value of the "username" field.
func (epu *EmailProfileUpdate) ClearUsername() *EmailProfileUpdate {
	epu.mutation.ClearUsername()
	return epu
}

// SetPassword sets the "password" field.
func (epu *EmailProfileUpdate) SetPassword(s string) *EmailProfileUpdate {
	epu.mutation.SetPassword(s)
	return epu
}

// SetNillablePassword sets the "password" field if the given value is not nil.
func (epu *EmailProfileUpdate) SetNillablePassword(s *string) *EmailProfileUpdate {
	if s != nil {
		epu.SetPassword(*s)
	}
	return epu
}

// ClearPassword clears the value of the "password" field.
func (epu *EmailProfileUpdate) ClearPassword() *EmailProfileUpdate {
	epu.mutation.ClearPassword()
	return epu
}

// SetIsDefault sets the "is_default" field.
func (epu *EmailProfileUpdate) SetIsDefault(b bool) *EmailProfileUpdate {
	epu.mutation.SetIsDefault(b)
	return epu
}

// SetNillableIsDefault sets the "is_default" field if the given value is not nil.
func (epu *EmailProfileUpdate) SetNillableIsDefault(b *bool) *EmailProfileUpdate {
	if b != nil {
		epu.SetIsDefault(*b)
	}
	return epu
}

// SetOrganization sets the "organization" edge to the Organization entity.
func (epu *EmailProfileUpdate) SetOrganization(o *Organization) *EmailProfileUpdate {
	return epu.SetOrganizationID(o.ID)
}

// Mutation returns the EmailProfileMutation object of the builder.
func (epu *EmailProfileUpdate) Mutation() *EmailProfileMutation {
	return epu.mutation
}

// ClearOrganization clears the "organization" edge to the Organization entity.
func (epu *EmailProfileUpdate) ClearOrganization() *EmailProfileUpdate {
	epu.mutation.ClearOrganization()
	return epu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (epu *EmailProfileUpdate) Save(ctx context.Context) (int, error) {
	if err := epu.defaults(); err != nil {
		return 0, err
	}
	return withHooks(ctx, epu.sqlSave, epu.mutation, epu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (epu *EmailProfileUpdate) SaveX(ctx context.Context) int {
	affected, err := epu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (epu *EmailProfileUpdate) Exec(ctx context.Context) error {
	_, err := epu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (epu *EmailProfileUpdate) ExecX(ctx context.Context) {
	if err := epu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (epu *EmailProfileUpdate) defaults() error {
	if _, ok := epu.mutation.UpdatedAt(); !ok {
		if emailprofile.UpdateDefaultUpdatedAt == nil {
			return fmt.Errorf("ent: uninitialized emailprofile.UpdateDefaultUpdatedAt (forgotten import ent/runtime?)")
		}
		v := emailprofile.UpdateDefaultUpdatedAt()
		epu.mutation.SetUpdatedAt(v)
	}
	return nil
}

// check runs all checks and user-defined validators on the builder.
func (epu *EmailProfileUpdate) check() error {
	if v, ok := epu.mutation.Name(); ok {
		if err := emailprofile.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "EmailProfile.name": %w`, err)}
		}
	}
	if v, ok := epu.mutation.Email(); ok {
		if err := emailprofile.EmailValidator(v); err != nil {
			return &ValidationError{Name: "email", err: fmt.Errorf(`ent: validator failed for field "EmailProfile.email": %w`, err)}
		}
	}
	if v, ok := epu.mutation.Protocol(); ok {
		if err := emailprofile.ProtocolValidator(v); err != nil {
			return &ValidationError{Name: "protocol", err: fmt.Errorf(`ent: validator failed for field "EmailProfile.protocol": %w`, err)}
		}
	}
	if _, ok := epu.mutation.BusinessUnitID(); epu.mutation.BusinessUnitCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "EmailProfile.business_unit"`)
	}
	if _, ok := epu.mutation.OrganizationID(); epu.mutation.OrganizationCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "EmailProfile.organization"`)
	}
	return nil
}

// Modify adds a statement modifier for attaching custom logic to the UPDATE statement.
func (epu *EmailProfileUpdate) Modify(modifiers ...func(u *sql.UpdateBuilder)) *EmailProfileUpdate {
	epu.modifiers = append(epu.modifiers, modifiers...)
	return epu
}

func (epu *EmailProfileUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := epu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(emailprofile.Table, emailprofile.Columns, sqlgraph.NewFieldSpec(emailprofile.FieldID, field.TypeUUID))
	if ps := epu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := epu.mutation.UpdatedAt(); ok {
		_spec.SetField(emailprofile.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := epu.mutation.Version(); ok {
		_spec.SetField(emailprofile.FieldVersion, field.TypeInt, value)
	}
	if value, ok := epu.mutation.AddedVersion(); ok {
		_spec.AddField(emailprofile.FieldVersion, field.TypeInt, value)
	}
	if value, ok := epu.mutation.Name(); ok {
		_spec.SetField(emailprofile.FieldName, field.TypeString, value)
	}
	if value, ok := epu.mutation.Email(); ok {
		_spec.SetField(emailprofile.FieldEmail, field.TypeString, value)
	}
	if value, ok := epu.mutation.Protocol(); ok {
		_spec.SetField(emailprofile.FieldProtocol, field.TypeEnum, value)
	}
	if epu.mutation.ProtocolCleared() {
		_spec.ClearField(emailprofile.FieldProtocol, field.TypeEnum)
	}
	if value, ok := epu.mutation.Host(); ok {
		_spec.SetField(emailprofile.FieldHost, field.TypeString, value)
	}
	if epu.mutation.HostCleared() {
		_spec.ClearField(emailprofile.FieldHost, field.TypeString)
	}
	if value, ok := epu.mutation.Port(); ok {
		_spec.SetField(emailprofile.FieldPort, field.TypeInt16, value)
	}
	if value, ok := epu.mutation.AddedPort(); ok {
		_spec.AddField(emailprofile.FieldPort, field.TypeInt16, value)
	}
	if epu.mutation.PortCleared() {
		_spec.ClearField(emailprofile.FieldPort, field.TypeInt16)
	}
	if value, ok := epu.mutation.Username(); ok {
		_spec.SetField(emailprofile.FieldUsername, field.TypeString, value)
	}
	if epu.mutation.UsernameCleared() {
		_spec.ClearField(emailprofile.FieldUsername, field.TypeString)
	}
	if value, ok := epu.mutation.Password(); ok {
		_spec.SetField(emailprofile.FieldPassword, field.TypeString, value)
	}
	if epu.mutation.PasswordCleared() {
		_spec.ClearField(emailprofile.FieldPassword, field.TypeString)
	}
	if value, ok := epu.mutation.IsDefault(); ok {
		_spec.SetField(emailprofile.FieldIsDefault, field.TypeBool, value)
	}
	if epu.mutation.OrganizationCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   emailprofile.OrganizationTable,
			Columns: []string{emailprofile.OrganizationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(organization.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := epu.mutation.OrganizationIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   emailprofile.OrganizationTable,
			Columns: []string{emailprofile.OrganizationColumn},
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
	_spec.AddModifiers(epu.modifiers...)
	if n, err = sqlgraph.UpdateNodes(ctx, epu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{emailprofile.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	epu.mutation.done = true
	return n, nil
}

// EmailProfileUpdateOne is the builder for updating a single EmailProfile entity.
type EmailProfileUpdateOne struct {
	config
	fields    []string
	hooks     []Hook
	mutation  *EmailProfileMutation
	modifiers []func(*sql.UpdateBuilder)
}

// SetOrganizationID sets the "organization_id" field.
func (epuo *EmailProfileUpdateOne) SetOrganizationID(u uuid.UUID) *EmailProfileUpdateOne {
	epuo.mutation.SetOrganizationID(u)
	return epuo
}

// SetNillableOrganizationID sets the "organization_id" field if the given value is not nil.
func (epuo *EmailProfileUpdateOne) SetNillableOrganizationID(u *uuid.UUID) *EmailProfileUpdateOne {
	if u != nil {
		epuo.SetOrganizationID(*u)
	}
	return epuo
}

// SetUpdatedAt sets the "updated_at" field.
func (epuo *EmailProfileUpdateOne) SetUpdatedAt(t time.Time) *EmailProfileUpdateOne {
	epuo.mutation.SetUpdatedAt(t)
	return epuo
}

// SetVersion sets the "version" field.
func (epuo *EmailProfileUpdateOne) SetVersion(i int) *EmailProfileUpdateOne {
	epuo.mutation.ResetVersion()
	epuo.mutation.SetVersion(i)
	return epuo
}

// SetNillableVersion sets the "version" field if the given value is not nil.
func (epuo *EmailProfileUpdateOne) SetNillableVersion(i *int) *EmailProfileUpdateOne {
	if i != nil {
		epuo.SetVersion(*i)
	}
	return epuo
}

// AddVersion adds i to the "version" field.
func (epuo *EmailProfileUpdateOne) AddVersion(i int) *EmailProfileUpdateOne {
	epuo.mutation.AddVersion(i)
	return epuo
}

// SetName sets the "name" field.
func (epuo *EmailProfileUpdateOne) SetName(s string) *EmailProfileUpdateOne {
	epuo.mutation.SetName(s)
	return epuo
}

// SetNillableName sets the "name" field if the given value is not nil.
func (epuo *EmailProfileUpdateOne) SetNillableName(s *string) *EmailProfileUpdateOne {
	if s != nil {
		epuo.SetName(*s)
	}
	return epuo
}

// SetEmail sets the "email" field.
func (epuo *EmailProfileUpdateOne) SetEmail(s string) *EmailProfileUpdateOne {
	epuo.mutation.SetEmail(s)
	return epuo
}

// SetNillableEmail sets the "email" field if the given value is not nil.
func (epuo *EmailProfileUpdateOne) SetNillableEmail(s *string) *EmailProfileUpdateOne {
	if s != nil {
		epuo.SetEmail(*s)
	}
	return epuo
}

// SetProtocol sets the "protocol" field.
func (epuo *EmailProfileUpdateOne) SetProtocol(e emailprofile.Protocol) *EmailProfileUpdateOne {
	epuo.mutation.SetProtocol(e)
	return epuo
}

// SetNillableProtocol sets the "protocol" field if the given value is not nil.
func (epuo *EmailProfileUpdateOne) SetNillableProtocol(e *emailprofile.Protocol) *EmailProfileUpdateOne {
	if e != nil {
		epuo.SetProtocol(*e)
	}
	return epuo
}

// ClearProtocol clears the value of the "protocol" field.
func (epuo *EmailProfileUpdateOne) ClearProtocol() *EmailProfileUpdateOne {
	epuo.mutation.ClearProtocol()
	return epuo
}

// SetHost sets the "host" field.
func (epuo *EmailProfileUpdateOne) SetHost(s string) *EmailProfileUpdateOne {
	epuo.mutation.SetHost(s)
	return epuo
}

// SetNillableHost sets the "host" field if the given value is not nil.
func (epuo *EmailProfileUpdateOne) SetNillableHost(s *string) *EmailProfileUpdateOne {
	if s != nil {
		epuo.SetHost(*s)
	}
	return epuo
}

// ClearHost clears the value of the "host" field.
func (epuo *EmailProfileUpdateOne) ClearHost() *EmailProfileUpdateOne {
	epuo.mutation.ClearHost()
	return epuo
}

// SetPort sets the "port" field.
func (epuo *EmailProfileUpdateOne) SetPort(i int16) *EmailProfileUpdateOne {
	epuo.mutation.ResetPort()
	epuo.mutation.SetPort(i)
	return epuo
}

// SetNillablePort sets the "port" field if the given value is not nil.
func (epuo *EmailProfileUpdateOne) SetNillablePort(i *int16) *EmailProfileUpdateOne {
	if i != nil {
		epuo.SetPort(*i)
	}
	return epuo
}

// AddPort adds i to the "port" field.
func (epuo *EmailProfileUpdateOne) AddPort(i int16) *EmailProfileUpdateOne {
	epuo.mutation.AddPort(i)
	return epuo
}

// ClearPort clears the value of the "port" field.
func (epuo *EmailProfileUpdateOne) ClearPort() *EmailProfileUpdateOne {
	epuo.mutation.ClearPort()
	return epuo
}

// SetUsername sets the "username" field.
func (epuo *EmailProfileUpdateOne) SetUsername(s string) *EmailProfileUpdateOne {
	epuo.mutation.SetUsername(s)
	return epuo
}

// SetNillableUsername sets the "username" field if the given value is not nil.
func (epuo *EmailProfileUpdateOne) SetNillableUsername(s *string) *EmailProfileUpdateOne {
	if s != nil {
		epuo.SetUsername(*s)
	}
	return epuo
}

// ClearUsername clears the value of the "username" field.
func (epuo *EmailProfileUpdateOne) ClearUsername() *EmailProfileUpdateOne {
	epuo.mutation.ClearUsername()
	return epuo
}

// SetPassword sets the "password" field.
func (epuo *EmailProfileUpdateOne) SetPassword(s string) *EmailProfileUpdateOne {
	epuo.mutation.SetPassword(s)
	return epuo
}

// SetNillablePassword sets the "password" field if the given value is not nil.
func (epuo *EmailProfileUpdateOne) SetNillablePassword(s *string) *EmailProfileUpdateOne {
	if s != nil {
		epuo.SetPassword(*s)
	}
	return epuo
}

// ClearPassword clears the value of the "password" field.
func (epuo *EmailProfileUpdateOne) ClearPassword() *EmailProfileUpdateOne {
	epuo.mutation.ClearPassword()
	return epuo
}

// SetIsDefault sets the "is_default" field.
func (epuo *EmailProfileUpdateOne) SetIsDefault(b bool) *EmailProfileUpdateOne {
	epuo.mutation.SetIsDefault(b)
	return epuo
}

// SetNillableIsDefault sets the "is_default" field if the given value is not nil.
func (epuo *EmailProfileUpdateOne) SetNillableIsDefault(b *bool) *EmailProfileUpdateOne {
	if b != nil {
		epuo.SetIsDefault(*b)
	}
	return epuo
}

// SetOrganization sets the "organization" edge to the Organization entity.
func (epuo *EmailProfileUpdateOne) SetOrganization(o *Organization) *EmailProfileUpdateOne {
	return epuo.SetOrganizationID(o.ID)
}

// Mutation returns the EmailProfileMutation object of the builder.
func (epuo *EmailProfileUpdateOne) Mutation() *EmailProfileMutation {
	return epuo.mutation
}

// ClearOrganization clears the "organization" edge to the Organization entity.
func (epuo *EmailProfileUpdateOne) ClearOrganization() *EmailProfileUpdateOne {
	epuo.mutation.ClearOrganization()
	return epuo
}

// Where appends a list predicates to the EmailProfileUpdate builder.
func (epuo *EmailProfileUpdateOne) Where(ps ...predicate.EmailProfile) *EmailProfileUpdateOne {
	epuo.mutation.Where(ps...)
	return epuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (epuo *EmailProfileUpdateOne) Select(field string, fields ...string) *EmailProfileUpdateOne {
	epuo.fields = append([]string{field}, fields...)
	return epuo
}

// Save executes the query and returns the updated EmailProfile entity.
func (epuo *EmailProfileUpdateOne) Save(ctx context.Context) (*EmailProfile, error) {
	if err := epuo.defaults(); err != nil {
		return nil, err
	}
	return withHooks(ctx, epuo.sqlSave, epuo.mutation, epuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (epuo *EmailProfileUpdateOne) SaveX(ctx context.Context) *EmailProfile {
	node, err := epuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (epuo *EmailProfileUpdateOne) Exec(ctx context.Context) error {
	_, err := epuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (epuo *EmailProfileUpdateOne) ExecX(ctx context.Context) {
	if err := epuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (epuo *EmailProfileUpdateOne) defaults() error {
	if _, ok := epuo.mutation.UpdatedAt(); !ok {
		if emailprofile.UpdateDefaultUpdatedAt == nil {
			return fmt.Errorf("ent: uninitialized emailprofile.UpdateDefaultUpdatedAt (forgotten import ent/runtime?)")
		}
		v := emailprofile.UpdateDefaultUpdatedAt()
		epuo.mutation.SetUpdatedAt(v)
	}
	return nil
}

// check runs all checks and user-defined validators on the builder.
func (epuo *EmailProfileUpdateOne) check() error {
	if v, ok := epuo.mutation.Name(); ok {
		if err := emailprofile.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "EmailProfile.name": %w`, err)}
		}
	}
	if v, ok := epuo.mutation.Email(); ok {
		if err := emailprofile.EmailValidator(v); err != nil {
			return &ValidationError{Name: "email", err: fmt.Errorf(`ent: validator failed for field "EmailProfile.email": %w`, err)}
		}
	}
	if v, ok := epuo.mutation.Protocol(); ok {
		if err := emailprofile.ProtocolValidator(v); err != nil {
			return &ValidationError{Name: "protocol", err: fmt.Errorf(`ent: validator failed for field "EmailProfile.protocol": %w`, err)}
		}
	}
	if _, ok := epuo.mutation.BusinessUnitID(); epuo.mutation.BusinessUnitCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "EmailProfile.business_unit"`)
	}
	if _, ok := epuo.mutation.OrganizationID(); epuo.mutation.OrganizationCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "EmailProfile.organization"`)
	}
	return nil
}

// Modify adds a statement modifier for attaching custom logic to the UPDATE statement.
func (epuo *EmailProfileUpdateOne) Modify(modifiers ...func(u *sql.UpdateBuilder)) *EmailProfileUpdateOne {
	epuo.modifiers = append(epuo.modifiers, modifiers...)
	return epuo
}

func (epuo *EmailProfileUpdateOne) sqlSave(ctx context.Context) (_node *EmailProfile, err error) {
	if err := epuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(emailprofile.Table, emailprofile.Columns, sqlgraph.NewFieldSpec(emailprofile.FieldID, field.TypeUUID))
	id, ok := epuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "EmailProfile.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := epuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, emailprofile.FieldID)
		for _, f := range fields {
			if !emailprofile.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != emailprofile.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := epuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := epuo.mutation.UpdatedAt(); ok {
		_spec.SetField(emailprofile.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := epuo.mutation.Version(); ok {
		_spec.SetField(emailprofile.FieldVersion, field.TypeInt, value)
	}
	if value, ok := epuo.mutation.AddedVersion(); ok {
		_spec.AddField(emailprofile.FieldVersion, field.TypeInt, value)
	}
	if value, ok := epuo.mutation.Name(); ok {
		_spec.SetField(emailprofile.FieldName, field.TypeString, value)
	}
	if value, ok := epuo.mutation.Email(); ok {
		_spec.SetField(emailprofile.FieldEmail, field.TypeString, value)
	}
	if value, ok := epuo.mutation.Protocol(); ok {
		_spec.SetField(emailprofile.FieldProtocol, field.TypeEnum, value)
	}
	if epuo.mutation.ProtocolCleared() {
		_spec.ClearField(emailprofile.FieldProtocol, field.TypeEnum)
	}
	if value, ok := epuo.mutation.Host(); ok {
		_spec.SetField(emailprofile.FieldHost, field.TypeString, value)
	}
	if epuo.mutation.HostCleared() {
		_spec.ClearField(emailprofile.FieldHost, field.TypeString)
	}
	if value, ok := epuo.mutation.Port(); ok {
		_spec.SetField(emailprofile.FieldPort, field.TypeInt16, value)
	}
	if value, ok := epuo.mutation.AddedPort(); ok {
		_spec.AddField(emailprofile.FieldPort, field.TypeInt16, value)
	}
	if epuo.mutation.PortCleared() {
		_spec.ClearField(emailprofile.FieldPort, field.TypeInt16)
	}
	if value, ok := epuo.mutation.Username(); ok {
		_spec.SetField(emailprofile.FieldUsername, field.TypeString, value)
	}
	if epuo.mutation.UsernameCleared() {
		_spec.ClearField(emailprofile.FieldUsername, field.TypeString)
	}
	if value, ok := epuo.mutation.Password(); ok {
		_spec.SetField(emailprofile.FieldPassword, field.TypeString, value)
	}
	if epuo.mutation.PasswordCleared() {
		_spec.ClearField(emailprofile.FieldPassword, field.TypeString)
	}
	if value, ok := epuo.mutation.IsDefault(); ok {
		_spec.SetField(emailprofile.FieldIsDefault, field.TypeBool, value)
	}
	if epuo.mutation.OrganizationCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   emailprofile.OrganizationTable,
			Columns: []string{emailprofile.OrganizationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(organization.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := epuo.mutation.OrganizationIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   emailprofile.OrganizationTable,
			Columns: []string{emailprofile.OrganizationColumn},
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
	_spec.AddModifiers(epuo.modifiers...)
	_node = &EmailProfile{config: epuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, epuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{emailprofile.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	epuo.mutation.done = true
	return _node, nil
}
