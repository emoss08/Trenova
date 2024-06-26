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
	"github.com/emoss08/trenova/internal/ent/customeremailprofile"
	"github.com/emoss08/trenova/internal/ent/emailprofile"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/predicate"
	"github.com/google/uuid"
)

// CustomerEmailProfileUpdate is the builder for updating CustomerEmailProfile entities.
type CustomerEmailProfileUpdate struct {
	config
	hooks     []Hook
	mutation  *CustomerEmailProfileMutation
	modifiers []func(*sql.UpdateBuilder)
}

// Where appends a list predicates to the CustomerEmailProfileUpdate builder.
func (cepu *CustomerEmailProfileUpdate) Where(ps ...predicate.CustomerEmailProfile) *CustomerEmailProfileUpdate {
	cepu.mutation.Where(ps...)
	return cepu
}

// SetOrganizationID sets the "organization_id" field.
func (cepu *CustomerEmailProfileUpdate) SetOrganizationID(u uuid.UUID) *CustomerEmailProfileUpdate {
	cepu.mutation.SetOrganizationID(u)
	return cepu
}

// SetNillableOrganizationID sets the "organization_id" field if the given value is not nil.
func (cepu *CustomerEmailProfileUpdate) SetNillableOrganizationID(u *uuid.UUID) *CustomerEmailProfileUpdate {
	if u != nil {
		cepu.SetOrganizationID(*u)
	}
	return cepu
}

// SetUpdatedAt sets the "updated_at" field.
func (cepu *CustomerEmailProfileUpdate) SetUpdatedAt(t time.Time) *CustomerEmailProfileUpdate {
	cepu.mutation.SetUpdatedAt(t)
	return cepu
}

// SetVersion sets the "version" field.
func (cepu *CustomerEmailProfileUpdate) SetVersion(i int) *CustomerEmailProfileUpdate {
	cepu.mutation.ResetVersion()
	cepu.mutation.SetVersion(i)
	return cepu
}

// SetNillableVersion sets the "version" field if the given value is not nil.
func (cepu *CustomerEmailProfileUpdate) SetNillableVersion(i *int) *CustomerEmailProfileUpdate {
	if i != nil {
		cepu.SetVersion(*i)
	}
	return cepu
}

// AddVersion adds i to the "version" field.
func (cepu *CustomerEmailProfileUpdate) AddVersion(i int) *CustomerEmailProfileUpdate {
	cepu.mutation.AddVersion(i)
	return cepu
}

// SetSubject sets the "subject" field.
func (cepu *CustomerEmailProfileUpdate) SetSubject(s string) *CustomerEmailProfileUpdate {
	cepu.mutation.SetSubject(s)
	return cepu
}

// SetNillableSubject sets the "subject" field if the given value is not nil.
func (cepu *CustomerEmailProfileUpdate) SetNillableSubject(s *string) *CustomerEmailProfileUpdate {
	if s != nil {
		cepu.SetSubject(*s)
	}
	return cepu
}

// ClearSubject clears the value of the "subject" field.
func (cepu *CustomerEmailProfileUpdate) ClearSubject() *CustomerEmailProfileUpdate {
	cepu.mutation.ClearSubject()
	return cepu
}

// SetEmailProfileID sets the "email_profile_id" field.
func (cepu *CustomerEmailProfileUpdate) SetEmailProfileID(u uuid.UUID) *CustomerEmailProfileUpdate {
	cepu.mutation.SetEmailProfileID(u)
	return cepu
}

// SetNillableEmailProfileID sets the "email_profile_id" field if the given value is not nil.
func (cepu *CustomerEmailProfileUpdate) SetNillableEmailProfileID(u *uuid.UUID) *CustomerEmailProfileUpdate {
	if u != nil {
		cepu.SetEmailProfileID(*u)
	}
	return cepu
}

// ClearEmailProfileID clears the value of the "email_profile_id" field.
func (cepu *CustomerEmailProfileUpdate) ClearEmailProfileID() *CustomerEmailProfileUpdate {
	cepu.mutation.ClearEmailProfileID()
	return cepu
}

