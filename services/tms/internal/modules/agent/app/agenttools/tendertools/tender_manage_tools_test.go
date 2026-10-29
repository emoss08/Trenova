package tendertools

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderbounce"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercancel"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendernegotiation"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderresponse"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTenderManager plans from a tender it holds, as the service would, and
// records what a write was asked.
type fakeTenderManager struct {
	guard    agenttoolstest.WriteGuard
	entity   *tender.Tender
	refusal  error
	canceled *tendercancel.CancelTenderRequest
	answered *tenderports.TenderResponseRequest
	bounced  *tenderbounce.BounceTenderRequest
}

func newFakeTenderManager(status tender.Status, offers ...tender.OfferStatus) *fakeTenderManager {
	entity := &tender.Tender{
		ID:      pulid.MustNew("ten_"),
		Mode:    tender.ModeWaterfall,
		Status:  status,
		Version: 6,
	}
	for idx, offerStatus := range offers {
		entity.Offers = append(entity.Offers, &tender.TenderOffer{
			ID:        pulid.MustNew("tof_"),
			TenderID:  entity.ID,
			CarrierID: pulid.MustNew("car_"),
			Carrier: &carrier.Carrier{
				Name: []string{"Ridgeline", "Blue Mesa", "Crestway"}[idx%3],
			},
			Rank:           int16(idx + 1),
			RateMethod:     shipmentvocab.CarrierRateMethodFlat,
			Rate:           decimal.NewFromInt(int64(1500 + 100*idx)),
			Channel:        tender.ChannelEmail,
			RecipientEmail: "dispatch" + string(rune('a'+idx)) + "@carrier.test",
			Status:         offerStatus,
		})
	}

	return &fakeTenderManager{entity: entity}
}

func (f *fakeTenderManager) copyOffers(change func(*tender.TenderOffer)) *tender.Tender {
	out := *f.entity
	out.Offers = nil
	for _, offer := range f.entity.Offers {
		copied := *offer
		change(&copied)
		out.Offers = append(out.Offers, &copied)
	}

	return &out
}

func (f *fakeTenderManager) PreviewCancel(
	_ context.Context,
	req *tendercancel.CancelTenderRequest,
) (*tendercancel.CancelPreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	after := f.copyOffers(func(offer *tender.TenderOffer) {
		offer.Status, _ = tendercancel.WithdrawnOfferStatus(offer.Status)
	})
	after.Status = tender.StatusCanceled
	after.CancellationReason = req.Reason

	return &tendercancel.CancelPreview{Before: f.entity, After: after}, nil
}

func (f *fakeTenderManager) Cancel(
	_ context.Context,
	req *tendercancel.CancelTenderRequest,
) error {
	if err := f.guard.Write(); err != nil {
		return err
	}
	f.canceled = req

	return nil
}

func (f *fakeTenderManager) PreviewResponse(
	_ context.Context,
	req *tenderports.TenderResponseRequest,
) (*tendernegotiation.ResponsePreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	offer := f.entity.FindOffer(req.OfferID)
	plan := &tendernegotiation.ResponsePreview{Tender: f.entity, Offer: offer}
	if req.Action == tender.ResponseActionCounter {
		plan.TenderAfter = f.copyOffers(func(o *tender.TenderOffer) {
			if o.ID == req.OfferID {
				o.Status = tender.OfferStatusCountered
				o.CounterRate = decimal.NewNullDecimal(req.CounterRate)
				o.CounterNote = req.CounterNote
			}
		})

		return plan, nil
	}
	if req.Action == tender.ResponseActionAccept {
		plan.TenderAfter = f.copyOffers(func(o *tender.TenderOffer) {
			switch {
			case o.ID == req.OfferID:
				o.Status = tender.OfferStatusAccepted
				o.ResponseSource = req.Source
			case o.Status == tender.OfferStatusPending:
				o.Status = tender.OfferStatusSkipped
			}
		})
		plan.TenderAfter.Status = tender.StatusAccepted

		return plan, nil
	}

	plan.TenderAfter = f.copyOffers(func(o *tender.TenderOffer) {
		if o.ID == req.OfferID {
			o.Status = tender.OfferStatusDeclined
			o.ResponseSource = req.Source
			o.DeclineReason = req.DeclineReason
		}
	})
	for _, o := range plan.TenderAfter.Offers {
		if o.Status == tender.OfferStatusPending {
			plan.NextOffer = o

			break
		}
	}
	if plan.NextOffer == nil {
		plan.Exhausted = true
		plan.TenderAfter.Status = tender.StatusExhausted
	}

	return plan, nil
}

