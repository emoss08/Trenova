package gqlctx

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestFieldCeiling_KeepsPlanWithholdingsApartFromAccessOnes(t *testing.T) {
	t.Parallel()

	ceiling := NewFieldCeiling(nil)
	ceiling.Withhold(ast.Path{ast.PathName("invoice"), ast.PathName("customer")}, false)
	ceiling.WithholdByPlan(ast.Path{
		ast.PathName("invoices"), ast.PathIndex(0), ast.PathName("factoringCompany"),
	}, platformcatalog.FeatureFactoring, false)
	ceiling.WithholdByPlan(ast.Path{
		ast.PathName("invoices"), ast.PathIndex(1), ast.PathName("factoringCompany"),
	}, platformcatalog.FeatureFactoring, false)
	ceiling.WithholdByPlan(ast.Path{ast.PathName("invoice"), ast.PathName("detention")},
		platformcatalog.FeatureDetention, true)

	assert.Equal(t, []string{"invoice.customer"}, ceiling.Withheld())
	assert.Equal(t, []WithheldByPlan{
		{Path: "invoice.detention", Feature: platformcatalog.FeatureDetention},
		{Path: "invoices.factoringCompany", Feature: platformcatalog.FeatureFactoring},
	}, ceiling.WithheldByPlan(), "by path without list indices, once each")
	assert.Len(t, ceiling.Stripped(), 1, "a non-null plan withholding is stripped too")
}
