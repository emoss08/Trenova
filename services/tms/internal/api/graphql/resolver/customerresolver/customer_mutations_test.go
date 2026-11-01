package customerresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/trackingprofile"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func customerInput() gqlmodel.CustomerInput {
	return gqlmodel.CustomerInput{
		Status:                 domaintypes.StatusActive,
		Code:                   "ACME",
		Name:                   "Acme Freight",
		AddressLine1:           "1 Main St",
		City:                   "Columbus",
		StateID:                pulid.MustNew("us_").String(),
		PostalCode:             "43004",
		AllowConsolidation:     true,
		ConsolidationPriority:  1,
		StatusUpdatePreference: customer.StatusUpdateNone,
	}
}

func TestCustomerMutations_CheckTheCustomerPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("cus_").String()
	input := customerInput()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createCustomer", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateCustomer(ctx, input)
			return err
		}},
		{"updateCustomer", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateCustomer(ctx, id, input)
			return err
		}},
		{"patchCustomer", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).PatchCustomer(
				ctx,
				id,
				gqlmodel.CustomerPatchInput{},
			)
			return err
		}},
		{
			"bulkUpdateCustomerStatus",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).BulkUpdateCustomerStatus(
					ctx,
					gqlmodel.BulkUpdateCustomerStatusInput{
						CustomerIds: []string{id},
						Status:      domaintypes.StatusInactive,
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

			resolvertest.RequireRefused(t, err, engine, permission.ResourceCustomer, call.operation)
		})
	}
}

func TestBulkUpdateCustomerStatus_RefusesAnAPIKey(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}}}
	authCtx := resolvertest.UserAuthContext()
	authCtx.PrincipalType = authctx.PrincipalTypeAPIKey

	_, err := (&MutationResolver{deps}).BulkUpdateCustomerStatus(
		resolvertest.AuthedContext(t, authCtx),
		gqlmodel.BulkUpdateCustomerStatusInput{
			CustomerIds: []string{pulid.MustNew("cus_").String()},
			Status:      domaintypes.StatusInactive,
		},
	)

	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "%v", err)
}

func TestCustomerFromInput_MapsTheCustomerAndItsProfiles(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	input := customerInput()
	dot := "1234567"
	interval := 4
	creditLimit := "25000.50"
	documentTypeID := pulid.MustNew("dt_")
	revenueAccountID := pulid.MustNew("gla_")
	revenueAccountRaw := revenueAccountID.String()
	toRecipients := "ap@acme.test"
	input.DOTNumber = &dot
	input.BrokerVettingEnabled = true
	input.CheckCallIntervalHours = &interval
	input.BillingProfile = &gqlmodel.CustomerBillingProfileInput{
		BillingCycleAnchorDay:  15,
		BillingCycleTimezone:   "America/New_York",
		MaxShipmentsPerInvoice: 20,
		InvoiceCopies:          2,
		GracePeriodDays:        5,
		CreditLimit:            &creditLimit,
		RevenueAccountID:       &revenueAccountRaw,
		EDIInvoiceEnabled:      true,
		RequirePONumber:        true,
		BillingCurrency:        "USD",
		DocumentTypeIds:        []string{documentTypeID.String()},
	}
	input.EmailProfile = &gqlmodel.CustomerEmailProfileInput{
		ToRecipients: &toRecipients,
		ReadReceipt:  true,
	}

	entity, err := customerFromInput(&input, authCtx)
	require.NoError(t, err)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, input.StateID, entity.StateID.String())
	assert.Equal(t, "1234567", entity.DOTNumber)
	assert.True(t, entity.BrokerVettingEnabled)
	require.NotNil(t, entity.CheckCallIntervalHours)
	assert.Equal(t, int16(4), *entity.CheckCallIntervalHours)
	assert.Nil(t, entity.EtaRiskThresholdMinutes)

	profile := entity.BillingProfile
	require.NotNil(t, profile)
	assert.Equal(t, int16(15), profile.BillingCycleAnchorDay)
	assert.Equal(t, int16(20), profile.MaxShipmentsPerInvoice)
	assert.Equal(t, int16(2), profile.InvoiceCopies)
	assert.Equal(t, int16(5), profile.GracePeriodDays)
	assert.True(t, profile.CreditLimit.Valid)
	assert.True(t, decimal.RequireFromString("25000.50").Equal(profile.CreditLimit.Decimal))
	assert.False(t, profile.LateChargeRate.Valid)
	require.NotNil(t, profile.RevenueAccountID)
	assert.Equal(t, revenueAccountID, *profile.RevenueAccountID)
	assert.Nil(t, profile.ARAccountID)
	assert.True(t, profile.EDIInvoiceEnabled)
	assert.True(t, profile.RequirePONumber)
	require.Len(t, profile.DocumentTypes, 1)
	assert.Equal(t, documentTypeID, profile.DocumentTypes[0].ID)

	require.NotNil(t, entity.EmailProfile)
	assert.Equal(t, "ap@acme.test", entity.EmailProfile.ToRecipients)
	assert.True(t, entity.EmailProfile.ReadReceipt)
}

func TestApplyCustomerInput_LeavesProfilesTheInputOmits(t *testing.T) {
	t.Parallel()

	profileID := pulid.MustNew("cbp_")
	existing := &customer.Customer{
		BillingProfile: &customer.CustomerBillingProfile{
			ID:            profileID,
			CreditBalance: decimal.RequireFromString("120"),
		},
		Version: 9,
	}

	input := customerInput()

	require.NoError(t, applyCustomerInput(existing, &input))

	require.NotNil(t, existing.BillingProfile)
	assert.Equal(t, profileID, existing.BillingProfile.ID)
	assert.True(t, decimal.RequireFromString("120").Equal(existing.BillingProfile.CreditBalance))
	assert.Nil(t, existing.EmailProfile)
	assert.Nil(t, existing.TrackingProfile)
	assert.Equal(t, int64(9), existing.Version)
}

