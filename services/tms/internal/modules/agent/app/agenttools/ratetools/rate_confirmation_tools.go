package ratetools

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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

func NewGenerateRateConfirmationTool(rateCons rateConfirmations) agentports.AgentTool {
	return &GenerateRateConfirmationTool{rateCons: rateCons}
}

func (t *GenerateRateConfirmationTool) Name() string { return "generate_rate_confirmation" }

func (t *GenerateRateConfirmationTool) SearchTerms() []string {
	return []string{"rate con", "create rate con", "carrier confirmation"}
}

func (t *GenerateRateConfirmationTool) Description() string {
	return "Generate the rate confirmation for a move covered by an outside carrier. It is " +
		"a new revision of the agreement, with the carrier's pay and the stops as they stand, " +
		"rendered as a PDF and filed on the shipment. Generate after assign_move_to_carrier, " +
		"or again after the rate or the stops change; a new revision voids the one before " +
		"it. Nothing is sent: find the new revision with list_rate_confirmations and send " +
		"it with send_rate_confirmation. A tender acceptance issues its own."
}

func (t *GenerateRateConfirmationTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		common.PreviewFieldShipmentMoveID: agenttoolschema.RecordIDText(
			permission.ResourceShipmentMove,
			"The carrier-covered move, from get_dispatch_board (moveId) or get_shipment "+
				"(its moves).",
		),
	}, common.PreviewFieldShipmentMoveID)
}

func (t *GenerateRateConfirmationTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceRateConfirmation,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Condition: &agentports.TierCondition{
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
	params agentports.ToolExecuteParams, //nolint:gocritic // a TierCondition passes params by value
) agent.AutonomyTier {
	moveID, err := t.moveID(&params)
	if err != nil {
		return agent.TierPropose
	}

	plan, err := t.rateCons.PreviewGenerate(ctx, common.TenantFrom(params), moveID, params.Actor)
	if err != nil || plan.SupersededBefore != nil {
		return agent.TierPropose
	}

	return agent.TierAutoExecute
}

func (t *GenerateRateConfirmationTool) Validate(
	_ context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, err := t.moveID(&params)

	return err
}

func (t *GenerateRateConfirmationTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	moveID, err := t.moveID(&params)
	if err != nil {
		return err
	}

	_, err = t.rateCons.Generate(ctx, common.TenantFrom(params), moveID, params.Actor)

	return err
}

func (t *GenerateRateConfirmationTool) moveID(
	params *agentports.ToolExecuteParams,
) (pulid.ID, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return pulid.Nil, err
	}

	return common.RequirePulid(params.Params, common.PreviewFieldShipmentMoveID)
}

func (t *GenerateRateConfirmationTool) Target(
	params map[string]any,
) (agentports.ToolTarget, bool) {
	return common.TargetOf(
		params,
		common.PreviewFieldShipmentMoveID,
		permission.ResourceShipmentMove,
	)
}

// sendRateConfirmationTool emails a revision to the carrier's rate
// confirmation contacts, which is the console's send. The recipients are the
// carrier's own, read from its record; the tool takes none from the model.
type SendRateConfirmationTool struct {
	rateCons rateConfirmations
}

func NewSendRateConfirmationTool(rateCons rateConfirmations) agentports.AgentTool {
	return &SendRateConfirmationTool{rateCons: rateCons}
}

func (t *SendRateConfirmationTool) Name() string { return "send_rate_confirmation" }

func (t *SendRateConfirmationTool) Recipe() []string {
	return []string{"list_rate_confirmations", "send_rate_confirmation"}
}

func (t *SendRateConfirmationTool) Description() string {
	return "Email a rate confirmation to the carrier, with a link to sign it while it is " +
		"unsigned. The PDF goes to the carrier's contacts that receive rate confirmations, " +
		"or the carrier's own address. The recipients come from the carrier's record and " +
		"cannot be chosen here; fix the carrier's contacts first if they are wrong. Send a " +
		"revision that is Generated, Sent (to send it again) or Confirmed (the executed " +
		"copy); find it with list_rate_confirmations."
}

func (t *SendRateConfirmationTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		fieldRateConfirmationID: agenttoolschema.RecordIDText(
			permission.ResourceRateConfirmation,
			"The revision to send, from list_rate_confirmations.",
		),
	}, fieldRateConfirmationID)
}

func (t *SendRateConfirmationTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceRateConfirmation,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Emails a binding agreement, with a link to sign it, to a carrier " +
			"outside the organization; the recipients come from the carrier's record, and " +
			"a sent email cannot be recalled, so a person decides every send.",
	}
}

func (t *SendRateConfirmationTool) Validate(
	_ context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, err := rateConfirmationArg(t, &params)

	return err
}

func (t *SendRateConfirmationTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	id, err := rateConfirmationArg(t, &params)
	if err != nil {
		return err
	}

	_, err = t.rateCons.Send(ctx, common.TenantFrom(params), id)

	return err
}

func (t *SendRateConfirmationTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, fieldRateConfirmationID, permission.ResourceRateConfirmation)
}

