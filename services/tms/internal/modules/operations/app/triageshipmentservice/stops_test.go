package triageshipmentservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStopTimesOf(t *testing.T) {
	t.Parallel()

	arrived := int64(1_791_000_000)
	occurrence := &detention.DetentionOccurrence{
		ID:             pulid.MustNew("dto_"),
		Currency:       "USD",
		LocationName:   " Louisville Store ",
		ArrivedAt:      &arrived,
		LateByMinutes:  -5,
		RoundedMinutes: 90,
		BillableAmount: decimal.RequireFromString("112.5"),
		PolicySnapshot: &detentionpolicy.PolicySnapshot{
			LateArrivalRule: detentionpolicy.LateArrivalRuleForfeit,
		},
		Evidence: []*detentionevidence.DetentionEvidence{
			{
				Kind:   detentionevidence.EvidenceKind("Departure"),
				Source: detentionevidence.EvidenceSourceManual,
			},
			{
				Kind:   detentionevidence.EvidenceKindArrival,
				Source: detentionevidence.EvidenceSourceTelematics,
			},
		},
	}

	out := stopTimesOf(occurrence)
	assert.Equal(t, occurrence.ID, out.OccurrenceID)
	assert.Equal(t, "USD", out.Currency)
	assert.Equal(t, "Louisville Store", out.StopName)
	assert.Equal(t, 0, out.LateMinutes)
	assert.Equal(t, 90, out.BilledDetentionMinutes)
	require.NotNil(t, out.BilledDetentionAmount)
	assert.Equal(t, "112.50", *out.BilledDetentionAmount)
	assert.False(t, out.DetentionPaysWhenLate)
	assert.Equal(t, "ELD", out.ArrivalSource)

	occurrence.BillableAmount = decimal.Zero
	occurrence.Evidence = nil
	occurrence.PolicySnapshot = nil
	out = stopTimesOf(occurrence)
	assert.Nil(t, out.BilledDetentionAmount)
	assert.True(t, out.DetentionPaysWhenLate)
	assert.Equal(t, arrivalSourceUnknown, out.ArrivalSource)
}

type fakeOccurrences struct {
	occurrences []*detention.DetentionOccurrence
}

func (f *fakeOccurrences) GetByShipment(
	context.Context,
	*detentionports.GetOccurrencesByShipmentRequest,
) ([]*detention.DetentionOccurrence, error) {
	return f.occurrences, nil
}

type fakeDecider struct {
	waived     []detentionservice.WaiveParams
	uninvoiced []detentionservice.WaiveParams
	disputed   []detentionservice.DisputeParams
}

func (f *fakeDecider) WaiveUninvoiced(
	_ context.Context,
	p *detentionservice.WaiveParams,
) (*detention.DetentionOccurrence, error) {
	f.uninvoiced = append(f.uninvoiced, *p)

	return &detention.DetentionOccurrence{
		ID:           p.OccurrenceID,
		Status:       detention.OccurrenceStatusWaived,
		WaivedAmount: decimal.RequireFromString("185"),
		Currency:     "USD",
	}, nil
}

func (f *fakeDecider) Waive(
	_ context.Context,
	p detentionservice.WaiveParams,
) (*detention.DetentionOccurrence, error) {
	f.waived = append(f.waived, p)

	return &detention.DetentionOccurrence{
		ID:           p.OccurrenceID,
		Status:       detention.OccurrenceStatusWaived,
		WaivedAmount: decimal.RequireFromString("185"),
		Currency:     "USD",
	}, nil
}

func (f *fakeDecider) Dispute(
	_ context.Context,
	p detentionservice.DisputeParams,
) (*detention.DetentionOccurrence, error) {
	f.disputed = append(f.disputed, p)

	return &detention.DetentionOccurrence{
		ID:             p.OccurrenceID,
		Status:         detention.OccurrenceStatusDisputed,
		BillableAmount: decimal.RequireFromString("185"),
		Currency:       "USD",
	}, nil
}

type fakeCreditor struct {
	credit   *detentionports.DetentionCredit
	err      error
	requests []*detentionports.CreditBilledDetentionRequest
}

func (f *fakeCreditor) CreditBilledCharge(
	_ context.Context,
	req *detentionports.CreditBilledDetentionRequest,
) (*detentionports.DetentionCredit, error) {
	f.requests = append(f.requests, req)

	return f.credit, f.err
}

