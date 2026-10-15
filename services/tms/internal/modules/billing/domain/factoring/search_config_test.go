package factoring_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
)

type searchable interface {
	GetPostgresSearchConfig() domaintypes.PostgresSearchConfig
}

func columnsOf(entity any) map[string]struct{} {
	columns := make(map[string]struct{})
	t := reflect.TypeOf(entity).Elem()
	for i := range t.NumField() {
		tag := t.Field(i).Tag.Get("bun")
		if tag == "" || strings.HasPrefix(tag, "rel:") || strings.HasPrefix(tag, "table:") {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name != "" {
			columns[name] = struct{}{}
		}
	}
	return columns
}

func TestEverySearchableFieldIsAColumn(t *testing.T) {
	t.Parallel()

	for _, entity := range []searchable{
		&factoring.InvoiceState{},
		&factoring.Noa{},
		&factoring.Remittance{},
		&factoring.RemittanceLine{},
		&factoring.ReserveEntry{},
		&factoring.Submission{},
	} {
		columns := columnsOf(entity)
		for _, field := range entity.GetPostgresSearchConfig().SearchableFields {
			_, ok := columns[field.Name]
			assert.True(t, ok, "%T searches %q, which is not one of its columns",
				entity, field.Name)
		}
	}
}