func (f *fakeTenderManager) RecordResponse(
	_ context.Context,
	req *tenderports.TenderResponseRequest,
) error {
	if err := f.guard.Write(); err != nil {
		return err
	}
	f.answered = req

	return nil
}

func (f *fakeTenderManager) PreviewBounce(
	_ context.Context,
	req *tenderbounce.BounceTenderRequest,
) (*tenderbounce.BouncePreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	revoked := f.entity.FindOffer(req.OfferID)
	after := f.copyOffers(func(o *tender.TenderOffer) {
		if o.ID == req.OfferID {
			o.Status = tender.OfferStatusRevoked
			o.RevokeReason = req.Reason
		}
	})
	after.Status = tender.StatusActive
	after.BounceCount++
	after.AcceptedOfferID = nil
	plan := &tenderbounce.BouncePreview{
		Before:    f.entity,
		After:     after,
		Revoked:   revoked,
		Reoffers:  f.entity.ReofferPlan(revoked),
		Unassigns: true,
	}
	for _, offer := range plan.Reoffers {
		offer.Carrier = f.carrierOf(offer.CarrierID)
	}
	plan.WillExhaust = len(plan.Reoffers) == 0

	return plan, nil
}

func (f *fakeTenderManager) carrierOf(carrierID pulid.ID) *carrier.Carrier {
	for _, offer := range f.entity.Offers {
		if offer.CarrierID == carrierID {
			return offer.Carrier
		}
	}
	return nil
}

func (f *fakeTenderManager) Bounce(
	_ context.Context,
	req *tenderbounce.BounceTenderRequest,
) (*tender.Tender, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.bounced = req

	return f.entity, nil
}

func TestCancelTender_SendsTheTenderAndTheReason(t *testing.T) {
	t.Parallel()

	tenders := newFakeTenderManager(tender.StatusActive, tender.OfferStatusSent)
	params := agenttoolstest.ExecuteParams(map[string]any{
		fieldTenderID:             tenders.entity.ID.String(),
		changepreview.FieldReason: " Covered by our own driver ",
	})

	require.NoError(t, NewCancelTenderTool(tenders).Execute(t.Context(), params))

	require.NotNil(t, tenders.canceled)
	assert.Equal(t, tenders.entity.ID, tenders.canceled.TenderID)
	assert.Equal(t, "Covered by our own driver", tenders.canceled.Reason)
	assert.Equal(t, params.Actor.UserID, tenders.canceled.TenantInfo.UserID)
}

// A tender waiting on review holds a carrier's acceptance; withdrawing it
// abandons that carrier, which is a person's call.
func TestCancelTender_ATenderAwaitingReviewIsAProposal(t *testing.T) {
	t.Parallel()

	limit := func(status tender.Status) agent.AutonomyTier {
		tenders := newFakeTenderManager(status, tender.OfferStatusSent)
		tool := NewCancelTenderTool(tenders)

		return tool.Policy().Condition.Limit(
			t.Context(),
			agenttoolstest.ExecuteParams(map[string]any{
				fieldTenderID:             tenders.entity.ID.String(),
				changepreview.FieldReason: "Not needed",
			}),
		)
	}

	assert.Equal(t, agent.TierActWithApproval, limit(tender.StatusActive))
	assert.Equal(t, agent.TierPropose, limit(tender.StatusNeedsReview))

	policy := NewCancelTenderTool(newFakeTenderManager(tender.StatusActive)).Policy()
	assert.Equal(t, permission.ResourceTender, policy.Resource)
	assert.Equal(t, permission.OpCancel, policy.Operation)
	assert.Equal(t, []agent.EgressClass{agent.EgressExternalRecipient}, policy.Egress)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
}