func TestApplyCustomerInput_SetsTheTrackingSettingsOnTheStoredProfile(t *testing.T) {
	t.Parallel()

	profileID := pulid.MustNew("ctp_")
	enabledAt := int64(1_700_000_000)
	existing := &customer.Customer{
		TrackingProfile: &trackingprofile.Profile{
			ID:                     profileID,
			NotifyDelivered:        true,
			NotificationsEnabledAt: &enabledAt,
		},
	}
	recipients := "ops@acme.test"
	input := customerInput()
	input.TrackingProfile = &gqlmodel.CustomerTrackingProfileInput{
		LinkEnabled:     false,
		VisibleFields:   []trackingprofile.Field{trackingprofile.FieldEta},
		LinkTTLDays:     14,
		NotifyDelivered: true,
		NotifyPodReady:  true,
		IncludeLink:     true,
		Recipients:      &recipients,
	}

	require.NoError(t, applyCustomerInput(existing, &input))

	profile := existing.TrackingProfile
	require.NotNil(t, profile)
	assert.Equal(t, profileID, profile.ID)
	assert.False(t, profile.LinkEnabled)
	assert.Equal(t, []trackingprofile.Field{trackingprofile.FieldEta}, profile.VisibleFields)
	assert.Equal(t, int16(14), profile.LinkTTLDays)
	assert.True(t, profile.NotifyDelivered)
	assert.True(t, profile.NotifyPodReady)
	assert.False(t, profile.NotifyPickedUp)
	assert.True(t, profile.IncludeLink)
	assert.Equal(t, "ops@acme.test", profile.Recipients)
	assert.Equal(t, &enabledAt, profile.NotificationsEnabledAt, "the service stamps the start time")
}

func TestClampLinkTTLDays_KeepsAnOutOfRangeNumberFromWrapping(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int16(0), clampLinkTTLDays(-5))
	assert.Equal(t, int16(30), clampLinkTTLDays(30))
	assert.Equal(t, int16(181), clampLinkTTLDays(1_000_000))
}

func TestApplyCustomerInput_KeepsTheCreditReviewOnTheExistingProfile(t *testing.T) {
	t.Parallel()

	reviewedAt := int64(1_700_000_000)
	existing := &customer.Customer{
		BillingProfile: &customer.CustomerBillingProfile{
			CreditBalance:       decimal.RequireFromString("800"),
			CreditHoldAutomatic: true,
			CreditReviewedAt:    &reviewedAt,
		},
	}
	input := customerInput()
	input.BillingProfile = &gqlmodel.CustomerBillingProfileInput{BillingCurrency: "CAD"}

	require.NoError(t, applyCustomerInput(existing, &input))

	assert.Equal(t, "CAD", existing.BillingProfile.BillingCurrency)
	assert.True(t, decimal.RequireFromString("800").Equal(existing.BillingProfile.CreditBalance))
	assert.True(t, existing.BillingProfile.CreditHoldAutomatic)
	assert.Equal(t, &reviewedAt, existing.BillingProfile.CreditReviewedAt)
	assert.Empty(t, existing.BillingProfile.DocumentTypes)
}

func TestApplyCustomerInput_NamesAMalformedCreditLimit(t *testing.T) {
	t.Parallel()

	creditLimit := "lots"
	input := customerInput()
	input.BillingProfile = &gqlmodel.CustomerBillingProfileInput{CreditLimit: &creditLimit}

	err := applyCustomerInput(&customer.Customer{}, &input)

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "billingProfile.creditLimit", valErr.Field)
}

func TestApplyCustomerInput_RequiresAState(t *testing.T) {
	t.Parallel()

	input := customerInput()
	input.StateID = ""

	err := applyCustomerInput(&customer.Customer{}, &input)

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "stateId", valErr.Field)
}

func TestApplyCustomerPatch_ChangesOnlyTheFieldsItCarries(t *testing.T) {
	t.Parallel()

	interval := int16(6)
	existing := &customer.Customer{
		Status:                 domaintypes.StatusActive,
		Name:                   "Acme",
		DOTNumber:              "123",
		CheckCallIntervalHours: &interval,
	}
	inactive := domaintypes.StatusInactive

	err := applyCustomerPatch(existing, &gqlmodel.CustomerPatchInput{
		Status:                 graphql.OmittableOf(&inactive),
		DOTNumber:              graphql.OmittableOf[*string](nil),
		CheckCallIntervalHours: graphql.OmittableOf[*int](nil),
	})
	require.NoError(t, err)

	assert.Equal(t, domaintypes.StatusInactive, existing.Status)
	assert.Equal(t, "Acme", existing.Name)
	assert.Empty(t, existing.DOTNumber)
	assert.Nil(t, existing.CheckCallIntervalHours)
}

func TestApplyCustomerPatch_RefusesToClearTheState(t *testing.T) {
	t.Parallel()

	err := applyCustomerPatch(
		&customer.Customer{StateID: pulid.MustNew("us_")},
		&gqlmodel.CustomerPatchInput{StateID: graphql.OmittableOf[*string](nil)},
	)

	resolvertest.RequireRequiredPatchError(t, err, "stateId")
}
