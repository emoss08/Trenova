package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func splitReadinessFixture(t *testing.T) (*shipment.Shipment, *shipment.ShareResolution, pulid.ID) {
	t.Helper()

	entity := validShipmentForValidation()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipment.StatusCompleted
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(1000))
	entity.RatingDetail = &shipment.RatingDetail{Result: 1000}
	entity.AdditionalCharges = []*shipment.AdditionalCharge{}
	amd := pulid.MustNew("cus_")
	entity.ChargeAllocations = []*shipment.ChargeAllocation{
		{
			ID:               pulid.MustNew("chal_"),
			ChargeKind:       shipment.ChargeAllocationKindFreight,
			BillToCustomerID: entity.CustomerID,
			Method:           shipment.ChargeAllocationMethodPercent,
			Percent:          decimal.NewNullDecimal(decimal.NewFromInt(60)),
		},
		{
			ID:               pulid.MustNew("chal_"),
			ChargeKind:       shipment.ChargeAllocationKindFreight,
			BillToCustomerID: amd,
			Method:           shipment.ChargeAllocationMethodPercent,
			Percent:          decimal.NewNullDecimal(decimal.NewFromInt(40)),
			Sequence:         1,
		},
	}

	resolution, err := shipment.ResolveShares(entity, entity.ChargeAllocations)
	require.NoError(t, err)

	return entity, resolution, amd
}

func autoTransferControl() *tenant.BillingControl {
	return &tenant.BillingControl{
		ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelBlock,
		BillingQueueTransferMode:              tenant.BillingQueueTransferModeAutomaticWhenReady,
		ReadyToBillAssignmentMode:             tenant.ReadyToBillAssignmentModeAutomaticWhenEligible,
	}
}

func TestApplyPayerReadiness_UnionsTheSecondPayersRequirements(t *testing.T) {
	t.Parallel()

	entity, resolution, amd := splitReadinessFixture(t)
	podType := &documenttype.DocumentType{
		ID:   pulid.MustNew("dt_"),
		Code: "POD",
		Name: "Proof of Delivery",
	}
	poType := &documenttype.DocumentType{
		ID:   pulid.MustNew("dt_"),
		Code: "PO",
		Name: "Purchase Order",
	}
	control := autoTransferControl()
	docs := []*document.Document{{ID: pulid.MustNew("doc_"), DocumentTypeID: &podType.ID}}

	intel := &customer.Customer{
		ID:   entity.CustomerID,
		Name: "Intel",
		Code: "INTEL",
		BillingProfile: &customer.CustomerBillingProfile{
			EnforceCustomerBillingReq: true,
			AutoTransfer:              true,
			AutoApprove:               true,
			DocumentTypes:             []*documenttype.DocumentType{podType},
		},
	}
	amdCustomer := &customer.Customer{
		ID:   amd,
		Name: "AMD",
		Code: "AMD",
		BillingProfile: &customer.CustomerBillingProfile{
			EnforceCustomerBillingReq: true,
			AutoTransfer:              true,
			AutoApprove:               true,
			RequireBOLNumber:          true,
			DocumentTypes:             []*documenttype.DocumentType{podType, poType},
		},
	}
	entity.BOL = ""

	readiness := buildShipmentBillingReadiness(entity, intel.BillingProfile, control, docs)
	require.Empty(t, readiness.MissingRequirements, "Intel alone is satisfied")
	require.True(t, readiness.ShouldAutoApproveBilling)

	applyPayerReadiness(readiness, entity, resolution, map[pulid.ID]*customer.Customer{
		intel.ID:       intel,
		amdCustomer.ID: amdCustomer,
	}, docs)

	require.Len(t, readiness.Requirements, 2, "the POD is shared; AMD adds the PO")
	require.Len(t, readiness.MissingRequirements, 1)
	assert.Equal(t, poType.ID.String(), readiness.MissingRequirements[0].DocumentTypeID)

	codes := make([]string, 0, len(readiness.ValidationFailures))
	for _, failure := range readiness.ValidationFailures {
		codes = append(codes, failure.Code)
	}
	assert.Contains(t, codes, "missing_bol", "AMD requires a BOL even though Intel does not")
	assert.NotContains(t, codes, "credit_hold")

	require.Len(t, readiness.Payers, 2)
	assert.Equal(t, entity.CustomerID, readiness.Payers[0].PayerID)
	assert.True(t, readiness.Payers[0].IsPrimary)
	assert.Equal(t, "Intel", readiness.Payers[0].PayerName)
	assert.Equal(t, "600.00", readiness.Payers[0].ShareAmount.StringFixed(2))
	assert.Equal(t, amd, readiness.Payers[1].PayerID)
	assert.False(t, readiness.Payers[1].IsPrimary)
	assert.Equal(t, "400.00", readiness.Payers[1].ShareAmount.StringFixed(2))

	assert.False(
		t,
		readiness.ShouldAutoApproveBilling,
		"a missing requirement stops auto-approval for everybody",
	)
	assert.False(t, readiness.CanMarkReadyToInvoice)
}

