// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/emoss08/trenova/internal/ent/customreport"
	"github.com/emoss08/trenova/internal/ent/predicate"
)

// CustomReportDelete is the builder for deleting a CustomReport entity.
type CustomReportDelete struct {
	config
	hooks    []Hook
	mutation *CustomReportMutation
}

// Where appends a list predicates to the CustomReportDelete builder.
func (crd *CustomReportDelete) Where(ps ...predicate.CustomReport) *CustomReportDelete {
	crd.mutation.Where(ps...)
	return crd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (crd *CustomReportDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, crd.sqlExec, crd.mutation, crd.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (crd *CustomReportDelete) ExecX(ctx context.Context) int {
	n, err := crd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (crd *CustomReportDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(customreport.Table, sqlgraph.NewFieldSpec(customreport.FieldID, field.TypeUUID))
	if ps := crd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, crd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	crd.mutation.done = true
	return affected, err
}

// CustomReportDeleteOne is the builder for deleting a single CustomReport entity.
type CustomReportDeleteOne struct {
	crd *CustomReportDelete
}

// Where appends a list predicates to the CustomReportDelete builder.
func (crdo *CustomReportDeleteOne) Where(ps ...predicate.CustomReport) *CustomReportDeleteOne {
	crdo.crd.mutation.Where(ps...)
	return crdo
}

// Exec executes the deletion query.
func (crdo *CustomReportDeleteOne) Exec(ctx context.Context) error {
	n, err := crdo.crd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{customreport.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (crdo *CustomReportDeleteOne) ExecX(ctx context.Context) {
	if err := crdo.Exec(ctx); err != nil {
		panic(err)
	}
}
