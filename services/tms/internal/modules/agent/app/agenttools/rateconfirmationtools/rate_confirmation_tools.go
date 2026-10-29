package rateconfirmationtools

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carrierpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateconfirmationservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
)

const fieldRateConfirmationID = "rateConfirmationId"

// rateConfirmations is the slice of the rate confirmation service the tools
// use, each write with the preview that plans it.
type rateConfirmations interface {
	Generate(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveID pulid.ID,
		actor *permissionports.RequestActor,
	) (*rateconfirmation.RateConfirmation, error)
	PreviewGenerate(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveID pulid.ID,
		actor *permissionports.RequestActor,
	) (*rateconfirmationservice.GeneratePreview, error)
	Send(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		rateConfirmationID pulid.ID,
	) (*rateconfirmation.RateConfirmation, error)
	PreviewSend(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		rateConfirmationID pulid.ID,
	) (*rateconfirmationservice.SendPreview, error)
	Void(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		rateConfirmationID pulid.ID,
		reason string,
	) (*rateconfirmation.RateConfirmation, error)
	PreviewVoid(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		rateConfirmationID pulid.ID,
		reason string,
	) (*rateconfirmationservice.ChangePreview, error)
	MarkConfirmed(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		rateConfirmationID pulid.ID,
		confirmedByName string,
	) (*rateconfirmation.RateConfirmation, error)
	PreviewMarkConfirmed(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		rateConfirmationID pulid.ID,
		confirmedByName string,
	) (*rateconfirmationservice.ChangePreview, error)
}

// generateRateConfirmationTool renders a new revision of the agreement for
// the move's carrier and files it on the shipment, which is the console's
// generate. Nothing is sent.
type GenerateRateConfirmationTool struct {
	rateCons rateConfirmations
}

func NewGenerateRateConfirmationTool(rateCons rateConfirmations) agenttoolports.AgentTool {
	return &GenerateRateConfirmationTool{rateCons: rateCons}
}

func (t *GenerateRateConfirmationTool) Name() string { return "rate_generate_confirmation" }

func (t *GenerateRateConfirmationTool) SearchTerms() []string {
	return []string{"rate con", "create rate con", "carrier confirmation"}
}

func (t *GenerateRateConfirmationTool) Description() string {
	return "Generate the rate confirmation for a move covered by an outside carrier. It is " +
		"a new revision of the agreement, with the carrier's pay and the stops as they stand, " +
		"rendered as a PDF and filed on the shipment. Generate after dispatch_create_carrier_assignment, " +
		"or again after the rate or the stops change; a new revision voids the one before " +
		"it. Nothing is sent: find the new revision with rate_list_confirmations and send " +
		"it with rate_send_confirmation. A tender acceptance issues its own."
}

func (t *GenerateRateConfirmationTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		changepreview.PreviewFieldShipmentMoveID: agenttoolschema.RecordIDText(
			permission.ResourceShipmentMove,
			"The carrier-covered move, from dispatch_get_board (moveId) or shipment_get "+
				"(its moves).",
		),
	}, changepreview.PreviewFieldShipmentMoveID)
}

func (t *GenerateRateConfirmationTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceRateConfirmation,
		Operation:     permission.OpCreate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Condition: &agenttoolports.TierCondition{
			Description: "A first revision is generated as far as the agent allows; one " +
				"that would void a revision already standing, which the carrier may hold " +
				"or have signed, is a proposal a person decides, as is one the service " +
				"would refuse or that cannot be read.",
			Limit: t.tierLimit,
		},
		Rationale: "Renders the agreement and files it on the shipment inside Trenova; " +
			"nothing reaches the carrier until it is sent, and voiding the revision undoes it.",
	}
}

func (t *GenerateRateConfirmationTool) tierLimit(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) agent.AutonomyTier {
	moveID, err := t.moveID(params)
	if err != nil {
		return agent.TierPropose
	}

	plan, err := t.rateCons.PreviewGenerate(
		ctx,
		toolparams.TenantFrom(params),
		moveID,
		params.Actor,
	)
	if err != nil || plan.SupersededBefore != nil {
		return agent.TierPropose
	}

	return agent.TierAutoExecute
}

func (t *GenerateRateConfirmationTool) Validate(
	_ context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	_, err := t.moveID(params)

	return err
}

func (t *GenerateRateConfirmationTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	moveID, err := t.moveID(params)
	if err != nil {
		return err
	}

	_, err = t.rateCons.Generate(ctx, toolparams.TenantFrom(params), moveID, params.Actor)

	return err
}