func TestCancelTender_PreviewShowsTheTenderAndEachOfferItMoves(t *testing.T) {
	t.Parallel()

	tenders := newFakeTenderManager(tender.StatusActive,
		tender.OfferStatusDeclined, tender.OfferStatusSent, tender.OfferStatusPending)
	tool := NewCancelTenderTool(tenders).(*CancelTenderTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&tenders.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				fieldTenderID:             tenders.entity.ID.String(),
				changepreview.FieldReason: "Covered by our own driver",
			}))
		},
	)

	require.Len(t, preview.Changes, 3, "the tender and the two offers it moves")
	record := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agentpreview.PreviewOperationArchive, record.Operation)
	assert.Equal(t, tenders.entity.ID, record.EntityID)
	assert.Equal(
		t,
		"Canceled",
		agenttoolstest.FieldByPath(t, record, changepreview.FieldStatus).After,
	)
	withdrawn := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, "Offer 2 to Blue Mesa", withdrawn.Label)
	assert.Equal(
		t,
		"Withdrawn",
		agenttoolstest.FieldByPath(t, withdrawn, changepreview.FieldStatus).After,
	)
	assert.Equal(
		t,
		"Skipped",
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 2),
			changepreview.FieldStatus,
		).After,
	)
	assert.Contains(t, preview.Summary, "1 offer carriers hold")
	assert.Contains(t, preview.Summary, "1 carrier not yet asked")
}

func TestRecordTenderResponse_RecordsAManualAnswer(t *testing.T) {
	t.Parallel()

	tenders := newFakeTenderManager(tender.StatusActive, tender.OfferStatusSent)
	offerID := tenders.entity.Offers[0].ID

	require.NoError(
		t,
		NewRecordTenderResponseTool(
			tenders,
			tenders,
		).Execute(t.Context(), agenttoolstest.ExecuteParams(
			map[string]any{
				fieldOfferID:       offerID.String(),
				fieldAction:        "Decline",
				fieldDeclineReason: " No truck near Reno ",
			},
		)),
	)

	require.NotNil(t, tenders.answered)
	assert.Equal(t, offerID, tenders.answered.OfferID)
	assert.Equal(t, tender.ResponseActionDecline, tenders.answered.Action)
	assert.Equal(t, tender.ResponseSourceManual, tenders.answered.Source)
	assert.Equal(t, "No truck near Reno", tenders.answered.DeclineReason)
}

func TestRecordTenderResponse_RefusesAnAnswerItCannotRecord(t *testing.T) {
	t.Parallel()

	tenders := newFakeTenderManager(tender.StatusActive, tender.OfferStatusSent)
	tool := NewRecordTenderResponseTool(tenders, tenders).(*RecordTenderResponseTool)
	offerID := tenders.entity.Offers[0].ID.String()

	for name, params := range map[string]map[string]any{
		"unknown action":       {fieldOfferID: offerID, fieldAction: "Maybe"},
		"reason on an accept":  {fieldOfferID: offerID, fieldAction: "Accept", fieldDeclineReason: "x"},
		"reason over the cap":  {fieldOfferID: offerID, fieldAction: "Decline", fieldDeclineReason: strings.Repeat("a", 501)},
		"no offer":             {fieldAction: "Accept"},
		"offer is not an id":   {fieldOfferID: "offer-2", fieldAction: "Accept"},
		"lowercase acceptance": {fieldOfferID: offerID, fieldAction: "accept"},
	} {
		require.Error(t, tool.Validate(t.Context(), agenttoolstest.ExecuteParams(params)), name)
	}
	assert.Nil(t, tenders.answered)
}

