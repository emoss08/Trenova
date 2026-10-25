//go:build integration

package rateagreementrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge/fuelsurchargedb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement/rateagreementdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func insertRow(t *testing.T, ctx context.Context, db *keel.DB, rows ...any) {
	t.Helper()

	for _, row := range rows {
		var err error
		switch v := row.(type) {
		case *customer.Customer:
			err = customerdb.Customers.Create(ctx, db, v)
		case *rateagreement.RateAgreement:
			err = rateagreementdb.RateAgreements.Create(ctx, db, v)
		case *rateagreement.RateAgreementRule:
			err = rateagreementdb.RateAgreementRules.Create(ctx, db, v)
		case *rateagreement.RateAgreementAccessorial:
			err = rateagreementdb.RateAgreementAccessorials.Create(ctx, db, v)
		case *rateagreement.RateAgreementFuelBinding:
			err = rateagreementdb.RateAgreementFuelBindings.Create(ctx, db, v)
		case *formulatemplate.FormulaTemplate:
			err = formulatemplatedb.FormulaTemplates.Create(ctx, db, v)
		case *accessorialcharge.AccessorialCharge:
			err = accessorialchargedb.AccessorialCharges.Create(ctx, db, v)
		case *fuelsurcharge.FuelIndex:
			err = fuelsurchargedb.FuelIndexes.Create(ctx, db, v)
		case *fuelsurcharge.FuelSurchargeProgram:
			err = fuelsurchargedb.FuelSurchargePrograms.Create(ctx, db, v)
		default:
			t.Fatalf("insertRow: no table for %T", row)
		}
		require.NoError(t, err)
	}
}

func scopedTo(ctx context.Context, info pagination.TenantInfo) context.Context {
	return dbscope.WithTenant(ctx, info.DBTenant())
}
