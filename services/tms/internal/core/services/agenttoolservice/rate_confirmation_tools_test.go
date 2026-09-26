package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/rateconfirmationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRateCons plans from one revision it holds and records each write.
type fakeRateCons struct {
	guard      writeGuard
	entity     *rateconfirmation.RateConfirmation
	assignment *shipment.CarrierAssignment
	standing   *rateconfirmation.RateConfirmation
	refusal    error

	tenant    pagination.TenantInfo
	actor     *serviceports.RequestActor
	generated pulid.ID
	sent      pulid.ID
	voided    string
	confirmed string
}

func newFakeRateCons(status rateconfirmation.Status) *fakeRateCons {
	carrierEntity := &carrier.Carrier{ID: pulid.MustNew("car_"), Name: "Eastline Transport"}
	assignment := &shipment.CarrierAssignment{
		ID:            pulid.MustNew("casn_"),
		CarrierID:     carrierEntity.ID,
		Status:        shipment.CarrierAssignmentStatusPending,
		RateMethod:    shipment.CarrierRateMethodFlat,
		BaseAmount:    decimal.NewFromInt(1850),
		FuelSurcharge: decimal.NewFromInt(150),
		CurrencyCode:  "USD",
	}
	if status == rateconfirmation.StatusConfirmed {
		assignment.Status = shipment.CarrierAssignmentStatusConfirmed
	}

	return &fakeRateCons{
		assignment: assignment,
		entity: &rateconfirmation.RateConfirmation{
			ID:                  pulid.MustNew("rc_"),
			CarrierAssignmentID: assignment.ID,
			CarrierID:           carrierEntity.ID,
			Carrier:             carrierEntity,
			Revision:            1,
			Status:              status,
			Version:             3,
		},
	}
}

func (f *fakeRateCons) PreviewGenerate(
	_ context.Context,
	_ pagination.TenantInfo,
	moveID pulid.ID,
	_ *serviceports.RequestActor,
) (*rateconfirmationservice.GeneratePreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	plan := &rateconfirmationservice.GeneratePreview{
		Created: &rateconfirmation.RateConfirmation{
			CarrierAssignmentID: f.assignment.ID,
			CarrierID:           f.assignment.CarrierID,
			ShipmentMoveID:      moveID,
			Revision:            2,
			Status:              rateconfirmation.StatusGenerated,
		},
		Assignment: f.assignment,
		Carrier:    f.entity.Carrier,
		Shipment:   &shipment.Shipment{ProNumber: "S-4001"},
	}
	if f.standing != nil {
		voided := *f.standing
		voided.Void(1790000000, "Superseded by a new revision")
		plan.SupersededBefore = f.standing
		plan.SupersededAfter = &voided
	}

	return plan, nil
}

func (f *fakeRateCons) Generate(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	actor *serviceports.RequestActor,
) (*rateconfirmation.RateConfirmation, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.tenant, f.actor, f.generated = tenantInfo, actor, moveID

	return f.entity, nil
}

