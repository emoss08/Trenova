package volumegen

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
)

func Taken[T any](
	ctx context.Context,
	db *keel.DB,
	h keel.Handle[*T],
	column keel.Expr,
) (map[string]struct{}, error) {
	values, err := keel.Into[string](ctx, db, h.Pick(keel.StringOf(column).Lower().Coalesce("")))
	if err != nil {
		return nil, fmt.Errorf("load existing %s values: %w", h.Ref.Table.Name, err)
	}
	taken := make(map[string]struct{}, len(values))
	for _, v := range values {
		if v != "" {
			taken[v] = struct{}{}
		}
	}

	return taken, nil
}

func Slug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
		if b.Len() == 20 {
			break
		}
	}
	return b.String()
}