// SetEmailRecipients sets the "email_recipients" field.
func (cepu *CustomerEmailProfileUpdate) SetEmailRecipients(s string) *CustomerEmailProfileUpdate {
	cepu.mutation.SetEmailRecipients(s)
	return cepu
}

// SetNillableEmailRecipients sets the "email_recipients" field if the given value is not nil.
func (cepu *CustomerEmailProfileUpdate) SetNillableEmailRecipients(s *string) *CustomerEmailProfileUpdate {
	if s != nil {
		cepu.SetEmailRecipients(*s)
	}
	return cepu
}

// SetEmailCcRecipients sets the "email_cc_recipients" field.
func (cepu *CustomerEmailProfileUpdate) SetEmailCcRecipients(s string) *CustomerEmailProfileUpdate {
	cepu.mutation.SetEmailCcRecipients(s)
	return cepu
}

// SetNillableEmailCcRecipients sets the "email_cc_recipients" field if the given value is not nil.
func (cepu *CustomerEmailProfileUpdate) SetNillableEmailCcRecipients(s *string) *CustomerEmailProfileUpdate {
	if s != nil {
		cepu.SetEmailCcRecipients(*s)
	}
	return cepu
}

// ClearEmailCcRecipients clears the value of the "email_cc_recipients" field.
func (cepu *CustomerEmailProfileUpdate) ClearEmailCcRecipients() *CustomerEmailProfileUpdate {
	cepu.mutation.ClearEmailCcRecipients()
	return cepu
}

// SetAttachmentName sets the "attachment_name" field.
func (cepu *CustomerEmailProfileUpdate) SetAttachmentName(s string) *CustomerEmailProfileUpdate {
	cepu.mutation.SetAttachmentName(s)
	return cepu
}

// SetNillableAttachmentName sets the "attachment_name" field if the given value is not nil.
func (cepu *CustomerEmailProfileUpdate) SetNillableAttachmentName(s *string) *CustomerEmailProfileUpdate {
	if s != nil {
		cepu.SetAttachmentName(*s)
	}
	return cepu
}

// ClearAttachmentName clears the value of the "attachment_name" field.
func (cepu *CustomerEmailProfileUpdate) ClearAttachmentName() *CustomerEmailProfileUpdate {
	cepu.mutation.ClearAttachmentName()
	return cepu
}

// SetEmailFormat sets the "email_format" field.
func (cepu *CustomerEmailProfileUpdate) SetEmailFormat(cf customeremailprofile.EmailFormat) *CustomerEmailProfileUpdate {
	cepu.mutation.SetEmailFormat(cf)
	return cepu
}

// SetNillableEmailFormat sets the "email_format" field if the given value is not nil.
func (cepu *CustomerEmailProfileUpdate) SetNillableEmailFormat(cf *customeremailprofile.EmailFormat) *CustomerEmailProfileUpdate {
	if cf != nil {
		cepu.SetEmailFormat(*cf)
	}
	return cepu
}

// SetOrganization sets the "organization" edge to the Organization entity.
func (cepu *CustomerEmailProfileUpdate) SetOrganization(o *Organization) *CustomerEmailProfileUpdate {
	return cepu.SetOrganizationID(o.ID)
}

// SetEmailProfile sets the "email_profile" edge to the EmailProfile entity.
func (cepu *CustomerEmailProfileUpdate) SetEmailProfile(e *EmailProfile) *CustomerEmailProfileUpdate {
	return cepu.SetEmailProfileID(e.ID)
}

// Mutation returns the CustomerEmailProfileMutation object of the builder.
func (cepu *CustomerEmailProfileUpdate) Mutation() *CustomerEmailProfileMutation {
	return cepu.mutation
}

// ClearOrganization clears the "organization" edge to the Organization entity.
func (cepu *CustomerEmailProfileUpdate) ClearOrganization() *CustomerEmailProfileUpdate {
	cepu.mutation.ClearOrganization()
	return cepu
}