func (f *fakeRateCons) PreviewSend(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (*rateconfirmationservice.SendPreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	after := *f.entity
	after.Status = rateconfirmation.StatusSent
	after.SentToEmails = "dispatch@eastline.test"

	return &rateconfirmationservice.SendPreview{
		Before:     f.entity,
		After:      &after,
		Recipients: []string{"dispatch@eastline.test"},
		Subject:    "Rate confirmation S-4001",
		Body:       "Please sign the attached rate confirmation.",
		Attachment: "rate-confirmation-rev1.pdf",
		SignLink:   true,
	}, nil
}

func (f *fakeRateCons) Send(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*rateconfirmation.RateConfirmation, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.tenant, f.sent = tenantInfo, id

	return f.entity, nil
}

func (f *fakeRateCons) change(
	mutate func(*rateconfirmation.RateConfirmation),
	moveAssignment func(*shipment.CarrierAssignment) bool,
) (*rateconfirmationservice.ChangePreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	after := *f.entity
	mutate(&after)
	plan := &rateconfirmationservice.ChangePreview{Before: f.entity, After: &after}
	assignment := *f.assignment
	if moveAssignment(&assignment) {
		plan.AssignmentBefore = f.assignment
		plan.AssignmentAfter = &assignment
	}

	return plan, nil
}

func (f *fakeRateCons) PreviewVoid(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	reason string,
) (*rateconfirmationservice.ChangePreview, error) {
	return f.change(func(rc *rateconfirmation.RateConfirmation) {
		rc.Void(1790000000, reason)
	}, (*shipment.CarrierAssignment).RevertConfirmation)
}

func (f *fakeRateCons) Void(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	reason string,
) (*rateconfirmation.RateConfirmation, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.voided = reason

	return f.entity, nil
}

func (f *fakeRateCons) PreviewMarkConfirmed(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	name string,
) (*rateconfirmationservice.ChangePreview, error) {
	return f.change(func(rc *rateconfirmation.RateConfirmation) {
		rc.Confirm(rateconfirmation.Confirmation{
			At:   1790000000,
			Name: name,
			Via:  rateconfirmation.ViaDispatcher,
		})
	}, func(assignment *shipment.CarrierAssignment) bool {
		return assignment.Confirm(1790000000)
	})
}

func (f *fakeRateCons) MarkConfirmed(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	name string,
) (*rateconfirmation.RateConfirmation, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.confirmed = name

	return f.entity, nil
}

func TestGenerateRateConfirmation_GeneratesForTheMoveAsTheActor(t *testing.T) {
	t.Parallel()

	rateCons := newFakeRateCons(rateconfirmation.StatusGenerated)
	moveID := pulid.MustNew("smv_")
	params := executeParams(map[string]any{previewFieldShipmentMoveID: moveID.String()})

	require.NoError(t, newGenerateRateConfirmationTool(rateCons).Execute(t.Context(), params))

	assert.Equal(t, moveID, rateCons.generated)
	assert.Equal(t, params.OrganizationID, rateCons.tenant.OrgID)
	assert.Equal(t, params.Actor, rateCons.actor)
}

// A first revision stays inside Trenova until it is sent, so it may run on
// the agent's own say; one that would void a revision the carrier may be
// holding, or have signed, is a person's call.
func TestGenerateRateConfirmation_SupersedingARevisionIsAProposal(t *testing.T) {
	t.Parallel()

	params := executeParams(map[string]any{
		previewFieldShipmentMoveID: pulid.MustNew("smv_").String(),
	})

	first := newFakeRateCons(rateconfirmation.StatusGenerated)
	policy := newGenerateRateConfirmationTool(first).Policy()
	assert.Equal(t, agent.TierAutoExecute, policy.Condition.Limit(t.Context(), params))

	again := newFakeRateCons(rateconfirmation.StatusGenerated)
	again.standing = &rateconfirmation.RateConfirmation{
		ID:       pulid.MustNew("rc_"),
		Revision: 1,
		Status:   rateconfirmation.StatusConfirmed,
	}
	policy = newGenerateRateConfirmationTool(again).Policy()
	assert.Equal(t, agent.TierPropose, policy.Condition.Limit(t.Context(), params))

	assert.Equal(t, permission.ResourceRateConfirmation, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
}

func TestGenerateRateConfirmation_PreviewShowsTheRevisionAndWhatItVoids(t *testing.T) {
	t.Parallel()

	rateCons := newFakeRateCons(rateconfirmation.StatusGenerated)
	rateCons.standing = &rateconfirmation.RateConfirmation{
		ID:       pulid.MustNew("rc_"),
		Revision: 1,
		Status:   rateconfirmation.StatusSent,
		Version:  4,
	}
	tool := newGenerateRateConfirmationTool(rateCons).(*generateRateConfirmationTool)

	preview := previewWithoutWrites(t, &rateCons.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{
			previewFieldShipmentMoveID: pulid.MustNew("smv_").String(),
		}))
	})

	created := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)
	assert.Equal(t, "Rate confirmation revision 2 for S-4001", created.Label)
	require.NotNil(t, created.Money)
	assert.True(t, decimal.NewFromInt(2000).Equal(created.Money.TotalAfter.Decimal))
	voided := previewChange(t, preview, 1)
	assert.Equal(t, rateCons.standing.ID, voided.EntityID)
	assert.Equal(t, "Voided", fieldByPath(t, voided, fieldStatus).After)
	assert.Contains(t, preview.Summary, "Nothing is sent to the carrier")
	assert.Contains(t, preview.Summary, "Revision 1, which is with the carrier to sign")
}