// An acceptance commits money and sends the carrier its rate confirmation;
// a decline sends the next carrier an offer. Each call is classed by what it
// does, and an acceptance is always a person's decision.
func TestRecordTenderResponse_IsClassedByTheAnswer(t *testing.T) {
	t.Parallel()

	manager := newFakeTenderManager(tender.StatusActive)
	policy := NewRecordTenderResponseTool(manager, manager).Policy()
	accept := agenttoolstest.ExecuteParams(map[string]any{fieldAction: "Accept"})
	decline := agenttoolstest.ExecuteParams(map[string]any{fieldAction: "Decline"})
	garbled := agenttoolstest.ExecuteParams(map[string]any{fieldAction: "yes"})

	assert.Equal(t, agent.EgressMoney, policy.Classified(accept).Egress)
	assert.Equal(t, agent.EgressExternalRecipient, policy.Classified(decline).Egress)
	assert.Equal(t, agent.EgressMoney, policy.Classified(garbled).Egress)
	assert.Equal(t, agent.TierPropose, policy.Condition.Limit(t.Context(), accept))
	assert.Equal(t, agent.TierActWithApproval, policy.Condition.Limit(t.Context(), decline))
	assert.Equal(t, permission.ResourceTender, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	assert.False(t, policy.Reversible)
}

func TestRecordTenderResponse_PreviewOfAnAcceptance(t *testing.T) {
	t.Parallel()

	tenders := newFakeTenderManager(tender.StatusActive,
		tender.OfferStatusSent, tender.OfferStatusPending)
	tool := NewRecordTenderResponseTool(tenders, tenders).(*RecordTenderResponseTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&tenders.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				fieldOfferID: tenders.entity.Offers[0].ID.String(),
				fieldAction:  "Accept",
			}))
		},
	)

	record := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(
		t,
		"Accepted",
		agenttoolstest.FieldByPath(t, record, changepreview.FieldStatus).After,
	)
	accepted := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, tenders.entity.Offers[0].ID, accepted.EntityID)
	assert.Equal(
		t,
		"Accepted",
		agenttoolstest.FieldByPath(t, accepted, changepreview.FieldStatus).After,
	)
	require.NotNil(t, accepted.Money)
	assert.True(t, decimal.NewFromInt(1500).Equal(accepted.Money.TotalAfter.Decimal))
	assert.Equal(
		t,
		"Skipped",
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 2),
			changepreview.FieldStatus,
		).After,
	)
	assert.True(t, preview.Partial, "the rate confirmation the acceptance issues is not shown")
	assert.Contains(t, preview.Summary, "Ridgeline accepted the offer")
	assert.Contains(t, preview.Summary, "rate confirmation")
}

func TestRecordTenderResponse_PreviewOfADeclineShowsTheNextOffer(t *testing.T) {
	t.Parallel()

	tenders := newFakeTenderManager(tender.StatusActive,
		tender.OfferStatusSent, tender.OfferStatusPending)
	tool := NewRecordTenderResponseTool(tenders, tenders).(*RecordTenderResponseTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&tenders.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				fieldOfferID:       tenders.entity.Offers[0].ID.String(),
				fieldAction:        "Decline",
				fieldDeclineReason: "No truck near Reno",
			}))
		},
	)

	declined := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(
		t,
		"Declined",
		agenttoolstest.FieldByPath(t, declined, changepreview.FieldStatus).After,
	)
	next := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, agentpreview.PreviewOperationSend, next.Operation)
	require.NotNil(t, next.Message)
	assert.Equal(t, []string{"dispatchb@carrier.test"}, next.Message.To)
	assert.Contains(t, preview.Summary, "offers the load to Blue Mesa")
	assert.False(t, preview.Partial)
}

func TestRecordTenderResponse_PreviewWarnsOfAnOfferNoLongerOpen(t *testing.T) {
	t.Parallel()

	tenders := newFakeTenderManager(tender.StatusActive, tender.OfferStatusExpired)
	tenders.refusal = tenderresponse.ErrOfferNoLongerAvailable
	tool := NewRecordTenderResponseTool(tenders, tenders).(*RecordTenderResponseTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&tenders.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				fieldOfferID: tenders.entity.Offers[0].ID.String(),
				fieldAction:  "Accept",
			}))
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}

