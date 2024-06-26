// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/emoss08/trenova/internal/ent/predicate"
	"github.com/emoss08/trenova/internal/ent/usernotification"
)

// UserNotificationDelete is the builder for deleting a UserNotification entity.
type UserNotificationDelete struct {
	config
	hooks    []Hook
	mutation *UserNotificationMutation
}

// Where appends a list predicates to the UserNotificationDelete builder.
func (und *UserNotificationDelete) Where(ps ...predicate.UserNotification) *UserNotificationDelete {
	und.mutation.Where(ps...)
	return und
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (und *UserNotificationDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, und.sqlExec, und.mutation, und.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (und *UserNotificationDelete) ExecX(ctx context.Context) int {
	n, err := und.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (und *UserNotificationDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(usernotification.Table, sqlgraph.NewFieldSpec(usernotification.FieldID, field.TypeUUID))
	if ps := und.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, und.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	und.mutation.done = true
	return affected, err
}

// UserNotificationDeleteOne is the builder for deleting a single UserNotification entity.
type UserNotificationDeleteOne struct {
	und *UserNotificationDelete
}

// Where appends a list predicates to the UserNotificationDelete builder.
func (undo *UserNotificationDeleteOne) Where(ps ...predicate.UserNotification) *UserNotificationDeleteOne {
	undo.und.mutation.Where(ps...)
	return undo
}

// Exec executes the deletion query.
func (undo *UserNotificationDeleteOne) Exec(ctx context.Context) error {
	n, err := undo.und.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{usernotification.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (undo *UserNotificationDeleteOne) ExecX(ctx context.Context) {
	if err := undo.Exec(ctx); err != nil {
		panic(err)
	}
}
