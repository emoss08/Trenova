package tablequeryresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/agent/app/tablequeryservice"
	"github.com/emoss08/trenova/pkg/filtercatalog"
)

func catalogueToModel(catalog *filtercatalog.Catalog) []*gqlmodel.TableQueryCatalogueResource {
	resources := catalog.Resources()
	out := make([]*gqlmodel.TableQueryCatalogueResource, 0, len(resources))
	for _, resource := range resources {
		fields := make([]*gqlmodel.TableQueryCatalogueField, 0, len(resource.Fields))
		for _, field := range resource.Fields {
			operators := filtercatalog.Operators(field.Kind)
			names := make([]string, 0, len(operators))
			for _, operator := range operators {
				names = append(names, string(operator))
			}
			values := field.Values
			if values == nil {
				values = []string{}
			}
			fields = append(fields, &gqlmodel.TableQueryCatalogueField{
				Name:      field.Name,
				Kind:      string(field.Kind),
				Note:      field.Note,
				Values:    values,
				Sortable:  field.Sortable,
				Operators: names,
			})
		}
		out = append(out, &gqlmodel.TableQueryCatalogueResource{
			Resource: resource.Resource.String(),
			Entity:   resource.Entity,
			Summary:  resource.Summary,
			Fields:   fields,
		})
	}

	return out
}

func currentViewFromInput(
	input *gqlmodel.TableQueryCurrentViewInput,
) tablequeryservice.CurrentView {
	if input == nil {
		return tablequeryservice.CurrentView{}
	}

	return tablequeryservice.CurrentView{
		Query:        gqlscalar.StringValue(input.Query),
		FieldFilters: gqlconnection.FieldFiltersFromGraphQL(input.FieldFilters),
		Sort:         gqlconnection.SortFieldsFromGraphQL(input.Sort),
	}
}