type settleFixture struct {
	service    *Service
	occurrence *detention.DetentionOccurrence
	decider    *fakeDecider
	creditor   *fakeCreditor
}

func newSettleFixture(status detention.OccurrenceStatus) *settleFixture {
	chargeID := pulid.MustNew("sac_")
	occurrence := &detention.DetentionOccurrence{
		ID:                 pulid.MustNew("dto_"),
		ShipmentID:         pulid.MustNew("shp_"),
		Status:             status,
		BillableAmount:     decimal.RequireFromString("185"),
		Currency:           "USD",
		AdditionalChargeID: &chargeID,
	}
	f := &settleFixture{
		occurrence: occurrence,
		decider:    &fakeDecider{},
		creditor:   &fakeCreditor{},
	}
	f.service = &Service{
		l:           zap.NewNop(),
		occurrences: &fakeOccurrences{occurrences: []*detention.DetentionOccurrence{occurrence}},
		detention:   f.decider,
		credits:     f.creditor,
	}

	return f
}

func (f *settleFixture) settle(
	t *testing.T,
	decision inboundmessage.StopsDecision,
) (*inboundmessageports.SettledTriageDetention, error) {
	t.Helper()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	tenantInfo.UserID = pulid.MustNew("usr_")

	return f.service.SettleDetention(t.Context(), &inboundmessageports.SettleTriageDetentionRequest{
		TenantInfo: tenantInfo,
		Actor:      permissionports.UserActor(tenantInfo),
		ShipmentID: f.occurrence.ShipmentID,
		Decision:   decision,
		Note:       "Decided in Triage",
		UserID:     tenantInfo.UserID,
	})
}

func TestSettleDetentionWaivesAnUnbilledCharge(t *testing.T) {
	t.Parallel()

	f := newSettleFixture(detention.OccurrenceStatusApproved)

	settled, err := f.settle(t, inboundmessage.StopsDecisionWaive)
	require.NoError(t, err)

	require.Len(t, f.creditor.requests, 1)
	assert.Equal(t, *f.occurrence.AdditionalChargeID, f.creditor.requests[0].ChargeID)
	assert.Empty(t, f.decider.waived)
	require.Len(t, f.decider.uninvoiced, 1, "billing said no invoice bills it")
	assert.Equal(t, f.occurrence.ID, f.decider.uninvoiced[0].OccurrenceID)
	assert.Equal(t, detention.WaiverReasonOther, f.decider.uninvoiced[0].Reason)
	assert.True(t, settled.Settled)
	assert.Equal(t, "185.00", settled.Amount)
	assert.False(t, settled.Credited())
}

func TestSettleDetentionCreditsABilledCharge(t *testing.T) {
	t.Parallel()

	f := newSettleFixture(detention.OccurrenceStatusBilled)
	f.creditor.credit = &detentionports.DetentionCredit{
		Amount:   decimal.RequireFromString("185"),
		Currency: "USD",
		Invoices: []detentionports.DetentionCreditInvoice{{
			InvoiceNumber:    "INV-20931",
			CreditMemoNumber: "CM-00012",
			Amount:           decimal.RequireFromString("185"),
			Status:           detentionports.DetentionCreditIssued,
		}},
	}

	settled, err := f.settle(t, inboundmessage.StopsDecisionWaive)
	require.NoError(t, err)

	assert.Empty(t, f.decider.waived, "the credit waives the charge when it executes")
	assert.Empty(t, f.decider.uninvoiced)
	require.Len(t, f.creditor.requests, 1)
	request := f.creditor.requests[0]
	assert.Equal(t, f.occurrence.ID, request.Waiver.OccurrenceID)
	assert.Equal(t, request.TenantInfo.UserID, request.Waiver.RequestedBy)
	require.NotNil(t, request.Actor)
	assert.Equal(t, "185.00", settled.Amount)
	assert.Equal(t, []string{"INV-20931"}, settled.InvoiceNumbers)
	assert.Equal(t, []string{"CM-00012"}, settled.CreditMemoNumbers)
	assert.False(t, settled.CreditPending)
	assert.True(t, settled.Credited())
}

