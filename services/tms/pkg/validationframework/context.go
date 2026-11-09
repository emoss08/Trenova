package validationframework

import (
	"context"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

type Context struct {
	Mode           Mode
	EntityID       pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Original       any
	memo           map[any]memoEntry
}

type memoEntry struct {
	value any
	err   error
}

func NewContext(ctx context.Context, entity Entity, mode Mode) *Context {
	valCtx := &Context{Mode: mode}
	if mode == ModeUpdate {
		valCtx.EntityID = entity.GetID()
	}

	if tenant, ok := dbscope.TenantFrom(ctx); ok && tenant.Valid() {
		valCtx.OrganizationID = tenant.OrganizationID
		valCtx.BusinessUnitID = tenant.BusinessUnitID
	} else if owned, isOwned := entity.(TenantOwned); isOwned {
		valCtx.OrganizationID = owned.GetOrganizationID()
		valCtx.BusinessUnitID = owned.GetBusinessUnitID()
	}

	return valCtx
}

func (c *Context) IsCreate() bool {
	return c.Mode == ModeCreate
}

func (c *Context) IsUpdate() bool {
	return c.Mode == ModeUpdate
}

func OriginalOf[T any](valCtx *Context) (T, bool) {
	var zero T
	if valCtx == nil || valCtx.Original == nil {
		return zero, false
	}

	original, ok := valCtx.Original.(T)
	return original, ok
}

type MemoKey[V any] struct {
	name string
}

func NewMemoKey[V any](name string) MemoKey[V] {
	return MemoKey[V]{name: name}
}

func Remember[V any](
	ctx context.Context,
	valCtx *Context,
	key MemoKey[V],
	load func(context.Context) (V, error),
) (V, error) {
	if valCtx == nil {
		return load(ctx)
	}
	if entry, ok := valCtx.memo[key]; ok {
		value, _ := entry.value.(V)
		return value, entry.err
	}

	value, err := load(ctx)
	if valCtx.memo == nil {
		valCtx.memo = make(map[any]memoEntry, 4)
	}
	valCtx.memo[key] = memoEntry{value: value, err: err}

	return value, err
}