func TestSendRateConfirmation_NeverRunsWithoutAPersonAndTakesNoRecipients(t *testing.T) {
	t.Parallel()

	tool := newSendRateConfirmationTool(newFakeRateCons(rateconfirmation.StatusGenerated))
	policy := tool.Policy()

	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressExternalRecipient}, policy.Egress)
	assert.Equal(t, permission.ResourceRateConfirmation, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	assert.False(t, policy.Reversible)

	properties, ok := tool.ParamSchema()["properties"].(map[string]any)
	require.True(t, ok)
	assert.Len(t, properties, 1, "the recipients come from the carrier's record, never the model")
}

func TestSendRateConfirmation_SendsTheRevisionNamed(t *testing.T) {
	t.Parallel()

	rateCons := newFakeRateCons(rateconfirmation.StatusGenerated)

	require.NoError(t, newSendRateConfirmationTool(rateCons).Execute(t.Context(), executeParams(
		map[string]any{fieldRateConfirmationID: rateCons.entity.ID.String()},
	)))

	assert.Equal(t, rateCons.entity.ID, rateCons.sent)
}

func TestSendRateConfirmation_PreviewIsTheMessageWithItsRecipientsAndAttachment(t *testing.T) {
	t.Parallel()

	rateCons := newFakeRateCons(rateconfirmation.StatusGenerated)
	tool := newSendRateConfirmationTool(rateCons).(*sendRateConfirmationTool)

	preview := previewWithoutWrites(t, &rateCons.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{
			fieldRateConfirmationID: rateCons.entity.ID.String(),
		}))
	})

	sent := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationSend, sent.Operation)
	require.NotNil(t, sent.Message)
	assert.Equal(t, agent.MessageChannelEmail, sent.Message.Channel)
	assert.Equal(t, []string{"dispatch@eastline.test"}, sent.Message.To)
	assert.Equal(t, "Rate confirmation S-4001", sent.Message.Subject)
	assert.Equal(t, []string{"rate-confirmation-rev1.pdf"}, sent.Message.Attachments)
	record := previewChange(t, preview, 1)
	assert.Equal(t, rateCons.entity.ID, record.EntityID)
	assert.Equal(t, "Sent", fieldByPath(t, record, fieldStatus).After)
	assert.Contains(t, preview.Summary, "Eastline Transport at dispatch@eastline.test")
	assert.Contains(t, preview.Summary, "link to sign it")
}

func TestSendRateConfirmation_PreviewWarnsOfARevisionItCannotSend(t *testing.T) {
	t.Parallel()

	rateCons := newFakeRateCons(rateconfirmation.StatusVoided)
	rateCons.refusal = errortypes.NewBusinessError("A Voided rate confirmation cannot be sent")
	tool := newSendRateConfirmationTool(rateCons).(*sendRateConfirmationTool)

	preview := previewWithoutWrites(t, &rateCons.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{
			fieldRateConfirmationID: rateCons.entity.ID.String(),
		}))
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestVoidRateConfirmation_VoidsWithTheReason(t *testing.T) {
	t.Parallel()

	rateCons := newFakeRateCons(rateconfirmation.StatusSent)
	tool := newVoidRateConfirmationTool(rateCons)

	require.Error(t, tool.(*voidRateConfirmationTool).Validate(t.Context(), executeParams(
		map[string]any{fieldRateConfirmationID: rateCons.entity.ID.String()},
	)), "a void needs a reason")
	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		fieldRateConfirmationID: rateCons.entity.ID.String(),
		fieldReason:             " Rate renegotiated ",
	})))

	assert.Equal(t, "Rate renegotiated", rateCons.voided)
}