// voidRateConfirmationTool withdraws a revision, which is the console's void.
type VoidRateConfirmationTool struct {
	rateCons rateConfirmations
}

func NewVoidRateConfirmationTool(rateCons rateConfirmations) agentports.AgentTool {
	return &VoidRateConfirmationTool{rateCons: rateCons}
}

func (t *VoidRateConfirmationTool) Name() string { return "void_rate_confirmation" }

func (t *VoidRateConfirmationTool) Description() string {
	return "Void a rate confirmation so the revision stops standing and its sign link stops " +
		"working. Voiding one the carrier confirmed returns their assignment to " +
		"awaiting confirmation. Use it when the agreement as written is wrong and should " +
		"not be signed; to correct it, generate a new revision instead, which voids this " +
		"one. Say why in reason. Find the revision with list_rate_confirmations."
}

func (t *VoidRateConfirmationTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		fieldRateConfirmationID: agenttoolschema.RecordIDText(
			permission.ResourceRateConfirmation,
			"The revision to void, from list_rate_confirmations.",
		),
		common.FieldReason: jsonschemautils.Text(
			"Why the agreement is withdrawn, in a sentence a person can check.",
		),
	}, fieldRateConfirmationID, common.FieldReason)
}

func (t *VoidRateConfirmationTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceRateConfirmation,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Condition: &agentports.TierCondition{
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
	params agentports.ToolExecuteParams, //nolint:gocritic // a TierCondition passes params by value
) agent.AutonomyTier {
	id, reason, err := t.arguments(&params)
	if err != nil {
		return agent.TierPropose
	}

	plan, err := t.rateCons.PreviewVoid(ctx, common.TenantFrom(params), id, reason)
	if err != nil || plan.Before.Status != rateconfirmation.StatusGenerated {
		return agent.TierPropose
	}

	return agent.TierActWithApproval
}

func (t *VoidRateConfirmationTool) Validate(
	_ context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, _, err := t.arguments(&params)

	return err
}

func (t *VoidRateConfirmationTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	id, reason, err := t.arguments(&params)
	if err != nil {
		return err
	}

	_, err = t.rateCons.Void(ctx, common.TenantFrom(params), id, reason)

	return err
}

func (t *VoidRateConfirmationTool) arguments(
	params *agentports.ToolExecuteParams,
) (pulid.ID, string, error) {
	id, err := rateConfirmationArg(t, params)
	if err != nil {
		return pulid.Nil, "", err
	}
	reason, err := common.RequireString(params.Params, common.FieldReason)
	if err != nil {
		return pulid.Nil, "", err
	}

	return id, strings.TrimSpace(reason), nil
}

func (t *VoidRateConfirmationTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, fieldRateConfirmationID, permission.ResourceRateConfirmation)
}

// recordRateConfirmationConfirmedTool records that the carrier confirmed a
// revision outside its sign link, which is the console's mark confirmed.
type RecordRateConfirmationConfirmedTool struct {
	rateCons rateConfirmations
}

func NewRecordRateConfirmationConfirmedTool(rateCons rateConfirmations) agentports.AgentTool {
	return &RecordRateConfirmationConfirmedTool{rateCons: rateCons}
}

func (t *RecordRateConfirmationConfirmedTool) Name() string {
	return "record_rate_confirmation_confirmed"
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
			"The revision the carrier confirmed, from list_rate_confirmations.",
		),
		common.FieldConfirmedByName: jsonschemautils.Text(
			"The name of the person at the carrier who confirmed it.",
		),
	}, fieldRateConfirmationID, common.FieldConfirmedByName)
}

func (t *RecordRateConfirmationConfirmedTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceRateConfirmation,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Makes the revision the executed agreement to pay the carrier and " +
			"confirms their assignment, on the word of someone outside the organization; " +
			"voiding it afterwards withdraws the agreement rather than restoring it.",
	}
}

func (t *RecordRateConfirmationConfirmedTool) Validate(
	_ context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, _, err := t.arguments(&params)

	return err
}

func (t *RecordRateConfirmationConfirmedTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	id, name, err := t.arguments(&params)
	if err != nil {
		return err
	}

	_, err = t.rateCons.MarkConfirmed(ctx, common.TenantFrom(params), id, name)

	return err
}

func (t *RecordRateConfirmationConfirmedTool) arguments(
	params *agentports.ToolExecuteParams,
) (pulid.ID, string, error) {
	id, err := rateConfirmationArg(t, params)
	if err != nil {
		return pulid.Nil, "", err
	}
	name, err := common.RequireString(params.Params, common.FieldConfirmedByName)
	if err != nil {
		return pulid.Nil, "", err
	}

	return id, strings.TrimSpace(name), nil
}

func (t *RecordRateConfirmationConfirmedTool) Target(
	params map[string]any,
) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, fieldRateConfirmationID, permission.ResourceRateConfirmation)
}

func rateConfirmationArg(
	tool agentports.AgentTool,
	params *agentports.ToolExecuteParams,
) (pulid.ID, error) {
	if err := common.GuardPreview(tool, params); err != nil {
		return pulid.Nil, err
	}

	return common.RequirePulid(params.Params, fieldRateConfirmationID)
}
