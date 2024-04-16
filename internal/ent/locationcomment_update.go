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
	"github.com/emoss08/trenova/internal/ent/commenttype"
	"github.com/emoss08/trenova/internal/ent/location"
	"github.com/emoss08/trenova/internal/ent/locationcomment"
	"github.com/emoss08/trenova/internal/ent/predicate"
	"github.com/emoss08/trenova/internal/ent/user"
	"github.com/google/uuid"
)

// LocationCommentUpdate is the builder for updating LocationComment entities.
type LocationCommentUpdate struct {
	config
	hooks     []Hook
	mutation  *LocationCommentMutation
	modifiers []func(*sql.UpdateBuilder)
}

// Where appends a list predicates to the LocationCommentUpdate builder.
func (lcu *LocationCommentUpdate) Where(ps ...predicate.LocationComment) *LocationCommentUpdate {
	lcu.mutation.Where(ps...)
	return lcu
}

// SetUpdatedAt sets the "updated_at" field.
func (lcu *LocationCommentUpdate) SetUpdatedAt(t time.Time) *LocationCommentUpdate {
	lcu.mutation.SetUpdatedAt(t)
	return lcu
}

// SetVersion sets the "version" field.
func (lcu *LocationCommentUpdate) SetVersion(i int) *LocationCommentUpdate {
	lcu.mutation.ResetVersion()
	lcu.mutation.SetVersion(i)
	return lcu
}

// SetNillableVersion sets the "version" field if the given value is not nil.
func (lcu *LocationCommentUpdate) SetNillableVersion(i *int) *LocationCommentUpdate {
	if i != nil {
		lcu.SetVersion(*i)
	}
	return lcu
}

// AddVersion adds i to the "version" field.
func (lcu *LocationCommentUpdate) AddVersion(i int) *LocationCommentUpdate {
	lcu.mutation.AddVersion(i)
	return lcu
}

// SetLocationID sets the "location_id" field.
func (lcu *LocationCommentUpdate) SetLocationID(u uuid.UUID) *LocationCommentUpdate {
	lcu.mutation.SetLocationID(u)
	return lcu
}

// SetNillableLocationID sets the "location_id" field if the given value is not nil.
func (lcu *LocationCommentUpdate) SetNillableLocationID(u *uuid.UUID) *LocationCommentUpdate {
	if u != nil {
		lcu.SetLocationID(*u)
	}
	return lcu
}

// SetUserID sets the "user_id" field.
func (lcu *LocationCommentUpdate) SetUserID(u uuid.UUID) *LocationCommentUpdate {
	lcu.mutation.SetUserID(u)
	return lcu
}

// SetNillableUserID sets the "user_id" field if the given value is not nil.
func (lcu *LocationCommentUpdate) SetNillableUserID(u *uuid.UUID) *LocationCommentUpdate {
	if u != nil {
		lcu.SetUserID(*u)
	}
	return lcu
}

// SetCommentTypeID sets the "comment_type_id" field.
func (lcu *LocationCommentUpdate) SetCommentTypeID(u uuid.UUID) *LocationCommentUpdate {
	lcu.mutation.SetCommentTypeID(u)
	return lcu
}

// SetNillableCommentTypeID sets the "comment_type_id" field if the given value is not nil.
func (lcu *LocationCommentUpdate) SetNillableCommentTypeID(u *uuid.UUID) *LocationCommentUpdate {
	if u != nil {
		lcu.SetCommentTypeID(*u)
	}
	return lcu
}

// SetComment sets the "comment" field.
func (lcu *LocationCommentUpdate) SetComment(s string) *LocationCommentUpdate {
	lcu.mutation.SetComment(s)
	return lcu
}

// SetNillableComment sets the "comment" field if the given value is not nil.
func (lcu *LocationCommentUpdate) SetNillableComment(s *string) *LocationCommentUpdate {
	if s != nil {
		lcu.SetComment(*s)
	}
	return lcu
}

// SetLocation sets the "location" edge to the Location entity.
func (lcu *LocationCommentUpdate) SetLocation(l *Location) *LocationCommentUpdate {
	return lcu.SetLocationID(l.ID)
}

// SetUser sets the "user" edge to the User entity.
func (lcu *LocationCommentUpdate) SetUser(u *User) *LocationCommentUpdate {
	return lcu.SetUserID(u.ID)
}

// SetCommentType sets the "comment_type" edge to the CommentType entity.
func (lcu *LocationCommentUpdate) SetCommentType(c *CommentType) *LocationCommentUpdate {
	return lcu.SetCommentTypeID(c.ID)
}

// Mutation returns the LocationCommentMutation object of the builder.
func (lcu *LocationCommentUpdate) Mutation() *LocationCommentMutation {
	return lcu.mutation
}

