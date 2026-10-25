package reportengine

import (
	"context"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/trenova/internal/keeltables"
	"github.com/emoss08/trenova/shared/i18n"
)

func NewCatalog() (*analytics.Catalog, error) {
	return analytics.Build(keeltables.All(), analytics.Options{Translate: Translate})
}

func Translate(ctx context.Context, _, fallback string) string {
	return i18n.Translate(i18n.FromContext(ctx), fallback)
}
