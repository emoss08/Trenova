// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/emoss08/trenova/internal/ent/customercontact"
	"github.com/emoss08/trenova/internal/ent/predicate"
)

// CustomerContactDelete is the builder for deleting a CustomerContact entity.
type CustomerContactDelete struct {
	config
	hooks    []Hook
	mutation *CustomerContactMutation
}

// Where appends a list predicates to the CustomerContactDelete builder.
func (ccd *CustomerContactDelete) Where(ps ...predicate.CustomerContact) *CustomerContactDelete {
	ccd.mutation.Where(ps...)
	return ccd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (ccd *CustomerContactDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, ccd.sqlExec, ccd.mutation, ccd.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (ccd *CustomerContactDelete) ExecX(ctx context.Context) int {
	n, err := ccd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (ccd *CustomerContactDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(customercontact.Table, sqlgraph.NewFieldSpec(customercontact.FieldID, field.TypeUUID))
	if ps := ccd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, ccd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	ccd.mutation.done = true
	return affected, err
}

// CustomerContactDeleteOne is the builder for deleting a single CustomerContact entity.
type CustomerContactDeleteOne struct {
	ccd *CustomerContactDelete
}

// Where appends a list predicates to the CustomerContactDelete builder.
func (ccdo *CustomerContactDeleteOne) Where(ps ...predicate.CustomerContact) *CustomerContactDeleteOne {
	ccdo.ccd.mutation.Where(ps...)
	return ccdo
}

// Exec executes the deletion query.
func (ccdo *CustomerContactDeleteOne) Exec(ctx context.Context) error {
	n, err := ccdo.ccd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{customercontact.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (ccdo *CustomerContactDeleteOne) ExecX(ctx context.Context) {
	if err := ccdo.Exec(ctx); err != nil {
		panic(err)
	}
}