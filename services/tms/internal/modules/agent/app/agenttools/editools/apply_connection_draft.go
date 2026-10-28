package editools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/edispec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const (
	ApplyConnectionDraftToolName = "edi_apply_connection_draft"
	ediPartnerRecordEntity       = "edi_partner"
)

var (
	errDraftDigest = errors.New(
		"the guide reads differently from the draft the person saw; run edi_draft_connection " +
			"again and propose its documentDigest",
	)
	errDraftNotReady = errors.New("the draft is not ready")
)

type connectionDrafts interface {
	PreviewConnectionDraft(
		ctx context.Context,
		req *ediconnectiondraftports.ConnectionDraftRequest,
	) (*ediconnectiondraftports.ConnectionDraftPlan, error)
	ApplyConnectionDraft(
		ctx context.Context,
		req *ediconnectiondraftports.ApplyConnectionDraftRequest,
	) (*ediconnectiondraftports.AppliedConnectionDraft, error)
}

type specReader interface {
	Read(
		ctx context.Context,
		req *edispec.ReadRequest,
	) (*ediconnectiondraftports.SpecDocument, error)
}

type ApplyConnectionDraftTool struct {
	drafts    connectionDrafts
	documents specReader
}

func NewApplyConnectionDraftTool(
	drafts connectionDrafts,
	documents specReader,
) agenttoolports.AgentTool {
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
	return []string{"edi_draft_connection"}
}

func (t *ApplyConnectionDraftTool) Recipe() []string {
	return []string{
		"agent_read_attachment",
		"edi_draft_connection",
		ApplyConnectionDraftToolName,
	}
}

func (t *ApplyConnectionDraftTool) Description() string {
	return "Propose saving the EDI connection edi_draft_connection drafted from a partner's guide. " +
		"It writes the partner, its communication profile, a document profile per document " +
		"type, control number starts and code mappings, all inactive until a person enters the " +
		"credentials and turns them on. For a partner that exists it changes only what the " +
		"guide changed. Pass the same input and the draft's documentDigest. A person always decides."
}

func (t *ApplyConnectionDraftTool) ParamSchema() map[string]any {
	return edispec.ParamSchema(true)
}

func (t *ApplyConnectionDraftTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
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
		Artifact:      ediPartnerRecordEntity,
		Rationale: "Sets up a trading partner from a guide the partner wrote, so a person " +
			"checks every value against its page before anything is saved" +
			ediDecisionRationaleEnd,
	}
}

func (t *ApplyConnectionDraftTool) input(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
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
	params *agenttoolports.ToolExecuteParams,
) (*edispec.Input, *ediconnectiondraftports.ConnectionDraftPlan, error) {
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

func readiness(input *edispec.Input, draft *ediconnectiondraftports.ConnectionDraft) error {
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
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolSelectionResolver interface passes params by value
) (map[string]any, error) {
	return edispec.RedactQuotes(params.Params), nil
}

func (t *ApplyConnectionDraftTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
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
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)
	return err
}

func (t *ApplyConnectionDraftTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := common.GuardExecute(t, params); err != nil {
		return nil, err
	}
	if !params.ApprovedFromProposal() {
		return nil, ErrEDIDecisionNeedsAPerson
	}

	input, err := t.input(ctx, &params)
	if err != nil {
		return nil, err
	}
	applied, err := t.drafts.ApplyConnectionDraft(
		ctx,
		&ediconnectiondraftports.ApplyConnectionDraftRequest{
			Draft:                  input.Draft,
			ExpectedDocumentDigest: input.Digest,
			Actor:                  params.Actor,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("write the EDI connection: %w", err)
	}
	return appliedResult(applied), nil
}

func appliedResult(
	applied *ediconnectiondraftports.AppliedConnectionDraft,
) *agent.ToolExecutionResult {
	action := "updated"
	if applied.Created {
		action = "created"
	}
	name := ""
	if applied.Draft != nil {
		name = draftPartnerName(applied.Draft)
	}
	return &agent.ToolExecutionResult{
		Action: action,
		Kind:   "EDI connection",
		Name:   name,
		IDs:    map[string]string{"ediPartnerId": applied.PartnerID.String()},
		Record: &agent.RecordRef{
			EntityType: ediPartnerRecordEntity,
			ID:         applied.PartnerID.String(),
		},
		State: appliedState(applied),
	}
}

func appliedState(applied *ediconnectiondraftports.AppliedConnectionDraft) string {
	order := make([]string, 0, len(applied.Writes))
	counts := make(map[string]int, len(applied.Writes))
	for _, write := range applied.Writes {
		label := strings.ToLower(write.Record)
		if counts[label] == 0 {
			order = append(order, label)
		}
		counts[label]++
	}
	parts := make([]string, 0, len(order))
	for _, label := range order {
		parts = append(
			parts,
			fmt.Sprintf("%d %s", counts[label], plural(counts[label], label, label+"s")),
		)
	}

	state := "Saved: " + strings.Join(parts, ", ") +
		". Anything new is inactive and what existed keeps its status."
	if applied.Created {
		state = "Saved inactive: " + strings.Join(parts, ", ") + "."
	}
	if missing := applied.Draft.CredentialsToEnter(); len(missing) > 0 {
		state += " Before turning it on, enter: " + strings.Join(missing, ", ") + "."
	}
	return state
}