// ClearEmailProfile clears the "email_profile" edge to the EmailProfile entity.
func (cepu *CustomerEmailProfileUpdate) ClearEmailProfile() *CustomerEmailProfileUpdate {
	cepu.mutation.ClearEmailProfile()
	return cepu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (cepu *CustomerEmailProfileUpdate) Save(ctx context.Context) (int, error) {
	cepu.defaults()
	return withHooks(ctx, cepu.sqlSave, cepu.mutation, cepu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (cepu *CustomerEmailProfileUpdate) SaveX(ctx context.Context) int {
	affected, err := cepu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (cepu *CustomerEmailProfileUpdate) Exec(ctx context.Context) error {
	_, err := cepu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (cepu *CustomerEmailProfileUpdate) ExecX(ctx context.Context) {
	if err := cepu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (cepu *CustomerEmailProfileUpdate) defaults() {
	if _, ok := cepu.mutation.UpdatedAt(); !ok {
		v := customeremailprofile.UpdateDefaultUpdatedAt()
		cepu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (cepu *CustomerEmailProfileUpdate) check() error {
	if v, ok := cepu.mutation.Subject(); ok {
		if err := customeremailprofile.SubjectValidator(v); err != nil {
			return &ValidationError{Name: "subject", err: fmt.Errorf(`ent: validator failed for field "CustomerEmailProfile.subject": %w`, err)}
		}
	}
	if v, ok := cepu.mutation.EmailRecipients(); ok {
		if err := customeremailprofile.EmailRecipientsValidator(v); err != nil {
			return &ValidationError{Name: "email_recipients", err: fmt.Errorf(`ent: validator failed for field "CustomerEmailProfile.email_recipients": %w`, err)}
		}
	}
	if v, ok := cepu.mutation.EmailFormat(); ok {
		if err := customeremailprofile.EmailFormatValidator(v); err != nil {
			return &ValidationError{Name: "email_format", err: fmt.Errorf(`ent: validator failed for field "CustomerEmailProfile.email_format": %w`, err)}
		}
	}
	if _, ok := cepu.mutation.BusinessUnitID(); cepu.mutation.BusinessUnitCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "CustomerEmailProfile.business_unit"`)
	}
	if _, ok := cepu.mutation.OrganizationID(); cepu.mutation.OrganizationCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "CustomerEmailProfile.organization"`)
	}
	if _, ok := cepu.mutation.CustomerID(); cepu.mutation.CustomerCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "CustomerEmailProfile.customer"`)
	}
	return nil
}

// Modify adds a statement modifier for attaching custom logic to the UPDATE statement.
func (cepu *CustomerEmailProfileUpdate) Modify(modifiers ...func(u *sql.UpdateBuilder)) *CustomerEmailProfileUpdate {
	cepu.modifiers = append(cepu.modifiers, modifiers...)
	return cepu
}

func (cepu *CustomerEmailProfileUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := cepu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(customeremailprofile.Table, customeremailprofile.Columns, sqlgraph.NewFieldSpec(customeremailprofile.FieldID, field.TypeUUID))
	if ps := cepu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := cepu.mutation.UpdatedAt(); ok {
		_spec.SetField(customeremailprofile.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := cepu.mutation.Version(); ok {
		_spec.SetField(customeremailprofile.FieldVersion, field.TypeInt, value)
	}
	if value, ok := cepu.mutation.AddedVersion(); ok {
		_spec.AddField(customeremailprofile.FieldVersion, field.TypeInt, value)
	}
	if value, ok := cepu.mutation.Subject(); ok {
		_spec.SetField(customeremailprofile.FieldSubject, field.TypeString, value)
	}
	if cepu.mutation.SubjectCleared() {
		_spec.ClearField(customeremailprofile.FieldSubject, field.TypeString)
	}
	if value, ok := cepu.mutation.EmailRecipients(); ok {
		_spec.SetField(customeremailprofile.FieldEmailRecipients, field.TypeString, value)
	}
	if value, ok := cepu.mutation.EmailCcRecipients(); ok {
		_spec.SetField(customeremailprofile.FieldEmailCcRecipients, field.TypeString, value)
	}
	if cepu.mutation.EmailCcRecipientsCleared() {
		_spec.ClearField(customeremailprofile.FieldEmailCcRecipients, field.TypeString)
	}
	if value, ok := cepu.mutation.AttachmentName(); ok {
		_spec.SetField(customeremailprofile.FieldAttachmentName, field.TypeString, value)
	}
	if cepu.mutation.AttachmentNameCleared() {
		_spec.ClearField(customeremailprofile.FieldAttachmentName, field.TypeString)
	}
	if value, ok := cepu.mutation.EmailFormat(); ok {
		_spec.SetField(customeremailprofile.FieldEmailFormat, field.TypeEnum, value)
	}
	if cepu.mutation.OrganizationCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   customeremailprofile.OrganizationTable,
			Columns: []string{customeremailprofile.OrganizationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(organization.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := cepu.mutation.OrganizationIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   customeremailprofile.OrganizationTable,
			Columns: []string{customeremailprofile.OrganizationColumn},
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
	if cepu.mutation.EmailProfileCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   customeremailprofile.EmailProfileTable,
			Columns: []string{customeremailprofile.EmailProfileColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(emailprofile.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := cepu.mutation.EmailProfileIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   customeremailprofile.EmailProfileTable,
			Columns: []string{customeremailprofile.EmailProfileColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(emailprofile.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_spec.AddModifiers(cepu.modifiers...)
	if n, err = sqlgraph.UpdateNodes(ctx, cepu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{customeremailprofile.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	cepu.mutation.done = true
	return n, nil
}

// CustomerEmailProfileUpdateOne is the builder for updating a single CustomerEmailProfile entity.
type CustomerEmailProfileUpdateOne struct {
	config
	fields    []string
	hooks     []Hook
	mutation  *CustomerEmailProfileMutation
	modifiers []func(*sql.UpdateBuilder)
}

// SetOrganizationID sets the "organization_id" field.
func (cepuo *CustomerEmailProfileUpdateOne) SetOrganizationID(u uuid.UUID) *CustomerEmailProfileUpdateOne {
	cepuo.mutation.SetOrganizationID(u)
	return cepuo
}

// SetNillableOrganizationID sets the "organization_id" field if the given value is not nil.
func (cepuo *CustomerEmailProfileUpdateOne) SetNillableOrganizationID(u *uuid.UUID) *CustomerEmailProfileUpdateOne {
	if u != nil {
		cepuo.SetOrganizationID(*u)
	}
	return cepuo
}

// SetUpdatedAt sets the "updated_at" field.
func (cepuo *CustomerEmailProfileUpdateOne) SetUpdatedAt(t time.Time) *CustomerEmailProfileUpdateOne {
	cepuo.mutation.SetUpdatedAt(t)
	return cepuo
}

// SetVersion sets the "version" field.
func (cepuo *CustomerEmailProfileUpdateOne) SetVersion(i int) *CustomerEmailProfileUpdateOne {
	cepuo.mutation.ResetVersion()
	cepuo.mutation.SetVersion(i)
	return cepuo
}

// SetNillableVersion sets the "version" field if the given value is not nil.
func (cepuo *CustomerEmailProfileUpdateOne) SetNillableVersion(i *int) *CustomerEmailProfileUpdateOne {
	if i != nil {
		cepuo.SetVersion(*i)
	}
	return cepuo
}

// AddVersion adds i to the "version" field.
func (cepuo *CustomerEmailProfileUpdateOne) AddVersion(i int) *CustomerEmailProfileUpdateOne {
	cepuo.mutation.AddVersion(i)
	return cepuo
}

// SetSubject sets the "subject" field.
func (cepuo *CustomerEmailProfileUpdateOne) SetSubject(s string) *CustomerEmailProfileUpdateOne {
	cepuo.mutation.SetSubject(s)
	return cepuo
}

// SetNillableSubject sets the "subject" field if the given value is not nil.
func (cepuo *CustomerEmailProfileUpdateOne) SetNillableSubject(s *string) *CustomerEmailProfileUpdateOne {
	if s != nil {
		cepuo.SetSubject(*s)
	}
	return cepuo
}

// ClearSubject clears the value of the "subject" field.
func (cepuo *CustomerEmailProfileUpdateOne) ClearSubject() *CustomerEmailProfileUpdateOne {
	cepuo.mutation.ClearSubject()
	return cepuo
}

// SetEmailProfileID sets the "email_profile_id" field.
func (cepuo *CustomerEmailProfileUpdateOne) SetEmailProfileID(u uuid.UUID) *CustomerEmailProfileUpdateOne {
	cepuo.mutation.SetEmailProfileID(u)
	return cepuo
}

// SetNillableEmailProfileID sets the "email_profile_id" field if the given value is not nil.
func (cepuo *CustomerEmailProfileUpdateOne) SetNillableEmailProfileID(u *uuid.UUID) *CustomerEmailProfileUpdateOne {
	if u != nil {
		cepuo.SetEmailProfileID(*u)
	}
	return cepuo
}

// ClearEmailProfileID clears the value of the "email_profile_id" field.
func (cepuo *CustomerEmailProfileUpdateOne) ClearEmailProfileID() *CustomerEmailProfileUpdateOne {
	cepuo.mutation.ClearEmailProfileID()
	return cepuo
}

// SetEmailRecipients sets the "email_recipients" field.
func (cepuo *CustomerEmailProfileUpdateOne) SetEmailRecipients(s string) *CustomerEmailProfileUpdateOne {
	cepuo.mutation.SetEmailRecipients(s)
	return cepuo
}

// SetNillableEmailRecipients sets the "email_recipients" field if the given value is not nil.
func (cepuo *CustomerEmailProfileUpdateOne) SetNillableEmailRecipients(s *string) *CustomerEmailProfileUpdateOne {
	if s != nil {
		cepuo.SetEmailRecipients(*s)
	}
	return cepuo
}

// SetEmailCcRecipients sets the "email_cc_recipients" field.
func (cepuo *CustomerEmailProfileUpdateOne) SetEmailCcRecipients(s string) *CustomerEmailProfileUpdateOne {
	cepuo.mutation.SetEmailCcRecipients(s)
	return cepuo
}

// SetNillableEmailCcRecipients sets the "email_cc_recipients" field if the given value is not nil.
func (cepuo *CustomerEmailProfileUpdateOne) SetNillableEmailCcRecipients(s *string) *CustomerEmailProfileUpdateOne {
	if s != nil {
		cepuo.SetEmailCcRecipients(*s)
	}
	return cepuo
}

// ClearEmailCcRecipients clears the value of the "email_cc_recipients" field.
func (cepuo *CustomerEmailProfileUpdateOne) ClearEmailCcRecipients() *CustomerEmailProfileUpdateOne {
	cepuo.mutation.ClearEmailCcRecipients()
	return cepuo
}

// SetAttachmentName sets the "attachment_name" field.
func (cepuo *CustomerEmailProfileUpdateOne) SetAttachmentName(s string) *CustomerEmailProfileUpdateOne {
	cepuo.mutation.SetAttachmentName(s)
	return cepuo
}

// SetNillableAttachmentName sets the "attachment_name" field if the given value is not nil.
func (cepuo *CustomerEmailProfileUpdateOne) SetNillableAttachmentName(s *string) *CustomerEmailProfileUpdateOne {
	if s != nil {
		cepuo.SetAttachmentName(*s)
	}
	return cepuo
}

// ClearAttachmentName clears the value of the "attachment_name" field.
func (cepuo *CustomerEmailProfileUpdateOne) ClearAttachmentName() *CustomerEmailProfileUpdateOne {
	cepuo.mutation.ClearAttachmentName()
	return cepuo
}

// SetEmailFormat sets the "email_format" field.
func (cepuo *CustomerEmailProfileUpdateOne) SetEmailFormat(cf customeremailprofile.EmailFormat) *CustomerEmailProfileUpdateOne {
	cepuo.mutation.SetEmailFormat(cf)
	return cepuo
}

// SetNillableEmailFormat sets the "email_format" field if the given value is not nil.
func (cepuo *CustomerEmailProfileUpdateOne) SetNillableEmailFormat(cf *customeremailprofile.EmailFormat) *CustomerEmailProfileUpdateOne {
	if cf != nil {
		cepuo.SetEmailFormat(*cf)
	}
	return cepuo
}

// SetOrganization sets the "organization" edge to the Organization entity.
func (cepuo *CustomerEmailProfileUpdateOne) SetOrganization(o *Organization) *CustomerEmailProfileUpdateOne {
	return cepuo.SetOrganizationID(o.ID)
}

// SetEmailProfile sets the "email_profile" edge to the EmailProfile entity.
func (cepuo *CustomerEmailProfileUpdateOne) SetEmailProfile(e *EmailProfile) *CustomerEmailProfileUpdateOne {
	return cepuo.SetEmailProfileID(e.ID)
}

// Mutation returns the CustomerEmailProfileMutation object of the builder.
func (cepuo *CustomerEmailProfileUpdateOne) Mutation() *CustomerEmailProfileMutation {
	return cepuo.mutation
}

// ClearOrganization clears the "organization" edge to the Organization entity.
func (cepuo *CustomerEmailProfileUpdateOne) ClearOrganization() *CustomerEmailProfileUpdateOne {
	cepuo.mutation.ClearOrganization()
	return cepuo
}

// ClearEmailProfile clears the "email_profile" edge to the EmailProfile entity.
func (cepuo *CustomerEmailProfileUpdateOne) ClearEmailProfile() *CustomerEmailProfileUpdateOne {
	cepuo.mutation.ClearEmailProfile()
	return cepuo
}

// Where appends a list predicates to the CustomerEmailProfileUpdate builder.
func (cepuo *CustomerEmailProfileUpdateOne) Where(ps ...predicate.CustomerEmailProfile) *CustomerEmailProfileUpdateOne {
	cepuo.mutation.Where(ps...)
	return cepuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (cepuo *CustomerEmailProfileUpdateOne) Select(field string, fields ...string) *CustomerEmailProfileUpdateOne {
	cepuo.fields = append([]string{field}, fields...)
	return cepuo
}

// Save executes the query and returns the updated CustomerEmailProfile entity.
func (cepuo *CustomerEmailProfileUpdateOne) Save(ctx context.Context) (*CustomerEmailProfile, error) {
	cepuo.defaults()
	return withHooks(ctx, cepuo.sqlSave, cepuo.mutation, cepuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (cepuo *CustomerEmailProfileUpdateOne) SaveX(ctx context.Context) *CustomerEmailProfile {
	node, err := cepuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (cepuo *CustomerEmailProfileUpdateOne) Exec(ctx context.Context) error {
	_, err := cepuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (cepuo *CustomerEmailProfileUpdateOne) ExecX(ctx context.Context) {
	if err := cepuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (cepuo *CustomerEmailProfileUpdateOne) defaults() {
	if _, ok := cepuo.mutation.UpdatedAt(); !ok {
		v := customeremailprofile.UpdateDefaultUpdatedAt()
		cepuo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (cepuo *CustomerEmailProfileUpdateOne) check() error {
	if v, ok := cepuo.mutation.Subject(); ok {
		if err := customeremailprofile.SubjectValidator(v); err != nil {
			return &ValidationError{Name: "subject", err: fmt.Errorf(`ent: validator failed for field "CustomerEmailProfile.subject": %w`, err)}
		}
	}
	if v, ok := cepuo.mutation.EmailRecipients(); ok {
		if err := customeremailprofile.EmailRecipientsValidator(v); err != nil {
			return &ValidationError{Name: "email_recipients", err: fmt.Errorf(`ent: validator failed for field "CustomerEmailProfile.email_recipients": %w`, err)}
		}
	}
	if v, ok := cepuo.mutation.EmailFormat(); ok {
		if err := customeremailprofile.EmailFormatValidator(v); err != nil {
			return &ValidationError{Name: "email_format", err: fmt.Errorf(`ent: validator failed for field "CustomerEmailProfile.email_format": %w`, err)}
		}
	}
	if _, ok := cepuo.mutation.BusinessUnitID(); cepuo.mutation.BusinessUnitCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "CustomerEmailProfile.business_unit"`)
	}
	if _, ok := cepuo.mutation.OrganizationID(); cepuo.mutation.OrganizationCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "CustomerEmailProfile.organization"`)
	}
	if _, ok := cepuo.mutation.CustomerID(); cepuo.mutation.CustomerCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "CustomerEmailProfile.customer"`)
	}
	return nil
}

// Modify adds a statement modifier for attaching custom logic to the UPDATE statement.
func (cepuo *CustomerEmailProfileUpdateOne) Modify(modifiers ...func(u *sql.UpdateBuilder)) *CustomerEmailProfileUpdateOne {
	cepuo.modifiers = append(cepuo.modifiers, modifiers...)
	return cepuo
}

func (cepuo *CustomerEmailProfileUpdateOne) sqlSave(ctx context.Context) (_node *CustomerEmailProfile, err error) {
	if err := cepuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(customeremailprofile.Table, customeremailprofile.Columns, sqlgraph.NewFieldSpec(customeremailprofile.FieldID, field.TypeUUID))
	id, ok := cepuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "CustomerEmailProfile.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := cepuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, customeremailprofile.FieldID)
		for _, f := range fields {
			if !customeremailprofile.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != customeremailprofile.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := cepuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := cepuo.mutation.UpdatedAt(); ok {
		_spec.SetField(customeremailprofile.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := cepuo.mutation.Version(); ok {
		_spec.SetField(customeremailprofile.FieldVersion, field.TypeInt, value)
	}
	if value, ok := cepuo.mutation.AddedVersion(); ok {
		_spec.AddField(customeremailprofile.FieldVersion, field.TypeInt, value)
	}
	if value, ok := cepuo.mutation.Subject(); ok {
		_spec.SetField(customeremailprofile.FieldSubject, field.TypeString, value)
	}
	if cepuo.mutation.SubjectCleared() {
		_spec.ClearField(customeremailprofile.FieldSubject, field.TypeString)
	}
	if value, ok := cepuo.mutation.EmailRecipients(); ok {
		_spec.SetField(customeremailprofile.FieldEmailRecipients, field.TypeString, value)
	}
	if value, ok := cepuo.mutation.EmailCcRecipients(); ok {
		_spec.SetField(customeremailprofile.FieldEmailCcRecipients, field.TypeString, value)
	}
	if cepuo.mutation.EmailCcRecipientsCleared() {
		_spec.ClearField(customeremailprofile.FieldEmailCcRecipients, field.TypeString)
	}
	if value, ok := cepuo.mutation.AttachmentName(); ok {
		_spec.SetField(customeremailprofile.FieldAttachmentName, field.TypeString, value)
	}
	if cepuo.mutation.AttachmentNameCleared() {
		_spec.ClearField(customeremailprofile.FieldAttachmentName, field.TypeString)
	}
	if value, ok := cepuo.mutation.EmailFormat(); ok {
		_spec.SetField(customeremailprofile.FieldEmailFormat, field.TypeEnum, value)
	}
	if cepuo.mutation.OrganizationCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   customeremailprofile.OrganizationTable,
			Columns: []string{customeremailprofile.OrganizationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(organization.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := cepuo.mutation.OrganizationIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   customeremailprofile.OrganizationTable,
			Columns: []string{customeremailprofile.OrganizationColumn},
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
	if cepuo.mutation.EmailProfileCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   customeremailprofile.EmailProfileTable,
			Columns: []string{customeremailprofile.EmailProfileColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(emailprofile.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := cepuo.mutation.EmailProfileIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   customeremailprofile.EmailProfileTable,
			Columns: []string{customeremailprofile.EmailProfileColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(emailprofile.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_spec.AddModifiers(cepuo.modifiers...)
	_node = &CustomerEmailProfile{config: cepuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, cepuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{customeremailprofile.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	cepuo.mutation.done = true
	return _node, nil
}