func acceptedFake() *fakeTenderManager {
	tenders := newFakeTenderManager(tender.StatusAccepted,
		tender.OfferStatusAccepted, tender.OfferStatusSkipped, tender.OfferStatusSkipped)
	accepted := tenders.entity.Offers[0].ID
	tenders.entity.AcceptedOfferID = &accepted
	return tenders
}

func TestRecordTenderResponse_RecordsACounterWithItsRate(t *testing.T) {
	t.Parallel()

	tenders := newFakeTenderManager(tender.StatusActive, tender.OfferStatusSent)
	offerID := tenders.entity.Offers[0].ID

	require.NoError(
		t,
		NewRecordTenderResponseTool(
			tenders,
			tenders,
		).Execute(t.Context(), agenttoolstest.ExecuteParams(
			map[string]any{
				fieldOfferID:     offerID.String(),
				fieldAction:      "Counter",
				fieldCounterRate: " 1850.00 ",
				fieldCounterNote: " Fuel is up on this lane ",
			},
		)),
	)

	require.NotNil(t, tenders.answered)
	assert.Equal(t, tender.ResponseActionCounter, tenders.answered.Action)
	assert.True(t, decimal.NewFromInt(1850).Equal(tenders.answered.CounterRate))
	assert.Equal(t, "Fuel is up on this lane", tenders.answered.CounterNote)
	assert.Equal(t, tender.ResponseSourceManual, tenders.answered.Source)
	assert.Nil(t, tenders.bounced)
}

func TestRecordTenderResponse_RefusesACounterItCannotRecord(t *testing.T) {
	t.Parallel()

	tenders := newFakeTenderManager(tender.StatusActive, tender.OfferStatusSent)
	tool := NewRecordTenderResponseTool(tenders, tenders).(*RecordTenderResponseTool)
	offerID := tenders.entity.Offers[0].ID.String()

	for name, params := range map[string]map[string]any{
		"counter with no rate": {fieldOfferID: offerID, fieldAction: "Counter"},
		"counter rate not a number": {
			fieldOfferID: offerID, fieldAction: "Counter", fieldCounterRate: "about 1800",
		},
		"counter rate zero":      {fieldOfferID: offerID, fieldAction: "Counter", fieldCounterRate: "0"},
		"rate on an accept":      {fieldOfferID: offerID, fieldAction: "Accept", fieldCounterRate: "1800"},
		"note on a decline":      {fieldOfferID: offerID, fieldAction: "Decline", fieldCounterNote: "x"},
		"reason on a counter":    {fieldOfferID: offerID, fieldAction: "Counter", fieldCounterRate: "1800", fieldDeclineReason: "x"},
		"backed out with no why": {fieldOfferID: offerID, fieldAction: "BackedOut"},
		"backed out with a rate": {fieldOfferID: offerID, fieldAction: "BackedOut", fieldDeclineReason: "x", fieldCounterRate: "1800"},
	} {
		require.Error(t, tool.Validate(t.Context(), agenttoolstest.ExecuteParams(params)), name)
	}
	assert.Nil(t, tenders.answered)
	assert.Nil(t, tenders.bounced)
}

func TestRecordTenderResponse_ACarrierBackingOutBouncesTheTender(t *testing.T) {
	t.Parallel()

	tenders := acceptedFake()
	offerID := tenders.entity.Offers[0].ID

	require.NoError(
		t,
		NewRecordTenderResponseTool(
			tenders,
			tenders,
		).Execute(t.Context(), agenttoolstest.ExecuteParams(
			map[string]any{
				fieldOfferID:       offerID.String(),
				fieldAction:        "BackedOut",
				fieldDeclineReason: " Truck broke down in Joliet ",
			},
		)),
	)

	require.NotNil(t, tenders.bounced)
	assert.Equal(t, offerID, tenders.bounced.OfferID)
	assert.Equal(t, "Truck broke down in Joliet", tenders.bounced.Reason)
	assert.Equal(t, tender.ResponseSourceManual, tenders.bounced.Source)
	assert.Nil(t, tenders.answered)
}