// ClearLocation clears the "location" edge to the Location entity.
func (lcu *LocationCommentUpdate) ClearLocation() *LocationCommentUpdate {
	lcu.mutation.ClearLocation()
	return lcu
}

// ClearUser clears the "user" edge to the User entity.
func (lcu *LocationCommentUpdate) ClearUser() *LocationCommentUpdate {
	lcu.mutation.ClearUser()
	return lcu
}

// ClearCommentType clears the "comment_type" edge to the CommentType entity.
func (lcu *LocationCommentUpdate) ClearCommentType() *LocationCommentUpdate {
	lcu.mutation.ClearCommentType()
	return lcu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (lcu *LocationCommentUpdate) Save(ctx context.Context) (int, error) {
	lcu.defaults()
	return withHooks(ctx, lcu.sqlSave, lcu.mutation, lcu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (lcu *LocationCommentUpdate) SaveX(ctx context.Context) int {
	affected, err := lcu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (lcu *LocationCommentUpdate) Exec(ctx context.Context) error {
	_, err := lcu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (lcu *LocationCommentUpdate) ExecX(ctx context.Context) {
	if err := lcu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (lcu *LocationCommentUpdate) defaults() {
	if _, ok := lcu.mutation.UpdatedAt(); !ok {
		v := locationcomment.UpdateDefaultUpdatedAt()
		lcu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (lcu *LocationCommentUpdate) check() error {
	if v, ok := lcu.mutation.Comment(); ok {
		if err := locationcomment.CommentValidator(v); err != nil {
			return &ValidationError{Name: "comment", err: fmt.Errorf(`ent: validator failed for field "LocationComment.comment": %w`, err)}
		}
	}
	if _, ok := lcu.mutation.BusinessUnitID(); lcu.mutation.BusinessUnitCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "LocationComment.business_unit"`)
	}
	if _, ok := lcu.mutation.OrganizationID(); lcu.mutation.OrganizationCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "LocationComment.organization"`)
	}
	if _, ok := lcu.mutation.LocationID(); lcu.mutation.LocationCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "LocationComment.location"`)
	}
	if _, ok := lcu.mutation.UserID(); lcu.mutation.UserCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "LocationComment.user"`)
	}
	if _, ok := lcu.mutation.CommentTypeID(); lcu.mutation.CommentTypeCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "LocationComment.comment_type"`)
	}
	return nil
}

// Modify adds a statement modifier for attaching custom logic to the UPDATE statement.
func (lcu *LocationCommentUpdate) Modify(modifiers ...func(u *sql.UpdateBuilder)) *LocationCommentUpdate {
	lcu.modifiers = append(lcu.modifiers, modifiers...)
	return lcu
}

func (lcu *LocationCommentUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := lcu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(locationcomment.Table, locationcomment.Columns, sqlgraph.NewFieldSpec(locationcomment.FieldID, field.TypeUUID))
	if ps := lcu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := lcu.mutation.UpdatedAt(); ok {
		_spec.SetField(locationcomment.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := lcu.mutation.Version(); ok {
		_spec.SetField(locationcomment.FieldVersion, field.TypeInt, value)
	}
	if value, ok := lcu.mutation.AddedVersion(); ok {
		_spec.AddField(locationcomment.FieldVersion, field.TypeInt, value)
	}
	if value, ok := lcu.mutation.Comment(); ok {
		_spec.SetField(locationcomment.FieldComment, field.TypeString, value)
	}
	if lcu.mutation.LocationCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   locationcomment.LocationTable,
			Columns: []string{locationcomment.LocationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(location.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := lcu.mutation.LocationIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   locationcomment.LocationTable,
			Columns: []string{locationcomment.LocationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(location.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if lcu.mutation.UserCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   locationcomment.UserTable,
			Columns: []string{locationcomment.UserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := lcu.mutation.UserIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   locationcomment.UserTable,
			Columns: []string{locationcomment.UserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if lcu.mutation.CommentTypeCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   locationcomment.CommentTypeTable,
			Columns: []string{locationcomment.CommentTypeColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(commenttype.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := lcu.mutation.CommentTypeIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   locationcomment.CommentTypeTable,
			Columns: []string{locationcomment.CommentTypeColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(commenttype.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_spec.AddModifiers(lcu.modifiers...)
	if n, err = sqlgraph.UpdateNodes(ctx, lcu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{locationcomment.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	lcu.mutation.done = true
	return n, nil
}

// LocationCommentUpdateOne is the builder for updating a single LocationComment entity.
type LocationCommentUpdateOne struct {
	config
	fields    []string
	hooks     []Hook
	mutation  *LocationCommentMutation
	modifiers []func(*sql.UpdateBuilder)
}

// SetUpdatedAt sets the "updated_at" field.
func (lcuo *LocationCommentUpdateOne) SetUpdatedAt(t time.Time) *LocationCommentUpdateOne {
	lcuo.mutation.SetUpdatedAt(t)
	return lcuo
}

// SetVersion sets the "version" field.
func (lcuo *LocationCommentUpdateOne) SetVersion(i int) *LocationCommentUpdateOne {
	lcuo.mutation.ResetVersion()
	lcuo.mutation.SetVersion(i)
	return lcuo
}

// SetNillableVersion sets the "version" field if the given value is not nil.
func (lcuo *LocationCommentUpdateOne) SetNillableVersion(i *int) *LocationCommentUpdateOne {
	if i != nil {
		lcuo.SetVersion(*i)
	}
	return lcuo
}

// AddVersion adds i to the "version" field.
func (lcuo *LocationCommentUpdateOne) AddVersion(i int) *LocationCommentUpdateOne {
	lcuo.mutation.AddVersion(i)
	return lcuo
}

// SetLocationID sets the "location_id" field.
func (lcuo *LocationCommentUpdateOne) SetLocationID(u uuid.UUID) *LocationCommentUpdateOne {
	lcuo.mutation.SetLocationID(u)
	return lcuo
}

// SetNillableLocationID sets the "location_id" field if the given value is not nil.
func (lcuo *LocationCommentUpdateOne) SetNillableLocationID(u *uuid.UUID) *LocationCommentUpdateOne {
	if u != nil {
		lcuo.SetLocationID(*u)
	}
	return lcuo
}

// SetUserID sets the "user_id" field.
func (lcuo *LocationCommentUpdateOne) SetUserID(u uuid.UUID) *LocationCommentUpdateOne {
	lcuo.mutation.SetUserID(u)
	return lcuo
}

// SetNillableUserID sets the "user_id" field if the given value is not nil.
func (lcuo *LocationCommentUpdateOne) SetNillableUserID(u *uuid.UUID) *LocationCommentUpdateOne {
	if u != nil {
		lcuo.SetUserID(*u)
	}
	return lcuo
}

// SetCommentTypeID sets the "comment_type_id" field.
func (lcuo *LocationCommentUpdateOne) SetCommentTypeID(u uuid.UUID) *LocationCommentUpdateOne {
	lcuo.mutation.SetCommentTypeID(u)
	return lcuo
}

// SetNillableCommentTypeID sets the "comment_type_id" field if the given value is not nil.
func (lcuo *LocationCommentUpdateOne) SetNillableCommentTypeID(u *uuid.UUID) *LocationCommentUpdateOne {
	if u != nil {
		lcuo.SetCommentTypeID(*u)
	}
	return lcuo
}

// SetComment sets the "comment" field.
func (lcuo *LocationCommentUpdateOne) SetComment(s string) *LocationCommentUpdateOne {
	lcuo.mutation.SetComment(s)
	return lcuo
}

// SetNillableComment sets the "comment" field if the given value is not nil.
func (lcuo *LocationCommentUpdateOne) SetNillableComment(s *string) *LocationCommentUpdateOne {
	if s != nil {
		lcuo.SetComment(*s)
	}
	return lcuo
}

// SetLocation sets the "location" edge to the Location entity.
func (lcuo *LocationCommentUpdateOne) SetLocation(l *Location) *LocationCommentUpdateOne {
	return lcuo.SetLocationID(l.ID)
}

// SetUser sets the "user" edge to the User entity.
func (lcuo *LocationCommentUpdateOne) SetUser(u *User) *LocationCommentUpdateOne {
	return lcuo.SetUserID(u.ID)
}

// SetCommentType sets the "comment_type" edge to the CommentType entity.
func (lcuo *LocationCommentUpdateOne) SetCommentType(c *CommentType) *LocationCommentUpdateOne {
	return lcuo.SetCommentTypeID(c.ID)
}

// Mutation returns the LocationCommentMutation object of the builder.
func (lcuo *LocationCommentUpdateOne) Mutation() *LocationCommentMutation {
	return lcuo.mutation
}

// ClearLocation clears the "location" edge to the Location entity.
func (lcuo *LocationCommentUpdateOne) ClearLocation() *LocationCommentUpdateOne {
	lcuo.mutation.ClearLocation()
	return lcuo
}

// ClearUser clears the "user" edge to the User entity.
func (lcuo *LocationCommentUpdateOne) ClearUser() *LocationCommentUpdateOne {
	lcuo.mutation.ClearUser()
	return lcuo
}

// ClearCommentType clears the "comment_type" edge to the CommentType entity.
func (lcuo *LocationCommentUpdateOne) ClearCommentType() *LocationCommentUpdateOne {
	lcuo.mutation.ClearCommentType()
	return lcuo
}

// Where appends a list predicates to the LocationCommentUpdate builder.
func (lcuo *LocationCommentUpdateOne) Where(ps ...predicate.LocationComment) *LocationCommentUpdateOne {
	lcuo.mutation.Where(ps...)
	return lcuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (lcuo *LocationCommentUpdateOne) Select(field string, fields ...string) *LocationCommentUpdateOne {
	lcuo.fields = append([]string{field}, fields...)
	return lcuo
}

// Save executes the query and returns the updated LocationComment entity.
func (lcuo *LocationCommentUpdateOne) Save(ctx context.Context) (*LocationComment, error) {
	lcuo.defaults()
	return withHooks(ctx, lcuo.sqlSave, lcuo.mutation, lcuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (lcuo *LocationCommentUpdateOne) SaveX(ctx context.Context) *LocationComment {
	node, err := lcuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (lcuo *LocationCommentUpdateOne) Exec(ctx context.Context) error {
	_, err := lcuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (lcuo *LocationCommentUpdateOne) ExecX(ctx context.Context) {
	if err := lcuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (lcuo *LocationCommentUpdateOne) defaults() {
	if _, ok := lcuo.mutation.UpdatedAt(); !ok {
		v := locationcomment.UpdateDefaultUpdatedAt()
		lcuo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (lcuo *LocationCommentUpdateOne) check() error {
	if v, ok := lcuo.mutation.Comment(); ok {
		if err := locationcomment.CommentValidator(v); err != nil {
			return &ValidationError{Name: "comment", err: fmt.Errorf(`ent: validator failed for field "LocationComment.comment": %w`, err)}
		}
	}
	if _, ok := lcuo.mutation.BusinessUnitID(); lcuo.mutation.BusinessUnitCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "LocationComment.business_unit"`)
	}
	if _, ok := lcuo.mutation.OrganizationID(); lcuo.mutation.OrganizationCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "LocationComment.organization"`)
	}
	if _, ok := lcuo.mutation.LocationID(); lcuo.mutation.LocationCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "LocationComment.location"`)
	}
	if _, ok := lcuo.mutation.UserID(); lcuo.mutation.UserCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "LocationComment.user"`)
	}
	if _, ok := lcuo.mutation.CommentTypeID(); lcuo.mutation.CommentTypeCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "LocationComment.comment_type"`)
	}
	return nil
}

