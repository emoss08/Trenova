package carrierresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCarrierInput() gqlmodel.CarrierInput {
	return gqlmodel.CarrierInput{
		Status:           carrier.StatusActive,
		Code:             "SWFT",
		Name:             "Swift Haulers",
		CarrierType:      carrier.Type("Common"),
		ComplianceStatus: carrier.ComplianceStatus("Qualified"),
		SafetyRating:     carrier.SafetyRating("Satisfactory"),
		PaymentMethod:    carrier.PaymentMethod("Check"),
		PaymentTermDays:  30,
	}
}

func TestCarrierMutations_CheckTheCarrierPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("car_").String()
	input := newCarrierInput()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createCarrier", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateCarrier(ctx, input)
			return err
		}},
		{"updateCarrier", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateCarrier(ctx, id, input)
			return err
		}},
		{"patchCarrier", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).PatchCarrier(ctx, id, gqlmodel.CarrierPatchInput{})
			return err
		}},
		{
			"bulkUpdateCarrierStatus",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).BulkUpdateCarrierStatus(
					ctx,
					gqlmodel.BulkUpdateCarrierStatusInput{
						CarrierIds: []string{id},
						Status:     carrier.StatusInactive,
					},
				)
				return err
			},
		},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(t, err, engine, permission.ResourceCarrier, call.operation)
		})
	}
}

func TestBulkUpdateCarrierStatus_RefusesAnAPIKey(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}}}
	authCtx := resolvertest.UserAuthContext()
	authCtx.PrincipalType = authctx.PrincipalTypeAPIKey
	ctx := resolvertest.AuthedContext(t, authCtx)

	_, err := (&MutationResolver{deps}).BulkUpdateCarrierStatus(
		ctx,
		gqlmodel.BulkUpdateCarrierStatusInput{
			CarrierIds: []string{pulid.MustNew("car_").String()},
			Status:     carrier.StatusInactive,
		},
	)

	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "%v", err)
}

func TestCarrierFromInput_MapsTheCarrierIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	input := newCarrierInput()
	stateID := pulid.MustNew("us_")
	stateRaw := stateID.String()
	dot := "1234567"
	taxType := carrier.TaxIDType("EIN")
	qualifiedAt := 1_700_000_000
	input.StateID = &stateRaw
	input.DOTNumber = &dot
	input.TaxIDType = &taxType
	input.QualifiedAt = &qualifiedAt

	entity, err := carrierFromInput(&input, authCtx)
	require.NoError(t, err)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	require.NotNil(t, entity.StateID)
	assert.Equal(t, stateID, *entity.StateID)
	assert.Nil(t, entity.RemitStateID)
	assert.Equal(t, "1234567", entity.DOTNumber)
	assert.Equal(t, &taxType, entity.TaxIDType)
	assert.Equal(t, int64(1_700_000_000), *entity.QualifiedAt)
	assert.Equal(t, 30, entity.PaymentTermDays)
	assert.Empty(t, entity.Contacts)
	assert.Empty(t, entity.InsurancePolicies)
}

func TestApplyCarrierInput_ReplacesContactsAndPoliciesAndKeepsEDIChannels(t *testing.T) {
	t.Parallel()

	savedContactID := pulid.MustNew("carc_")
	savedContactRaw := savedContactID.String()
	channels := []*carrier.CarrierEDIChannel{{ID: pulid.MustNew("cec_"), IsDefault: true}}
	existing := &carrier.Carrier{
		ID:          pulid.MustNew("car_"),
		Contacts:    []*carrier.CarrierContact{{ID: savedContactID}, {ID: pulid.MustNew("carc_")}},
		EDIChannels: channels,
	}
	input := newCarrierInput()
	email := "dana@swift.test"
	contactVersion := 2
	input.Contacts = []*gqlmodel.CarrierContactInput{
		{
			ID:                        &savedContactRaw,
			Name:                      "Dana Reyes",
			Email:                     &email,
			IsPrimary:                 true,
			ReceivesRateConfirmations: true,
			Version:                   &contactVersion,
		},
		{Name: "Lee Park"},
	}
	input.InsurancePolicies = []*gqlmodel.CarrierInsurancePolicyInput{
		{
			PolicyType:     carrier.InsurancePolicyType("AutoLiability"),
			PolicyNumber:   "AL-1",
			ProviderName:   "Shield Mutual",
			CoverageAmount: "1000000.50",
			EffectiveDate:  1_700_000_000,
			ExpirationDate: 1_731_536_000,
		},
	}

	require.NoError(t, applyCarrierInput(existing, &input))

	require.Len(t, existing.Contacts, 2)
	assert.Equal(t, savedContactID, existing.Contacts[0].ID)
	assert.Equal(t, "dana@swift.test", existing.Contacts[0].Email)
	assert.Equal(t, int64(2), existing.Contacts[0].Version)
	assert.True(t, existing.Contacts[1].ID.IsNil())
	assert.Equal(t, "Lee Park", existing.Contacts[1].Name)
	require.Len(t, existing.InsurancePolicies, 1)
	assert.True(
		t,
		decimal.RequireFromString("1000000.50").Equal(existing.InsurancePolicies[0].CoverageAmount),
	)
	assert.Equal(t, int64(1_731_536_000), existing.InsurancePolicies[0].ExpirationDate)
	assert.Equal(t, channels, existing.EDIChannels)
}

func TestApplyCarrierInput_NamesAMalformedCoverageAmount(t *testing.T) {
	t.Parallel()

	input := newCarrierInput()
	input.InsurancePolicies = []*gqlmodel.CarrierInsurancePolicyInput{
		{PolicyNumber: "AL-1", CoverageAmount: "a lot"},
	}

	err := applyCarrierInput(&carrier.Carrier{}, &input)

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "insurancePolicies[0].coverageAmount", valErr.Field)
}

func TestApplyCarrierPatch_ChangesOnlyTheFieldsItCarries(t *testing.T) {
	t.Parallel()

	contacts := []*carrier.CarrierContact{{ID: pulid.MustNew("carc_")}}
	existing := &carrier.Carrier{
		Status:   carrier.StatusActive,
		Name:     "Swift Haulers",
		Notes:    "Prefers email",
		Phone:    "555-0100",
		Contacts: contacts,
	}
	inactive := carrier.StatusInactive

	err := applyCarrierPatch(existing, &gqlmodel.CarrierPatchInput{
		Status: graphql.OmittableOf(&inactive),
		Notes:  graphql.OmittableOf[*string](nil),
	})
	require.NoError(t, err)

	assert.Equal(t, carrier.StatusInactive, existing.Status)
	assert.Empty(t, existing.Notes)
	assert.Equal(t, "Swift Haulers", existing.Name)
	assert.Equal(t, "555-0100", existing.Phone)
	assert.Equal(t, contacts, existing.Contacts)
}

func TestApplyCarrierPatch_RefusesToClearTheName(t *testing.T) {
	t.Parallel()

	existing := &carrier.Carrier{Name: "Swift Haulers"}

	err := applyCarrierPatch(existing, &gqlmodel.CarrierPatchInput{
		Name: graphql.OmittableOf[*string](nil),
	})

	resolvertest.RequireRequiredPatchError(t, err, "name")
	assert.Equal(t, "Swift Haulers", existing.Name)
}
