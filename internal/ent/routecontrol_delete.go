// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/emoss08/trenova/internal/ent/predicate"
	"github.com/emoss08/trenova/internal/ent/routecontrol"
)

// RouteControlDelete is the builder for deleting a RouteControl entity.
type RouteControlDelete struct {
	config
	hooks    []Hook
	mutation *RouteControlMutation
}

// Where appends a list predicates to the RouteControlDelete builder.
func (rcd *RouteControlDelete) Where(ps ...predicate.RouteControl) *RouteControlDelete {
	rcd.mutation.Where(ps...)
	return rcd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (rcd *RouteControlDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, rcd.sqlExec, rcd.mutation, rcd.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (rcd *RouteControlDelete) ExecX(ctx context.Context) int {
	n, err := rcd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (rcd *RouteControlDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(routecontrol.Table, sqlgraph.NewFieldSpec(routecontrol.FieldID, field.TypeUUID))
	if ps := rcd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, rcd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	rcd.mutation.done = true
	return affected, err
}

// RouteControlDeleteOne is the builder for deleting a single RouteControl entity.
type RouteControlDeleteOne struct {
	rcd *RouteControlDelete
}

// Where appends a list predicates to the RouteControlDelete builder.
func (rcdo *RouteControlDeleteOne) Where(ps ...predicate.RouteControl) *RouteControlDeleteOne {
	rcdo.rcd.mutation.Where(ps...)
	return rcdo
}

// Exec executes the deletion query.
func (rcdo *RouteControlDeleteOne) Exec(ctx context.Context) error {
	n, err := rcdo.rcd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{routecontrol.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (rcdo *RouteControlDeleteOne) ExecX(ctx context.Context) {
	if err := rcdo.Exec(ctx); err != nil {
		panic(err)
	}
}