package editools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/edispec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const ApplyConnectionDraftToolName = "apply_edi_connection_draft"

var (
	errDraftDigest = errors.New(
		"the guide reads differently from the draft the person saw; run draft_edi_connection " +
			"again and propose its documentDigest",
	)
	errDraftNotReady = errors.New("the draft is not ready")
)

type connectionDrafts interface {
	PreviewConnectionDraft(
		ctx context.Context,
		req *ediports.ConnectionDraftRequest,
	) (*ediports.ConnectionDraftPlan, error)
	ApplyConnectionDraft(
		ctx context.Context,
		req *ediports.ApplyConnectionDraftRequest,
	) (*ediports.AppliedConnectionDraft, error)
}

type specReader interface {
	Read(ctx context.Context, req *edispec.ReadRequest) (*ediports.SpecDocument, error)
}

type ApplyConnectionDraftTool struct {
	drafts    connectionDrafts
	documents specReader
}

func NewApplyConnectionDraftTool(
	drafts connectionDrafts,
	documents specReader,
) agentports.AgentTool {
	return &ApplyConnectionDraftTool{drafts: drafts, documents: documents}
}

func (t *ApplyConnectionDraftTool) Name() string { return ApplyConnectionDraftToolName }

func (t *ApplyConnectionDraftTool) SearchTerms() []string {
	return []string{
		"save the drafted edi connection", "apply the edi draft",
		"approve the drafted connection", "commit the edi draft",
	}
}

func (t *ApplyConnectionDraftTool) Prerequisites() []string {
	return []string{"draft_edi_connection"}
}

func (t *ApplyConnectionDraftTool) Recipe() []string {
	return []string{
		"read_attachment",
		"draft_edi_connection",
		ApplyConnectionDraftToolName,
	}
}

func (t *ApplyConnectionDraftTool) Description() string {
	return "Propose saving the EDI connection draft_edi_connection drafted from a partner's guide. " +
		"It writes the partner, its communication profile, a document profile per document " +
		"type, control number starts and code mappings, all inactive until a person enters the " +
		"credentials and turns them on. For a partner that exists it changes only what the " +
		"guide changed. Pass the same input and the draft's documentDigest. A person always decides."
}

func (t *ApplyConnectionDraftTool) ParamSchema() map[string]any {
	return edispec.ParamSchema(true)
}

func (t *ApplyConnectionDraftTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceEDI,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadAlways,
		Source:        agent.TaintSourceAttachment,
		Rationale: "Sets up a trading partner from a guide the partner wrote, so a person " +
			"checks every value against its page before anything is saved" +
			ediDecisionRationaleEnd,
	}
}

func (t *ApplyConnectionDraftTool) input(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*edispec.Input, error) {
	input, err := edispec.Parse(params.Params, common.TenantFrom(*params))
	if err != nil {
		return nil, err
	}
	if input.Digest == "" {
		return nil, fmt.Errorf("parameter %q is required", edispec.ParamDocumentDigest)
	}
	document, err := t.documents.Read(ctx, &edispec.ReadRequest{
		Tenant:     common.TenantFrom(*params),
		DocumentID: input.DocumentID,
		ThreadID:   params.ThreadID,
	})
	if err != nil {
		return nil, err
	}
	input.Draft.Document = *document
	return input, nil
}

func (t *ApplyConnectionDraftTool) plan(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*edispec.Input, *ediports.ConnectionDraftPlan, error) {
	input, err := t.input(ctx, params)
	if err != nil {
		return nil, nil, err
	}
	plan, err := t.drafts.PreviewConnectionDraft(ctx, input.Draft)
	if err != nil {
		return nil, nil, err
	}
	return input, plan, nil
}

func readiness(input *edispec.Input, draft *ediports.ConnectionDraft) error {
	if draft.DocumentDigest != input.Digest {
		return errDraftDigest
	}
	if !draft.Ready {
		return fmt.Errorf("%w: %s", errDraftNotReady, strings.Join(draft.Blockers, "; "))
	}
	return nil
}

func (t *ApplyConnectionDraftTool) ResolveSelection(
	_ context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolSelectionResolver interface passes params by value
) (map[string]any, error) {
	return edispec.RedactQuotes(params.Params), nil
}

func (t *ApplyConnectionDraftTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	if err := common.GuardPreview(t, &params); err != nil {
		return err
	}
	input, plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}
	return readiness(input, plan.Draft)
}

func (t *ApplyConnectionDraftTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}
	if !params.ApprovedFromProposal() {
		return ErrEDIDecisionNeedsAPerson
	}

	input, err := t.input(ctx, &params)
	if err != nil {
		return err
	}
	if _, err = t.drafts.ApplyConnectionDraft(ctx, &ediports.ApplyConnectionDraftRequest{
		Draft:                  input.Draft,
		ExpectedDocumentDigest: input.Digest,
		Actor:                  params.Actor,
	}); err != nil {
		return fmt.Errorf("write the EDI connection: %w", err)
	}
	return nil
}