// Modify adds a statement modifier for attaching custom logic to the UPDATE statement.
func (lcuo *LocationCommentUpdateOne) Modify(modifiers ...func(u *sql.UpdateBuilder)) *LocationCommentUpdateOne {
	lcuo.modifiers = append(lcuo.modifiers, modifiers...)
	return lcuo
}

func (lcuo *LocationCommentUpdateOne) sqlSave(ctx context.Context) (_node *LocationComment, err error) {
	if err := lcuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(locationcomment.Table, locationcomment.Columns, sqlgraph.NewFieldSpec(locationcomment.FieldID, field.TypeUUID))
	id, ok := lcuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "LocationComment.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := lcuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, locationcomment.FieldID)
		for _, f := range fields {
			if !locationcomment.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != locationcomment.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := lcuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := lcuo.mutation.UpdatedAt(); ok {
		_spec.SetField(locationcomment.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := lcuo.mutation.Version(); ok {
		_spec.SetField(locationcomment.FieldVersion, field.TypeInt, value)
	}
	if value, ok := lcuo.mutation.AddedVersion(); ok {
		_spec.AddField(locationcomment.FieldVersion, field.TypeInt, value)
	}
	if value, ok := lcuo.mutation.Comment(); ok {
		_spec.SetField(locationcomment.FieldComment, field.TypeString, value)
	}
	if lcuo.mutation.LocationCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   locationcomment.LocationTable,
			Columns: []string{locationcomment.LocationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(location.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := lcuo.mutation.LocationIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   locationcomment.LocationTable,
			Columns: []string{locationcomment.LocationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(location.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if lcuo.mutation.UserCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   locationcomment.UserTable,
			Columns: []string{locationcomment.UserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := lcuo.mutation.UserIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   locationcomment.UserTable,
			Columns: []string{locationcomment.UserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if lcuo.mutation.CommentTypeCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   locationcomment.CommentTypeTable,
			Columns: []string{locationcomment.CommentTypeColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(commenttype.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := lcuo.mutation.CommentTypeIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   locationcomment.CommentTypeTable,
			Columns: []string{locationcomment.CommentTypeColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(commenttype.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_spec.AddModifiers(lcuo.modifiers...)
	_node = &LocationComment{config: lcuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, lcuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{locationcomment.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	lcuo.mutation.done = true
	return _node, nil
}