func (t *GenerateRateConfirmationTool) moveID(
	params *agenttoolports.ToolExecuteParams,
) (pulid.ID, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return pulid.Nil, err
	}

	return toolparams.RequirePulid(params.Params, changepreview.PreviewFieldShipmentMoveID)
}

func (t *GenerateRateConfirmationTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(
		params,
		changepreview.PreviewFieldShipmentMoveID,
		permission.ResourceShipmentMove,
	)
}

// sendRateConfirmationTool emails a revision to the carrier's rate
// confirmation contacts, which is the console's send. The recipients are the
// carrier's own, read from its record; the tool takes none from the model.
type SendRateConfirmationTool struct {
	rateCons rateConfirmations
}

func NewSendRateConfirmationTool(rateCons rateConfirmations) agenttoolports.AgentTool {
	return &SendRateConfirmationTool{rateCons: rateCons}
}

func (t *SendRateConfirmationTool) Name() string { return "rate_send_confirmation" }

func (t *SendRateConfirmationTool) Recipe() []string {
	return []string{"rate_list_confirmations", "rate_send_confirmation"}
}

func (t *SendRateConfirmationTool) Description() string {
	return "Email a rate confirmation to the carrier, with a link to sign it while it is " +
		"unsigned. The PDF goes to the carrier's contacts that receive rate confirmations, " +
		"or the carrier's own address. The recipients come from the carrier's record and " +
		"cannot be chosen here; fix the carrier's contacts first if they are wrong. Send a " +
		"revision that is Generated, Sent (to send it again) or Confirmed (the executed " +
		"copy); find it with rate_list_confirmations."
}

func (t *SendRateConfirmationTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		fieldRateConfirmationID: agenttoolschema.RecordIDText(
			permission.ResourceRateConfirmation,
			"The revision to send, from rate_list_confirmations.",
		),
	}, fieldRateConfirmationID)
}

func (t *SendRateConfirmationTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceRateConfirmation,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Emails a binding agreement, with a link to sign it, to a carrier " +
			"outside the organization; the recipients come from the carrier's record, and " +
			"a sent email cannot be recalled, so a person decides every send.",
	}
}

func (t *SendRateConfirmationTool) Validate(
	_ context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	_, err := rateConfirmationArg(t, params)

	return err
}

func (t *SendRateConfirmationTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	id, err := rateConfirmationArg(t, params)
	if err != nil {
		return err
	}

	_, err = t.rateCons.Send(ctx, toolparams.TenantFrom(params), id)

	return err
}

func (t *SendRateConfirmationTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, fieldRateConfirmationID, permission.ResourceRateConfirmation)
}

// voidRateConfirmationTool withdraws a revision, which is the console's void.
type VoidRateConfirmationTool struct {
	rateCons rateConfirmations
}

func NewVoidRateConfirmationTool(rateCons rateConfirmations) agenttoolports.AgentTool {
	return &VoidRateConfirmationTool{rateCons: rateCons}
}

func (t *VoidRateConfirmationTool) Name() string { return "rate_void_confirmation" }

func (t *VoidRateConfirmationTool) Description() string {
	return "Void a rate confirmation so the revision stops standing and its sign link stops " +
		"working. Voiding one the carrier confirmed returns their assignment to " +
		"awaiting confirmation. Use it when the agreement as written is wrong and should " +
		"not be signed; to correct it, generate a new revision instead, which voids this " +
		"one. Say why in reason. Find the revision with rate_list_confirmations."
}

func (t *VoidRateConfirmationTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		fieldRateConfirmationID: agenttoolschema.RecordIDText(
			permission.ResourceRateConfirmation,
			"The revision to void, from rate_list_confirmations.",
		),
		changepreview.FieldReason: jsonschemautils.Text(
			"Why the agreement is withdrawn, in a sentence a person can check.",
		),
	}, fieldRateConfirmationID, changepreview.FieldReason)
}

func (t *VoidRateConfirmationTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceRateConfirmation,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Condition: &agenttoolports.TierCondition{
			Description: "A revision the carrier has been sent or has signed is voided only " +
				"as a proposal a person decides, as is one that cannot be read; one never " +
				"sent is voided once a person approves it.",
			Limit: t.tierLimit,
		},
		Rationale: "Withdraws the organization's written agreement to pay a carrier, whose " +
			"sign link stops working, and undoes a confirmation it carried; a voided " +
			"revision cannot be restored, only replaced.",
	}
}

