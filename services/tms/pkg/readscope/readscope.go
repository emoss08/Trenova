package readscope

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/shared/pulid"
)

const (
	ScopeOwn  = "own"
	ScopeTeam = "team"
	ScopeNone = "none"

	GinContextKey = "trenova.readscope"
)

var (
	ErrUnsafeValue = errors.New("read scope value is not safe to send to the database")

	safeName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	safeID   = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
)

type Grant struct {
	Scope string
	Team  []pulid.ID
}

type Grants map[string]Grant

type Resolver func(ctx context.Context) (Grants, error)

type Holder struct {
	once    sync.Once
	resolve Resolver
	grants  Grants
	encoded string
	err     error
}

type ctxKey struct{}

func With(ctx context.Context, resolve Resolver) context.Context {
	return context.WithValue(ctx, ctxKey{}, &Holder{resolve: resolve})
}

func Current(ctx context.Context) *Holder {
	if holder, ok := ctx.Value(ctxKey{}).(*Holder); ok {
		return holder
	}
	if holder, ok := ctx.Value(GinContextKey).(*Holder); ok {
		return holder
	}
	return nil
}

func Without(ctx context.Context) context.Context {
	if Current(ctx) == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, (*Holder)(nil))
}

func Encode(ctx context.Context) (string, error) {
	holder := active(ctx)
	if holder == nil {
		return "", nil
	}
	holder.load(ctx)

	return holder.encoded, holder.err
}

// Narrows reports whether the request reads fewer records of kind than its
// tenant holds. Anything that answers for the database without asking it,
// a cache in particular, must not answer a request it narrows.
func Narrows(ctx context.Context, kind string) (bool, error) {
	holder := active(ctx)
	if holder == nil {
		return false, nil
	}
	holder.load(ctx)
	if holder.err != nil {
		return true, holder.err
	}

	grant, ok := holder.grants[kind]
	return ok && narrow(grant.Scope), nil
}

func active(ctx context.Context) *Holder {
	if holder, ok := ctx.Value(ctxKey{}).(*Holder); ok && holder == nil {
		return nil
	}
	holder := Current(ctx)
	if holder == nil || holder.resolve == nil {
		return nil
	}
	return holder
}

func (h *Holder) load(ctx context.Context) {
	h.once.Do(func() {
		grants, err := h.resolve(Without(ctx))
		if err != nil {
			h.err = fmt.Errorf("resolve the records this request may read: %w", err)
			return
		}
		h.grants = grants
		h.encoded, h.err = grants.Encode()
	})
}

func narrow(scope string) bool {
	return scope == ScopeOwn || scope == ScopeTeam || scope == ScopeNone
}

func (g Grants) Encode() (string, error) {
	kinds := make([]string, 0, len(g))
	for kind, grant := range g {
		if narrow(grant.Scope) {
			kinds = append(kinds, kind)
		}
	}
	slices.Sort(kinds)

	parts := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		grant := g[kind]
		if !safeName.MatchString(kind) {
			return "", fmt.Errorf("%w: kind %q", ErrUnsafeValue, kind)
		}
		part := kind + ":" + grant.Scope
		if grant.Scope == ScopeTeam {
			team, err := teamIDs(grant.Team)
			if err != nil {
				return "", err
			}
			part += ":" + strings.Join(team, ",")
		}
		parts = append(parts, part)
	}

	return strings.Join(parts, ";"), nil
}

func teamIDs(ids []pulid.ID) ([]string, error) {
	team := make([]string, 0, len(ids))
	for _, id := range ids {
		value := id.String()
		if !safeID.MatchString(value) {
			return nil, fmt.Errorf("%w: team member %q", ErrUnsafeValue, value)
		}
		team = append(team, value)
	}
	slices.Sort(team)
	return slices.Compact(team), nil
}