func TestApplyPayerReadiness_BlocksOnASecondaryPayersCreditHold(t *testing.T) {
	t.Parallel()

	entity, resolution, amd := splitReadinessFixture(t)
	control := autoTransferControl()

	intel := &customer.Customer{
		ID:             entity.CustomerID,
		Name:           "Intel",
		BillingProfile: &customer.CustomerBillingProfile{AutoTransfer: true, AutoApprove: true},
	}
	amdCustomer := &customer.Customer{
		ID:   amd,
		Name: "AMD",
		BillingProfile: &customer.CustomerBillingProfile{
			AutoTransfer:       true,
			AutoApprove:        true,
			EnforceCreditLimit: true,
			CreditStatus:       customer.CreditStatusHold,
		},
	}

	readiness := buildShipmentBillingReadiness(entity, intel.BillingProfile, control, nil)
	applyPayerReadiness(readiness, entity, resolution, map[pulid.ID]*customer.Customer{
		intel.ID:       intel,
		amdCustomer.ID: amdCustomer,
	}, nil)

	require.Len(t, readiness.ValidationFailures, 1)
	assert.Equal(t, "credit_hold", readiness.ValidationFailures[0].Code)
	assert.Equal(t, "billToCustomerId", readiness.ValidationFailures[0].Field)
	assert.Contains(t, readiness.ValidationFailures[0].Message, "AMD")

	require.Len(t, readiness.Payers, 2)
	assert.False(t, readiness.Payers[0].CreditHold)
	assert.True(t, readiness.Payers[1].CreditHold)
	assert.Equal(t, customer.CreditStatusHold, readiness.Payers[1].CreditStatus)
	assert.False(t, readiness.ShouldAutoApproveBilling)
	assert.False(
		t,
		readiness.CanMarkReadyToInvoice,
		"Block enforcement treats a held payer like a missing document",
	)
}

func TestApplyPayerReadiness_CreditHoldWithoutEnforcementIsOnlyReported(t *testing.T) {
	t.Parallel()

	entity, resolution, amd := splitReadinessFixture(t)

	payers := map[pulid.ID]*customer.Customer{
		entity.CustomerID: {
			ID:             entity.CustomerID,
			BillingProfile: &customer.CustomerBillingProfile{},
		},
		amd: {ID: amd, Name: "AMD", BillingProfile: &customer.CustomerBillingProfile{
			CreditStatus: customer.CreditStatusHold,
		}},
	}

	readiness := buildShipmentBillingReadiness(
		entity,
		payers[entity.CustomerID].BillingProfile,
		nil,
		nil,
	)
	applyPayerReadiness(readiness, entity, resolution, payers, nil)

	assert.Empty(t, readiness.ValidationFailures)
	assert.False(t, readiness.Payers[1].CreditHold)
	assert.Equal(t, customer.CreditStatusHold, readiness.Payers[1].CreditStatus)
}

