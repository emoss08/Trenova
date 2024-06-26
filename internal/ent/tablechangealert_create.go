// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/tablechangealert"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// TableChangeAlertCreate is the builder for creating a TableChangeAlert entity.
type TableChangeAlertCreate struct {
	config
	mutation *TableChangeAlertMutation
	hooks    []Hook
}

// SetBusinessUnitID sets the "business_unit_id" field.
func (tcac *TableChangeAlertCreate) SetBusinessUnitID(u uuid.UUID) *TableChangeAlertCreate {
	tcac.mutation.SetBusinessUnitID(u)
	return tcac
}

// SetOrganizationID sets the "organization_id" field.
func (tcac *TableChangeAlertCreate) SetOrganizationID(u uuid.UUID) *TableChangeAlertCreate {
	tcac.mutation.SetOrganizationID(u)
	return tcac
}

// SetCreatedAt sets the "created_at" field.
func (tcac *TableChangeAlertCreate) SetCreatedAt(t time.Time) *TableChangeAlertCreate {
	tcac.mutation.SetCreatedAt(t)
	return tcac
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (tcac *TableChangeAlertCreate) SetNillableCreatedAt(t *time.Time) *TableChangeAlertCreate {
	if t != nil {
		tcac.SetCreatedAt(*t)
	}
	return tcac
}

// SetUpdatedAt sets the "updated_at" field.
func (tcac *TableChangeAlertCreate) SetUpdatedAt(t time.Time) *TableChangeAlertCreate {
	tcac.mutation.SetUpdatedAt(t)
	return tcac
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (tcac *TableChangeAlertCreate) SetNillableUpdatedAt(t *time.Time) *TableChangeAlertCreate {
	if t != nil {
		tcac.SetUpdatedAt(*t)
	}
	return tcac
}

// SetVersion sets the "version" field.
func (tcac *TableChangeAlertCreate) SetVersion(i int) *TableChangeAlertCreate {
	tcac.mutation.SetVersion(i)
	return tcac
}

// SetNillableVersion sets the "version" field if the given value is not nil.
func (tcac *TableChangeAlertCreate) SetNillableVersion(i *int) *TableChangeAlertCreate {
	if i != nil {
		tcac.SetVersion(*i)
	}
	return tcac
}

// SetStatus sets the "status" field.
func (tcac *TableChangeAlertCreate) SetStatus(t tablechangealert.Status) *TableChangeAlertCreate {
	tcac.mutation.SetStatus(t)
	return tcac
}

// SetNillableStatus sets the "status" field if the given value is not nil.
func (tcac *TableChangeAlertCreate) SetNillableStatus(t *tablechangealert.Status) *TableChangeAlertCreate {
	if t != nil {
		tcac.SetStatus(*t)
	}
	return tcac
}

// SetName sets the "name" field.
func (tcac *TableChangeAlertCreate) SetName(s string) *TableChangeAlertCreate {
	tcac.mutation.SetName(s)
	return tcac
}

// SetDatabaseAction sets the "database_action" field.
func (tcac *TableChangeAlertCreate) SetDatabaseAction(ta tablechangealert.DatabaseAction) *TableChangeAlertCreate {
	tcac.mutation.SetDatabaseAction(ta)
	return tcac
}

// SetTopicName sets the "topic_name" field.
func (tcac *TableChangeAlertCreate) SetTopicName(s string) *TableChangeAlertCreate {
	tcac.mutation.SetTopicName(s)
	return tcac
}

// SetNillableTopicName sets the "topic_name" field if the given value is not nil.
func (tcac *TableChangeAlertCreate) SetNillableTopicName(s *string) *TableChangeAlertCreate {
	if s != nil {
		tcac.SetTopicName(*s)
	}
	return tcac
}

// SetDescription sets the "description" field.
func (tcac *TableChangeAlertCreate) SetDescription(s string) *TableChangeAlertCreate {
	tcac.mutation.SetDescription(s)
	return tcac
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (tcac *TableChangeAlertCreate) SetNillableDescription(s *string) *TableChangeAlertCreate {
	if s != nil {
		tcac.SetDescription(*s)
	}
	return tcac
}

// SetCustomSubject sets the "custom_subject" field.
func (tcac *TableChangeAlertCreate) SetCustomSubject(s string) *TableChangeAlertCreate {
	tcac.mutation.SetCustomSubject(s)
	return tcac
}

// SetNillableCustomSubject sets the "custom_subject" field if the given value is not nil.
func (tcac *TableChangeAlertCreate) SetNillableCustomSubject(s *string) *TableChangeAlertCreate {
	if s != nil {
		tcac.SetCustomSubject(*s)
	}
	return tcac
}

// SetDeliveryMethod sets the "delivery_method" field.
func (tcac *TableChangeAlertCreate) SetDeliveryMethod(tm tablechangealert.DeliveryMethod) *TableChangeAlertCreate {
	tcac.mutation.SetDeliveryMethod(tm)
	return tcac
}

// SetNillableDeliveryMethod sets the "delivery_method" field if the given value is not nil.
func (tcac *TableChangeAlertCreate) SetNillableDeliveryMethod(tm *tablechangealert.DeliveryMethod) *TableChangeAlertCreate {
	if tm != nil {
		tcac.SetDeliveryMethod(*tm)
	}
	return tcac
}

// SetEmailRecipients sets the "email_recipients" field.
func (tcac *TableChangeAlertCreate) SetEmailRecipients(s string) *TableChangeAlertCreate {
	tcac.mutation.SetEmailRecipients(s)
	return tcac
}

// SetNillableEmailRecipients sets the "email_recipients" field if the given value is not nil.
func (tcac *TableChangeAlertCreate) SetNillableEmailRecipients(s *string) *TableChangeAlertCreate {
	if s != nil {
		tcac.SetEmailRecipients(*s)
	}
	return tcac
}

// SetEffectiveDate sets the "effective_date" field.
func (tcac *TableChangeAlertCreate) SetEffectiveDate(pg *pgtype.Date) *TableChangeAlertCreate {
	tcac.mutation.SetEffectiveDate(pg)
	return tcac
}

// SetExpirationDate sets the "expiration_date" field.
func (tcac *TableChangeAlertCreate) SetExpirationDate(pg *pgtype.Date) *TableChangeAlertCreate {
	tcac.mutation.SetExpirationDate(pg)
	return tcac
}

// SetConditionalLogic sets the "conditional_logic" field.
func (tcac *TableChangeAlertCreate) SetConditionalLogic(m map[string]interface{}) *TableChangeAlertCreate {
	tcac.mutation.SetConditionalLogic(m)
	return tcac
}

// SetID sets the "id" field.
func (tcac *TableChangeAlertCreate) SetID(u uuid.UUID) *TableChangeAlertCreate {
	tcac.mutation.SetID(u)
	return tcac
}

// SetNillableID sets the "id" field if the given value is not nil.
func (tcac *TableChangeAlertCreate) SetNillableID(u *uuid.UUID) *TableChangeAlertCreate {
	if u != nil {
		tcac.SetID(*u)
	}
	return tcac
}

// SetBusinessUnit sets the "business_unit" edge to the BusinessUnit entity.
func (tcac *TableChangeAlertCreate) SetBusinessUnit(b *BusinessUnit) *TableChangeAlertCreate {
	return tcac.SetBusinessUnitID(b.ID)
}

// SetOrganization sets the "organization" edge to the Organization entity.
func (tcac *TableChangeAlertCreate) SetOrganization(o *Organization) *TableChangeAlertCreate {
	return tcac.SetOrganizationID(o.ID)
}

// Mutation returns the TableChangeAlertMutation object of the builder.
func (tcac *TableChangeAlertCreate) Mutation() *TableChangeAlertMutation {
	return tcac.mutation
}

// Save creates the TableChangeAlert in the database.
func (tcac *TableChangeAlertCreate) Save(ctx context.Context) (*TableChangeAlert, error) {
	if err := tcac.defaults(); err != nil {
		return nil, err
	}
	return withHooks(ctx, tcac.sqlSave, tcac.mutation, tcac.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (tcac *TableChangeAlertCreate) SaveX(ctx context.Context) *TableChangeAlert {
	v, err := tcac.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (tcac *TableChangeAlertCreate) Exec(ctx context.Context) error {
	_, err := tcac.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (tcac *TableChangeAlertCreate) ExecX(ctx context.Context) {
	if err := tcac.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (tcac *TableChangeAlertCreate) defaults() error {
	if _, ok := tcac.mutation.CreatedAt(); !ok {
		if tablechangealert.DefaultCreatedAt == nil {
			return fmt.Errorf("ent: uninitialized tablechangealert.DefaultCreatedAt (forgotten import ent/runtime?)")
		}
		v := tablechangealert.DefaultCreatedAt()
		tcac.mutation.SetCreatedAt(v)
	}
	if _, ok := tcac.mutation.UpdatedAt(); !ok {
		if tablechangealert.DefaultUpdatedAt == nil {
			return fmt.Errorf("ent: uninitialized tablechangealert.DefaultUpdatedAt (forgotten import ent/runtime?)")
		}
		v := tablechangealert.DefaultUpdatedAt()
		tcac.mutation.SetUpdatedAt(v)
	}
	if _, ok := tcac.mutation.Version(); !ok {
		v := tablechangealert.DefaultVersion
		tcac.mutation.SetVersion(v)
	}
	if _, ok := tcac.mutation.Status(); !ok {
		v := tablechangealert.DefaultStatus
		tcac.mutation.SetStatus(v)
	}
	if _, ok := tcac.mutation.DeliveryMethod(); !ok {
		v := tablechangealert.DefaultDeliveryMethod
		tcac.mutation.SetDeliveryMethod(v)
	}
	if _, ok := tcac.mutation.ID(); !ok {
		if tablechangealert.DefaultID == nil {
			return fmt.Errorf("ent: uninitialized tablechangealert.DefaultID (forgotten import ent/runtime?)")
		}
		v := tablechangealert.DefaultID()
		tcac.mutation.SetID(v)
	}
	return nil
}

// check runs all checks and user-defined validators on the builder.
func (tcac *TableChangeAlertCreate) check() error {
	if _, ok := tcac.mutation.BusinessUnitID(); !ok {
		return &ValidationError{Name: "business_unit_id", err: errors.New(`ent: missing required field "TableChangeAlert.business_unit_id"`)}
	}
	if _, ok := tcac.mutation.OrganizationID(); !ok {
		return &ValidationError{Name: "organization_id", err: errors.New(`ent: missing required field "TableChangeAlert.organization_id"`)}
	}
	if _, ok := tcac.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`ent: missing required field "TableChangeAlert.created_at"`)}
	}
	if _, ok := tcac.mutation.UpdatedAt(); !ok {
		return &ValidationError{Name: "updated_at", err: errors.New(`ent: missing required field "TableChangeAlert.updated_at"`)}
	}
	if _, ok := tcac.mutation.Version(); !ok {
		return &ValidationError{Name: "version", err: errors.New(`ent: missing required field "TableChangeAlert.version"`)}
	}
	if _, ok := tcac.mutation.Status(); !ok {
		return &ValidationError{Name: "status", err: errors.New(`ent: missing required field "TableChangeAlert.status"`)}
	}
	if v, ok := tcac.mutation.Status(); ok {
		if err := tablechangealert.StatusValidator(v); err != nil {
			return &ValidationError{Name: "status", err: fmt.Errorf(`ent: validator failed for field "TableChangeAlert.status": %w`, err)}
		}
	}
	if _, ok := tcac.mutation.Name(); !ok {
		return &ValidationError{Name: "name", err: errors.New(`ent: missing required field "TableChangeAlert.name"`)}
	}
	if v, ok := tcac.mutation.Name(); ok {
		if err := tablechangealert.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "TableChangeAlert.name": %w`, err)}
		}
	}
	if _, ok := tcac.mutation.DatabaseAction(); !ok {
		return &ValidationError{Name: "database_action", err: errors.New(`ent: missing required field "TableChangeAlert.database_action"`)}
	}
	if v, ok := tcac.mutation.DatabaseAction(); ok {
		if err := tablechangealert.DatabaseActionValidator(v); err != nil {
			return &ValidationError{Name: "database_action", err: fmt.Errorf(`ent: validator failed for field "TableChangeAlert.database_action": %w`, err)}
		}
	}
	if _, ok := tcac.mutation.DeliveryMethod(); !ok {
		return &ValidationError{Name: "delivery_method", err: errors.New(`ent: missing required field "TableChangeAlert.delivery_method"`)}
	}
	if v, ok := tcac.mutation.DeliveryMethod(); ok {
		if err := tablechangealert.DeliveryMethodValidator(v); err != nil {
			return &ValidationError{Name: "delivery_method", err: fmt.Errorf(`ent: validator failed for field "TableChangeAlert.delivery_method": %w`, err)}
		}
	}
	if _, ok := tcac.mutation.BusinessUnitID(); !ok {
		return &ValidationError{Name: "business_unit", err: errors.New(`ent: missing required edge "TableChangeAlert.business_unit"`)}
	}
	if _, ok := tcac.mutation.OrganizationID(); !ok {
		return &ValidationError{Name: "organization", err: errors.New(`ent: missing required edge "TableChangeAlert.organization"`)}
	}
	return nil
}

func (tcac *TableChangeAlertCreate) sqlSave(ctx context.Context) (*TableChangeAlert, error) {
	if err := tcac.check(); err != nil {
		return nil, err
	}
	_node, _spec := tcac.createSpec()
	if err := sqlgraph.CreateNode(ctx, tcac.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(*uuid.UUID); ok {
			_node.ID = *id
		} else if err := _node.ID.Scan(_spec.ID.Value); err != nil {
			return nil, err
		}
	}
	tcac.mutation.id = &_node.ID
	tcac.mutation.done = true
	return _node, nil
}

func (tcac *TableChangeAlertCreate) createSpec() (*TableChangeAlert, *sqlgraph.CreateSpec) {
	var (
		_node = &TableChangeAlert{config: tcac.config}
		_spec = sqlgraph.NewCreateSpec(tablechangealert.Table, sqlgraph.NewFieldSpec(tablechangealert.FieldID, field.TypeUUID))
	)
	if id, ok := tcac.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := tcac.mutation.CreatedAt(); ok {
		_spec.SetField(tablechangealert.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := tcac.mutation.UpdatedAt(); ok {
		_spec.SetField(tablechangealert.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if value, ok := tcac.mutation.Version(); ok {
		_spec.SetField(tablechangealert.FieldVersion, field.TypeInt, value)
		_node.Version = value
	}
	if value, ok := tcac.mutation.Status(); ok {
		_spec.SetField(tablechangealert.FieldStatus, field.TypeEnum, value)
		_node.Status = value
	}
	if value, ok := tcac.mutation.Name(); ok {
		_spec.SetField(tablechangealert.FieldName, field.TypeString, value)
		_node.Name = value
	}
	if value, ok := tcac.mutation.DatabaseAction(); ok {
		_spec.SetField(tablechangealert.FieldDatabaseAction, field.TypeEnum, value)
		_node.DatabaseAction = value
	}
	if value, ok := tcac.mutation.TopicName(); ok {
		_spec.SetField(tablechangealert.FieldTopicName, field.TypeString, value)
		_node.TopicName = value
	}
	if value, ok := tcac.mutation.Description(); ok {
		_spec.SetField(tablechangealert.FieldDescription, field.TypeString, value)
		_node.Description = value
	}
	if value, ok := tcac.mutation.CustomSubject(); ok {
		_spec.SetField(tablechangealert.FieldCustomSubject, field.TypeString, value)
		_node.CustomSubject = value
	}
	if value, ok := tcac.mutation.DeliveryMethod(); ok {
		_spec.SetField(tablechangealert.FieldDeliveryMethod, field.TypeEnum, value)
		_node.DeliveryMethod = value
	}
	if value, ok := tcac.mutation.EmailRecipients(); ok {
		_spec.SetField(tablechangealert.FieldEmailRecipients, field.TypeString, value)
		_node.EmailRecipients = value
	}
	if value, ok := tcac.mutation.EffectiveDate(); ok {
		_spec.SetField(tablechangealert.FieldEffectiveDate, field.TypeOther, value)
		_node.EffectiveDate = value
	}
	if value, ok := tcac.mutation.ExpirationDate(); ok {
		_spec.SetField(tablechangealert.FieldExpirationDate, field.TypeOther, value)
		_node.ExpirationDate = value
	}
	if value, ok := tcac.mutation.ConditionalLogic(); ok {
		_spec.SetField(tablechangealert.FieldConditionalLogic, field.TypeJSON, value)
		_node.ConditionalLogic = value
	}
	if nodes := tcac.mutation.BusinessUnitIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   tablechangealert.BusinessUnitTable,
			Columns: []string{tablechangealert.BusinessUnitColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(businessunit.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.BusinessUnitID = nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := tcac.mutation.OrganizationIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   tablechangealert.OrganizationTable,
			Columns: []string{tablechangealert.OrganizationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(organization.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.OrganizationID = nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// TableChangeAlertCreateBulk is the builder for creating many TableChangeAlert entities in bulk.
type TableChangeAlertCreateBulk struct {
	config
	err      error
	builders []*TableChangeAlertCreate
}

// Save creates the TableChangeAlert entities in the database.
func (tcacb *TableChangeAlertCreateBulk) Save(ctx context.Context) ([]*TableChangeAlert, error) {
	if tcacb.err != nil {
		return nil, tcacb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(tcacb.builders))
	nodes := make([]*TableChangeAlert, len(tcacb.builders))
	mutators := make([]Mutator, len(tcacb.builders))
	for i := range tcacb.builders {
		func(i int, root context.Context) {
			builder := tcacb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*TableChangeAlertMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				var err error
				nodes[i], specs[i] = builder.createSpec()
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, tcacb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, tcacb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{msg: err.Error(), wrap: err}
						}
					}
				}
				if err != nil {
					return nil, err
				}
				mutation.id = &nodes[i].ID
				mutation.done = true
				return nodes[i], nil
			})
			for i := len(builder.hooks) - 1; i >= 0; i-- {
				mut = builder.hooks[i](mut)
			}
			mutators[i] = mut
		}(i, ctx)
	}
	if len(mutators) > 0 {
		if _, err := mutators[0].Mutate(ctx, tcacb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (tcacb *TableChangeAlertCreateBulk) SaveX(ctx context.Context) []*TableChangeAlert {
	v, err := tcacb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (tcacb *TableChangeAlertCreateBulk) Exec(ctx context.Context) error {
	_, err := tcacb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (tcacb *TableChangeAlertCreateBulk) ExecX(ctx context.Context) {
	if err := tcacb.Exec(ctx); err != nil {
		panic(err)
	}
}
