package reportresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogEntityReadsInTheRequestersLanguage(t *testing.T) {
	entity := &reportcatalog.Entity{
		Key:         "customer",
		Resource:    permission.ResourceCustomer,
		Label:       "Customer",
		PluralLabel: "Customers",
		Fields: []reportcatalog.Field{{
			Key:   "status",
			Label: "Status",
			EnumValues: []reportcatalog.EnumValue{
				{Value: "Active", Label: "Active"},
			},
		}},
		Edges: []reportcatalog.Edge{{Name: "parent", Label: "Customer"}},
	}
	decided := &reportaccess.Entity{
		Entity:   entity,
		Readable: true,
		Fields:   []reportaccess.Use{{Decision: reportaccess.Available}},
		Edges:    []reportaccess.Use{{Decision: reportaccess.Available}},
	}

	english := catalogEntityToModel(i18n.EN, nil, decided)
	spanish := catalogEntityToModel(i18n.ES, nil, decided)

	assert.Equal(t, "Customer", english.Label)
	require.Len(t, spanish.Fields, 1)
	require.Len(t, spanish.Fields[0].EnumValues, 1)
	require.Len(t, spanish.Edges, 1)

	for source, got := range map[string]string{
		"Customer":  spanish.Label,
		"Customers": spanish.PluralLabel,
		"Status":    spanish.Fields[0].Label,
		"Active":    spanish.Fields[0].EnumValues[0].Label,
	} {
		want := i18n.Translate(i18n.ES, source)
		require.NotEqual(t, source, want, "the Spanish catalog has no entry for %q", source)
		assert.Equal(t, want, got)
	}
	assert.Equal(t, spanish.Label, spanish.Edges[0].Label)
	assert.Equal(t, "customer", spanish.Key)
	assert.Equal(t, "Active", spanish.Fields[0].EnumValues[0].Value)
}

func TestCatalogNamesWhatEachFieldNeeds(t *testing.T) {
	entity := &reportcatalog.Entity{
		Key:         "worker",
		Resource:    permission.ResourceWorker,
		Label:       "Worker",
		PluralLabel: "Workers",
		Fields: []reportcatalog.Field{
			{Key: "firstName", Label: "First Name", Class: reportcatalog.ClassReference},
			{Key: "phoneNumber", Label: "Phone Number"},
			{Key: "licenseNumber", Label: "License Number", Protected: permission.FieldGroupPersonal},
		},
		Edges: []reportcatalog.Edge{{Name: "profile", Label: "Profile"}},
	}
	decided := &reportaccess.Entity{
		Entity: entity,
		Fields: []reportaccess.Use{
			{Decision: reportaccess.Available, Reference: true},
			{Decision: reportaccess.NeedsKindRead, Kind: permission.ResourceWorker},
			{
				Decision: reportaccess.NeedsFieldGroup,
				Group:    permission.FieldGroupPersonal,
			},
		},
		Edges: []reportaccess.Use{
			{Decision: reportaccess.NeedsKindRead, Kind: permission.ResourceWorker},
		},
	}

	model := catalogEntityToModel(i18n.EN, nil, decided)

	assert.False(t, model.Readable)
	require.Len(t, model.Fields, 3)
	assert.Equal(t, gqlmodel.ReportFieldAccessAvailable, model.Fields[0].Access)
	assert.True(t, model.Fields[0].InReferenceView)
	assert.Nil(t, model.Fields[0].Protection)
	assert.Equal(t, gqlmodel.ReportFieldAccessNeedsKindRead, model.Fields[1].Access)
	assert.Equal(t, gqlmodel.ReportFieldAccessNeedsFieldGroup, model.Fields[2].Access)
	require.NotNil(t, model.Fields[2].Protection)
	assert.Equal(t, permission.FieldGroupPersonal, model.Fields[2].Protection.Group)
	assert.False(t, model.Fields[2].Protection.Held)
	require.Len(t, model.Edges, 1)
	assert.Equal(t, gqlmodel.ReportFieldAccessNeedsKindRead, model.Edges[0].Access)
	require.NotNil(t, model.Edges[0].NeedsReadOf)
	assert.Equal(t, "worker", model.Edges[0].NeedsReadOf.Resource)
	assert.Equal(t, "worker", model.Edges[0].NeedsReadOf.Label)

	named := catalogEntityToModel(i18n.EN, reportaccess.New(reportaccess.Params{}), decided)
	assert.Equal(t, "Workers", named.Edges[0].NeedsReadOf.Label)
}
