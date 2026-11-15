package pageaccessresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/identity/app/pageaccessservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/pageaccess"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageAccessToModel_NamesTheAddOnAKindIsWithheldBy(t *testing.T) {
	t.Parallel()

	denial := permissionports.NewPlanDenial(platformcatalog.FeatureFactoring)
	model := pageAccessToModel(&pageaccessservice.Access{
		Path: "/billing/invoices/$id",
		Kinds: []pageaccessservice.Kind{
			{
				Resource:     "factoring_company",
				ResourceName: "Factoring company",
				Use:          pageaccess.UseReference,
				Fields: []pageaccessservice.Field{
					{Name: "name", InView: true, Decision: pageaccessservice.DecisionWithheldPlan},
				},
				Withheld:   1,
				PlanDenial: &denial,
			},
			{
				Resource:     "customer",
				ResourceName: "Customer",
				Use:          pageaccess.UseReference,
				Fields: []pageaccessservice.Field{
					{Name: "name", InView: true, Decision: pageaccessservice.DecisionVisible},
					{Name: "creditLimit", Decision: pageaccessservice.DecisionWithheldKind},
				},
				Withheld: 1,
			},
		},
	})

	require.Len(t, model.Kinds, 2)
	factoring := model.Kinds[0]
	assert.Equal(t, &gqlmodel.PageAccessPlanDenial{
		Feature:     string(platformcatalog.FeatureFactoring),
		FeatureName: "Factoring",
		AddOn:       string(denial.AddOn),
		AddOnName:   denial.AddOnName,
	}, factoring.PlanDenial)
	assert.Equal(t, gqlmodel.PageAccessDecisionWithheldPlan, factoring.Fields[0].Decision)

	customer := model.Kinds[1]
	assert.Nil(t, customer.PlanDenial)
	assert.Equal(t, gqlmodel.PageAccessDecisionVisible, customer.Fields[0].Decision)
	assert.Equal(t, gqlmodel.PageAccessDecisionWithheldKind, customer.Fields[1].Decision)
}
