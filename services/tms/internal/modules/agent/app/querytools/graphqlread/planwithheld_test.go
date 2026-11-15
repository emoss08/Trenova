package graphqlread

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
)

func TestPlanWithheld_NamesTheAddOnEachFieldIsPartOf(t *testing.T) {
	t.Parallel()

	sentences := PlanWithheld([]graphqlreadports.WithheldByPlan{
		{Path: "invoice.factoringCompany", Feature: platformcatalog.FeatureFactoring},
		{Path: "invoice.factoringStatus", Feature: platformcatalog.FeatureFactoring},
		{Path: "invoice.detentionCharge", Feature: platformcatalog.FeatureDetention},
	}, "invoice.")

	assert.Equal(t, []string{
		"detentionCharge is part of the Rating & Revenue Recovery add-on (Detention), which " +
			"this organization's plan does not include. Do not look it up another way; say " +
			"the plan does not include it.",
		"factoringCompany, factoringStatus are part of the Accounting & Cash add-on " +
			"(Factoring), which this organization's plan does not include. Do not look them " +
			"up another way; say the plan does not include them.",
	}, sentences)
}

func TestPlanWithheld_IsEmptyWhenThePlanWithheldNothing(t *testing.T) {
	t.Parallel()

	assert.Nil(t, PlanWithheld(nil, "invoice."))
}

func TestPrefixedPlanWithheld_NamesFieldsUnderTheKeyTheyAreShownUnder(t *testing.T) {
	t.Parallel()

	sentences := PrefixedPlanWithheld([]graphqlreadports.WithheldByPlan{
		{Path: "customer.invoices.factoringCompany", Feature: platformcatalog.FeatureFactoring},
	}, "customer.invoices.", "openInvoices")

	assert.Len(t, sentences, 1)
	assert.Contains(t, sentences[0], "openInvoices.factoringCompany is part of the Accounting")
}