func (t *VoidRateConfirmationTool) tierLimit(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) agent.AutonomyTier {
	id, reason, err := t.arguments(params)
	if err != nil {
		return agent.TierPropose
	}

	plan, err := t.rateCons.PreviewVoid(ctx, toolparams.TenantFrom(params), id, reason)
	if err != nil || plan.Before.Status != rateconfirmation.StatusGenerated {
		return agent.TierPropose
	}

	return agent.TierActWithApproval
}

func (t *VoidRateConfirmationTool) Validate(
	_ context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	_, _, err := t.arguments(params)

	return err
}

func (t *VoidRateConfirmationTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	id, reason, err := t.arguments(params)
	if err != nil {
		return err
	}

	_, err = t.rateCons.Void(ctx, toolparams.TenantFrom(params), id, reason)

	return err
}

func (t *VoidRateConfirmationTool) arguments(
	params *agenttoolports.ToolExecuteParams,
) (pulid.ID, string, error) {
	id, err := rateConfirmationArg(t, params)
	if err != nil {
		return pulid.Nil, "", err
	}
	reason, err := toolparams.RequireString(params.Params, changepreview.FieldReason)
	if err != nil {
		return pulid.Nil, "", err
	}

	return id, strings.TrimSpace(reason), nil
}

func (t *VoidRateConfirmationTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, fieldRateConfirmationID, permission.ResourceRateConfirmation)
}

// recordRateConfirmationConfirmedTool records that the carrier confirmed a
// revision outside its sign link, which is the console's mark confirmed.
type RecordRateConfirmationConfirmedTool struct {
	rateCons rateConfirmations
}

func NewRecordRateConfirmationConfirmedTool(rateCons rateConfirmations) agenttoolports.AgentTool {
	return &RecordRateConfirmationConfirmedTool{rateCons: rateCons}
}

func (t *RecordRateConfirmationConfirmedTool) Name() string {
	return "rate_record_confirmation"
}

func (t *RecordRateConfirmationConfirmedTool) SearchTerms() []string {
	return []string{"signed rate con", "faxed", "carrier signed"}
}

func (t *RecordRateConfirmationConfirmedTool) Description() string {
	return "Record that the carrier confirmed a rate confirmation outside its sign link, " +
		"for instance by signing and returning the PDF or agreeing on the phone. The " +
		"revision becomes the executed agreement and the carrier's assignment is " +
		"confirmed. Record only a confirmation you were given, naming the person at the " +
		"carrier who gave it; a revision already confirmed or voided cannot be confirmed. " +
		"A carrier signing through the link is recorded on its own."
}

func (t *RecordRateConfirmationConfirmedTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		fieldRateConfirmationID: agenttoolschema.RecordIDText(
			permission.ResourceRateConfirmation,
			"The revision the carrier confirmed, from rate_list_confirmations.",
		),
		carrierpreview.FieldConfirmedByName: jsonschemautils.Text(
			"The name of the person at the carrier who confirmed it.",
		),
	}, fieldRateConfirmationID, carrierpreview.FieldConfirmedByName)
}

func (t *RecordRateConfirmationConfirmedTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceRateConfirmation,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Makes the revision the executed agreement to pay the carrier and " +
			"confirms their assignment, on the word of someone outside the organization; " +
			"voiding it afterwards withdraws the agreement rather than restoring it.",
	}
}

func (t *RecordRateConfirmationConfirmedTool) Validate(
	_ context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	_, _, err := t.arguments(params)

	return err
}

func (t *RecordRateConfirmationConfirmedTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	id, name, err := t.arguments(params)
	if err != nil {
		return err
	}

	_, err = t.rateCons.MarkConfirmed(ctx, toolparams.TenantFrom(params), id, name)

	return err
}

func (t *RecordRateConfirmationConfirmedTool) arguments(
	params *agenttoolports.ToolExecuteParams,
) (pulid.ID, string, error) {
	id, err := rateConfirmationArg(t, params)
	if err != nil {
		return pulid.Nil, "", err
	}
	name, err := toolparams.RequireString(params.Params, carrierpreview.FieldConfirmedByName)
	if err != nil {
		return pulid.Nil, "", err
	}

	return id, strings.TrimSpace(name), nil
}

func (t *RecordRateConfirmationConfirmedTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, fieldRateConfirmationID, permission.ResourceRateConfirmation)
}

func rateConfirmationArg(
	tool agenttoolports.AgentTool,
	params *agenttoolports.ToolExecuteParams,
) (pulid.ID, error) {
	if err := toolguard.GuardPreview(tool, params); err != nil {
		return pulid.Nil, err
	}

	return toolparams.RequirePulid(params.Params, fieldRateConfirmationID)
}
