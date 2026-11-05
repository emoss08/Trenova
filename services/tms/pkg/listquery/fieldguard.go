package listquery

import "context"

type FieldGuard func(table, field string) bool

type fieldGuardContextKey struct{}

func WithFieldGuard(ctx context.Context, guard FieldGuard) context.Context {
	return context.WithValue(ctx, fieldGuardContextKey{}, guard)
}

func fieldGuardFrom(ctx context.Context) FieldGuard {
	guard, _ := ctx.Value(fieldGuardContextKey{}).(FieldGuard)
	return guard
}
