package gqlctx

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/vektah/gqlparser/v2/ast"
)

type fieldCeilingKey struct{}

type ResourceCeilings interface {
	For(ctx context.Context, resource permission.Resource) permission.FieldSensitivity
	Reads(ctx context.Context, resource permission.Resource) bool
}

// FieldCeiling is an agent read's data access ceiling and the record of what
// it withheld. Only the agent read path sets one; an HTTP request never
// carries it, so the screen is gated by @protected groups alone.
type FieldCeiling struct {
	ceilings ResourceCeilings

	mu       sync.Mutex
	withheld map[string]struct{}
	strip    []ast.Path
}

func NewFieldCeiling(ceilings ResourceCeilings) *FieldCeiling {
	return &FieldCeiling{ceilings: ceilings, withheld: make(map[string]struct{}, 4)}
}

func WithFieldCeiling(ctx context.Context, ceiling *FieldCeiling) context.Context {
	return context.WithValue(ctx, fieldCeilingKey{}, ceiling)
}

func FieldCeilingFrom(ctx context.Context) (*FieldCeiling, bool) {
	ceiling, ok := ctx.Value(fieldCeilingKey{}).(*FieldCeiling)
	return ceiling, ok && ceiling != nil
}

func (c *FieldCeiling) For(
	ctx context.Context,
	resource permission.Resource,
) permission.FieldSensitivity {
	if c.ceilings == nil {
		return permission.SensitivityInternal
	}

	return c.ceilings.For(ctx, resource)
}

// Reads reports whether the reader may read a resource at all.
func (c *FieldCeiling) Reads(ctx context.Context, resource permission.Resource) bool {
	if c.ceilings == nil {
		return true
	}

	return c.ceilings.Reads(ctx, resource)
}

// Withhold records a field the read may not show. A field answered null
// needs nothing more; strip marks a non-null one whose value the reader must
// remove from the result, because nulling it would null its parent.
func (c *FieldCeiling) Withhold(path ast.Path, strip bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.withheld[withheldName(path)] = struct{}{}
	if strip {
		c.strip = append(c.strip, slices.Clone(path))
	}
}

// Withheld is every withheld field, by its path without list indices, sorted.
func (c *FieldCeiling) Withheld() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]string, 0, len(c.withheld))
	for name := range c.withheld {
		out = append(out, name)
	}
	slices.Sort(out)

	return out
}

// Stripped is every non-null withheld value's full path, for the reader to
// remove from the result.
func (c *FieldCeiling) Stripped() []ast.Path {
	c.mu.Lock()
	defer c.mu.Unlock()

	return slices.Clone(c.strip)
}

func withheldName(path ast.Path) string {
	var b strings.Builder
	for _, element := range path {
		name, ok := element.(ast.PathName)
		if !ok {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('.')
		}
		b.WriteString(string(name))
	}

	return b.String()
}