func TestRecordTenderResponse_CountersAndBouncesArePeoplesDecisions(t *testing.T) {
	t.Parallel()

	manager := newFakeTenderManager(tender.StatusActive)
	policy := NewRecordTenderResponseTool(manager, manager).Policy()
	for _, answer := range []string{"Counter", "BackedOut"} {
		params := agenttoolstest.ExecuteParams(map[string]any{fieldAction: answer})
		assert.Equal(t, agent.EgressMoney, policy.Classified(params).Egress, answer)
		assert.Equal(t, agent.TierPropose, policy.Condition.Limit(t.Context(), params), answer)
	}
}

func TestRecordTenderResponse_PreviewOfACounter(t *testing.T) {
	t.Parallel()

	tenders := newFakeTenderManager(tender.StatusActive, tender.OfferStatusSent)
	tool := NewRecordTenderResponseTool(tenders, tenders).(*RecordTenderResponseTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&tenders.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				fieldOfferID:     tenders.entity.Offers[0].ID.String(),
				fieldAction:      "Counter",
				fieldCounterRate: "1850",
			}))
		},
	)

	countered := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(
		t,
		"Countered",
		agenttoolstest.FieldByPath(t, countered, changepreview.FieldStatus).After,
	)
	assert.Contains(t, preview.Summary, "Ridgeline countered at 1850.00 USD flat")
	assert.Contains(t, preview.Summary, "against 1500.00 USD flat offered")
}

func TestRecordTenderResponse_PreviewOfTakingACounterPaysTheCounterRate(t *testing.T) {
	t.Parallel()

	tenders := newFakeTenderManager(tender.StatusActive,
		tender.OfferStatusCountered, tender.OfferStatusPending)
	tenders.entity.Offers[0].CounterRate = decimal.NewNullDecimal(decimal.NewFromInt(1900))
	tenders.entity.Offers[0].CounterNeedsApproval = true
	tool := NewRecordTenderResponseTool(tenders, tenders).(*RecordTenderResponseTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&tenders.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				fieldOfferID: tenders.entity.Offers[0].ID.String(),
				fieldAction:  "Accept",
			}))
		},
	)

	accepted := agenttoolstest.PreviewChange(t, preview, 1)
	require.NotNil(t, accepted.Money)
	assert.True(t, decimal.NewFromInt(1900).Equal(accepted.Money.TotalAfter.Decimal))
	assert.Contains(t, preview.Summary, "Would take Ridgeline's counter at 1900.00 USD flat")
	assert.Contains(t, preview.Summary, "below the margin floor")
}

func TestRecordTenderResponse_PreviewOfACarrierBackingOut(t *testing.T) {
	t.Parallel()

	tenders := acceptedFake()
	tool := NewRecordTenderResponseTool(tenders, tenders).(*RecordTenderResponseTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&tenders.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				fieldOfferID:       tenders.entity.Offers[0].ID.String(),
				fieldAction:        "BackedOut",
				fieldDeclineReason: "Truck broke down",
			}))
		},
	)

	record := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(
		t,
		"Active",
		agenttoolstest.FieldByPath(t, record, changepreview.FieldStatus).After,
	)
	revoked := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(
		t,
		"Revoked",
		agenttoolstest.FieldByPath(t, revoked, changepreview.FieldStatus).After,
	)
	next := agenttoolstest.PreviewChange(t, preview, 2)
	assert.Equal(t, agentpreview.PreviewOperationSend, next.Operation)
	assert.Equal(t, []string{"dispatchb@carrier.test"}, next.Message.To)
	assert.Contains(t, preview.Summary, "Ridgeline backed out")
	assert.Contains(t, preview.Summary, "rate confirmation is voided")
	assert.Contains(t, preview.Summary, "starting with Blue Mesa")
	assert.True(t, preview.Partial)
}
