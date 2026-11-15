package gqlext

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/shared/textcase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

// restRecords are the domain types a REST response writes a record of each
// kind with, for the kinds that declare FieldFeatures: a field feature names
// a field of one of them or of a GraphQL type of the kind.
var restRecords = map[permission.Resource][]reflect.Type{
	permission.ResourceShipment:       {reflect.TypeFor[shipment.Shipment]()},
	permission.ResourceShipmentMove:   {reflect.TypeFor[shipment.ShipmentMove]()},
	permission.ResourceShipmentStop:   {reflect.TypeFor[shipment.Stop]()},
	permission.ResourceInvoice:        {reflect.TypeFor[invoice.Invoice]()},
	permission.ResourceBillingQueue:   {reflect.TypeFor[billingqueue.BillingQueueItem]()},
	permission.ResourceCustomer:       {reflect.TypeFor[customer.Customer]()},
	permission.ResourceCarrier:        {reflect.TypeFor[carrier.Carrier]()},
	permission.ResourceServiceFailure: {reflect.TypeFor[servicefailure.ServiceFailure]()},
	permission.ResourceDocument:       {reflect.TypeFor[document.Document]()},
	permission.ResourceTractor:        {reflect.TypeFor[tractor.Tractor]()},
	permission.ResourceWorker:         {reflect.TypeFor[worker.Worker]()},
	permission.ResourceOrganization:   {reflect.TypeFor[tenant.Organization]()},
}

// Every field a kind declares in FieldFeatures is a field of a GraphQL type
// of that kind or of a type it carries, or of the REST record of the kind, so
// a renamed field cannot leave its add-on's value unguarded.
func TestEveryFieldFeatureNamesAFieldOfItsKind(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	registry := fieldsensitivity.DefaultRegistry()
	kinds, err := KindTypes(schema, registry)
	require.NoError(t, err)

	graphql := make(map[permission.Resource]map[string]struct{})
	for name, kind := range kinds {
		def := schema.Types[name]
		holders := []*ast.Definition{def}
		for carried := range carriedTypes(schema, kinds, def) {
			holders = append(holders, schema.Types[carried])
		}
		for _, holder := range holders {
			for _, field := range holder.Fields {
				if graphql[kind] == nil {
					graphql[kind] = make(map[string]struct{})
				}
				graphql[kind][field.Name] = struct{}{}
			}
		}
	}

	var missing []string
	for _, def := range registry.All() {
		kind := permission.Resource(def.Resource)
		var rest map[string]struct{}
		for field := range def.FieldFeatures {
			if _, ok := graphql[kind][field]; ok {
				continue
			}
			if rest == nil {
				rest = restFieldNames(registry, restRecords[kind])
			}
			if _, ok := rest[field]; !ok {
				missing = append(missing, def.Resource+"."+field)
			}
		}
	}
	slices.Sort(missing)
	assert.Empty(t, missing,
		"FieldFeatures that name no field of a GraphQL type of their kind or of its REST "+
			"record. Fix the API name, or add the kind's domain type to restRecords:\n  %s",
		strings.Join(missing, "\n  "))
}

// restFieldNames is every JSON name a record writes, its plain structs' and
// parts' included, stopping at a struct of another kind.
func restFieldNames(
	registry *permissioncatalog.Registry,
	records []reflect.Type,
) map[string]struct{} {
	out := make(map[string]struct{})
	seen := make(map[reflect.Type]struct{})
	var walk func(t reflect.Type, root bool)
	walk = func(t reflect.Type, root bool) {
		for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return
		}
		if _, done := seen[t]; done {
			return
		}
		seen[t] = struct{}{}
		if !root && strings.Contains(t.PkgPath(), "/domain/") &&
			registry.HasResource(textcase.ConvertCamelToSnake(t.Name())) {
			return
		}
		for i := range t.NumField() {
			sf := t.Field(i)
			name, _, _ := strings.Cut(sf.Tag.Get("json"), ",")
			if name == "-" || !sf.IsExported() {
				continue
			}
			if name == "" && !sf.Anonymous {
				name = sf.Name
			}
			if name != "" {
				out[name] = struct{}{}
			}
			walk(sf.Type, false)
		}
	}
	for _, record := range records {
		walk(record, true)
	}

	return out
}