func TestApplyPayerReadiness_AutoApprovalNeedsEveryPayer(t *testing.T) {
	t.Parallel()

	entity, resolution, amd := splitReadinessFixture(t)
	control := autoTransferControl()

	tests := []struct {
		name         string
		intelOptsIn  bool
		amdOptsIn    bool
		wantApproval bool
	}{
		{name: "both opt in", intelOptsIn: true, amdOptsIn: true, wantApproval: true},
		{
			name:         "only the primary opts in",
			intelOptsIn:  true,
			amdOptsIn:    false,
			wantApproval: false,
		},
		{
			name:         "only the secondary opts in",
			intelOptsIn:  false,
			amdOptsIn:    true,
			wantApproval: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			intel := &customer.Customer{
				ID: entity.CustomerID,
				BillingProfile: &customer.CustomerBillingProfile{
					AutoTransfer: true,
					AutoApprove:  tc.intelOptsIn,
				},
			}
			amdCustomer := &customer.Customer{
				ID: amd,
				BillingProfile: &customer.CustomerBillingProfile{
					AutoTransfer: true,
					AutoApprove:  tc.amdOptsIn,
				},
			}

			readiness := buildShipmentBillingReadiness(entity, intel.BillingProfile, control, nil)
			applyPayerReadiness(readiness, entity, resolution, map[pulid.ID]*customer.Customer{
				intel.ID:       intel,
				amdCustomer.ID: amdCustomer,
			}, nil)

			assert.Equal(t, tc.wantApproval, readiness.ShouldAutoApproveBilling)
			assert.Equal(t, tc.intelOptsIn, readiness.Payers[0].ShouldAutoApproveBilling)
			assert.Equal(t, tc.amdOptsIn, readiness.Payers[1].ShouldAutoApproveBilling)
		})
	}
}

func TestApplyPayerReadiness_ToleratesMissingInputs(t *testing.T) {
	t.Parallel()

	entity, resolution, _ := splitReadinessFixture(t)
	readiness := buildShipmentBillingReadiness(entity, nil, nil, nil)

	applyPayerReadiness(nil, entity, resolution, nil, nil)
	applyPayerReadiness(readiness, entity, nil, nil, nil)
	assert.Nil(t, readiness.Payers)

	applyPayerReadiness(readiness, entity, resolution, map[pulid.ID]*customer.Customer{}, nil)
	require.Len(t, readiness.Payers, 2, "unknown payers are still listed, unnamed")
	assert.Empty(t, readiness.Payers[1].PayerName)
	assert.False(t, readiness.ShouldAutoApproveBilling)
}

func TestPreviewForResolutionMapsPositionalAllocationsToPlaceholderIDs(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(500))
	saved := &shipment.AdditionalCharge{
		ID:     pulid.MustNew("ac_"),
		Method: "Flat",
		Amount: decimal.NewFromInt(50),
		Unit:   1,
	}
	unsaved := &shipment.AdditionalCharge{Method: "Flat", Amount: decimal.NewFromInt(80), Unit: 1}
	entity.AdditionalCharges = []*shipment.AdditionalCharge{saved, nil, unsaved}

	payer := pulid.MustNew("cus_")
	savedID := saved.ID
	unsavedIndex := 2
	entity.ChargeAllocations = []*shipment.ChargeAllocation{
		{
			ChargeKind:         shipment.ChargeAllocationKindAccessorial,
			AdditionalChargeID: &savedID,
			BillToCustomerID:   payer,
			Method:             shipment.ChargeAllocationMethodPercent,
			Percent:            decimal.NewNullDecimal(decimal.NewFromInt(100)),
		},
		{
			ChargeKind:            shipment.ChargeAllocationKindAccessorial,
			AdditionalChargeIndex: &unsavedIndex,
			BillToCustomerID:      payer,
			Method:                shipment.ChargeAllocationMethodPercent,
			Percent:               decimal.NewNullDecimal(decimal.NewFromInt(100)),
		},
		nil,
	}

	preview := previewForResolution(entity)

	require.Len(t, preview.AdditionalCharges, 3)
	assert.Equal(t, saved.ID, preview.AdditionalCharges[0].ID)
	assert.Nil(t, preview.AdditionalCharges[1])
	assert.True(
		t,
		preview.AdditionalCharges[2].ID.IsNotNil(),
		"the id-less charge gets a placeholder",
	)
	assert.True(t, unsaved.ID.IsNil(), "the caller's charge is untouched")

	require.Len(t, preview.ChargeAllocations, 2)
	require.NotNil(t, preview.ChargeAllocations[1].AdditionalChargeID)
	assert.Equal(
		t,
		preview.AdditionalCharges[2].ID,
		*preview.ChargeAllocations[1].AdditionalChargeID,
	)
	assert.Nil(
		t,
		entity.ChargeAllocations[1].AdditionalChargeID,
		"the caller's allocation is untouched",
	)

	resolution, err := shipment.ResolveShares(preview, preview.ChargeAllocations)
	require.NoError(t, err)
	share := resolution.ShareFor(payer)
	require.NotNil(t, share)
	assert.True(
		t,
		share.AccessorialAmount.Equal(decimal.NewFromInt(130)),
		"both charges resolve against the payer",
	)
}