// A revision never sent is internal paperwork; one the carrier holds or has
// signed is an agreement, and withdrawing it is a person's decision.
func TestVoidRateConfirmation_AnAgreementTheCarrierHoldsIsAProposal(t *testing.T) {
	t.Parallel()

	limit := func(status rateconfirmation.Status) agent.AutonomyTier {
		rateCons := newFakeRateCons(status)

		return newVoidRateConfirmationTool(rateCons).Policy().Condition.Limit(
			t.Context(),
			executeParams(map[string]any{
				fieldRateConfirmationID: rateCons.entity.ID.String(),
				fieldReason:             "Wrong rate",
			}),
		)
	}

	assert.Equal(t, agent.TierActWithApproval, limit(rateconfirmation.StatusGenerated))
	assert.Equal(t, agent.TierPropose, limit(rateconfirmation.StatusSent))
	assert.Equal(t, agent.TierPropose, limit(rateconfirmation.StatusConfirmed))
}

func TestVoidRateConfirmation_PreviewShowsTheAssignmentGoingBackToPending(t *testing.T) {
	t.Parallel()

	rateCons := newFakeRateCons(rateconfirmation.StatusConfirmed)
	tool := newVoidRateConfirmationTool(rateCons).(*voidRateConfirmationTool)

	preview := previewWithoutWrites(t, &rateCons.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{
			fieldRateConfirmationID: rateCons.entity.ID.String(),
			fieldReason:             "Wrong rate",
		}))
	})

	record := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationArchive, record.Operation)
	assert.Equal(t, "Voided", fieldByPath(t, record, fieldStatus).After)
	assert.Equal(t, "Wrong rate", fieldByPath(t, record, "voidReason").After)
	assignment := previewChange(t, preview, 1)
	assert.Equal(t, "Pending", fieldByPath(t, assignment, fieldStatus).After)
	assert.Contains(t, preview.Summary, "signed by the carrier")
	assert.Contains(t, preview.Summary, "back to awaiting confirmation")
}

func TestRecordRateConfirmationConfirmed_RecordsWhoConfirmed(t *testing.T) {
	t.Parallel()

	rateCons := newFakeRateCons(rateconfirmation.StatusSent)
	tool := newRecordRateConfirmationConfirmedTool(rateCons)

	require.Error(t, tool.(*recordRateConfirmationConfirmedTool).Validate(t.Context(),
		executeParams(map[string]any{fieldRateConfirmationID: rateCons.entity.ID.String()}),
	), "a confirmation names who gave it")
	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		fieldRateConfirmationID: rateCons.entity.ID.String(),
		fieldConfirmedByName:    " Dana Ruiz ",
	})))

	assert.Equal(t, "Dana Ruiz", rateCons.confirmed)

	policy := tool.Policy()
	assert.Equal(t, permission.OpUpdate, policy.Operation, "recording is not approving")
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
}

func TestRecordRateConfirmationConfirmed_PreviewShowsTheAgreementExecuted(t *testing.T) {
	t.Parallel()

	rateCons := newFakeRateCons(rateconfirmation.StatusSent)
	tool := newRecordRateConfirmationConfirmedTool(rateCons).(*recordRateConfirmationConfirmedTool)

	preview := previewWithoutWrites(t, &rateCons.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{
			fieldRateConfirmationID: rateCons.entity.ID.String(),
			fieldConfirmedByName:    "Dana Ruiz",
		}))
	})

	record := previewChange(t, preview, 0)
	assert.Equal(t, "Confirmed", fieldByPath(t, record, fieldStatus).After)
	assert.Equal(t, "Dana Ruiz", fieldByPath(t, record, fieldConfirmedByName).After)
	assert.Equal(t, "Confirmed by", fieldByPath(t, record, fieldConfirmedByName).Label)
	assignment := previewChange(t, preview, 1)
	assert.Equal(t, "Confirmed", fieldByPath(t, assignment, fieldStatus).After)
	assert.Contains(t, preview.Summary, "Dana Ruiz confirmed revision 1 for Eastline Transport")
}