func TestSettleDetentionReportsACreditWaitingForApproval(t *testing.T) {
	t.Parallel()

	f := newSettleFixture(detention.OccurrenceStatusBilled)
	f.creditor.credit = &detentionports.DetentionCredit{
		Amount:   decimal.RequireFromString("185"),
		Currency: "USD",
		Invoices: []detentionports.DetentionCreditInvoice{{
			InvoiceNumber: "INV-20931",
			Status:        detentionports.DetentionCreditPendingApproval,
		}},
	}

	settled, err := f.settle(t, inboundmessage.StopsDecisionWaive)
	require.NoError(t, err)
	assert.True(t, settled.CreditPending)
	assert.Empty(t, settled.CreditMemoNumbers)
	assert.Empty(t, f.decider.waived)
}

func TestSettleDetentionPassesOnARefusedCredit(t *testing.T) {
	t.Parallel()

	f := newSettleFixture(detention.OccurrenceStatusBilled)
	f.creditor.err = errortypes.NewValidationError(
		"decision", errortypes.ErrInvalidOperation, "Invoice INV-20931 could not be credited",
	)

	_, err := f.settle(t, inboundmessage.StopsDecisionWaive)
	require.Error(t, err)
	fieldErr, ok := errors.AsType[*errortypes.Error](err)
	require.True(t, ok)
	assert.Equal(t, "decision", fieldErr.Field)
	assert.Empty(t, f.decider.waived)
}

func TestSettleDetentionStandsByWithoutCrediting(t *testing.T) {
	t.Parallel()

	f := newSettleFixture(detention.OccurrenceStatusBilled)

	settled, err := f.settle(t, inboundmessage.StopsDecisionStandBy)
	require.NoError(t, err)
	assert.Empty(t, f.creditor.requests)
	require.Len(t, f.decider.disputed, 1)
	assert.Equal(t, "185.00", settled.Amount)
}

func TestSettleDetentionWaivesAsBeforeWithoutBilling(t *testing.T) {
	t.Parallel()

	f := newSettleFixture(detention.OccurrenceStatusApproved)
	f.service.credits = nil

	settled, err := f.settle(t, inboundmessage.StopsDecisionWaive)
	require.NoError(t, err)
	require.Len(t, f.decider.waived, 1)
	assert.Empty(t, f.decider.uninvoiced)
	assert.Equal(t, "185.00", settled.Amount)
}

func TestSettleDetentionWaivesAChargeNeverPostedToTheShipment(t *testing.T) {
	t.Parallel()

	f := newSettleFixture(detention.OccurrenceStatusPending)
	f.occurrence.AdditionalChargeID = nil

	_, err := f.settle(t, inboundmessage.StopsDecisionWaive)
	require.NoError(t, err)
	assert.Empty(t, f.creditor.requests)
	require.Len(t, f.decider.uninvoiced, 1)
}

func TestSettleDetentionKeepsAnAlreadyWaivedCharge(t *testing.T) {
	t.Parallel()

	f := newSettleFixture(detention.OccurrenceStatusWaived)
	f.occurrence.WaivedAmount = decimal.RequireFromString("185")

	settled, err := f.settle(t, inboundmessage.StopsDecisionWaive)
	require.NoError(t, err)
	assert.Empty(t, f.creditor.requests)
	assert.Empty(t, f.decider.waived)
	assert.Empty(t, f.decider.uninvoiced)
	assert.Equal(t, "185.00", settled.Amount)
}

func TestDisputedOccurrenceStaysOnTheWaivedCharge(t *testing.T) {
	t.Parallel()

	f := newSettleFixture(detention.OccurrenceStatusWaived)
	f.occurrence.BillableAmount = decimal.Zero
	f.occurrence.WaivedAmount = decimal.RequireFromString("1200")
	f.occurrence.ClockStartAt = 100
	later := &detention.DetentionOccurrence{
		ID:           pulid.MustNew("dto_"),
		Status:       detention.OccurrenceStatusNotBillable,
		ClockStartAt: 200,
	}
	f.service.occurrences = &fakeOccurrences{
		occurrences: []*detention.DetentionOccurrence{f.occurrence, later},
	}

	chosen, err := f.service.disputedOccurrence(
		t.Context(),
		pagination.TenantInfo{},
		f.occurrence.ShipmentID,
		false,
	)
	require.NoError(t, err)
	assert.Equal(t, f.occurrence.ID, chosen.ID)
}
