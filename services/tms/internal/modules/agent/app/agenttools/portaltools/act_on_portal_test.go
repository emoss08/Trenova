package portaltools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentextensionservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePortals struct {
	planErr  error
	status   agentextension.PortalRunStatus
	planned  []*agentextensionports.PortalActionRequest
	started  []*agentextensionports.PortalActionRequest
	tracked  []pulid.ID
	trackErr error
	runID    pulid.ID
	site     agentextensionports.PortalSiteSummary

	authenticator bool
}

func newFakePortals() *fakePortals {
	return &fakePortals{
		status: agentextension.PortalRunStatusRunning,
		runID:  pulid.MustNew(agentextension.PortalRunIDPrefix),
		site: agentextensionports.PortalSiteSummary{
			ID:           pulid.MustNew(agentextension.PortalSiteIDPrefix),
			Name:         "Acme DC",
			Domains:      []string{"portal.acme.com"},
			AllowActions: true,
			HasSignIn:    true,
		},
	}
}

func (f *fakePortals) ListPortalSites(
	context.Context,
	pagination.TenantInfo,
) ([]agentextensionports.PortalSiteSummary, error) {
	return []agentextensionports.PortalSiteSummary{f.site}, nil
}

func (f *fakePortals) ReadPortal(
	context.Context,
	*agentextensionports.PortalReadRequest,
) (*agentextensionports.PortalReadOutcome, error) {
	return nil, errors.New("not used")
}

func (f *fakePortals) PlanPortalAction(
	_ context.Context,
	req *agentextensionports.PortalActionRequest,
) (*agentextensionports.PortalActionPlan, error) {
	f.planned = append(f.planned, req)
	if f.planErr != nil {
		return nil, f.planErr
	}

	return &agentextensionports.PortalActionPlan{
		Site:        f.site,
		Instruction: req.Instruction,
		Details:     req.Details,
		SentTask:    "On the Acme DC portal, do this: " + req.Instruction,
		UsesSignIn:  f.site.HasSignIn,
		Vendor:      "Browser Use",

		UsesAuthenticator: f.authenticator,
	}, nil
}

func (f *fakePortals) StartPortalAction(
	_ context.Context,
	req *agentextensionports.PortalActionRequest,
) (*agentextensionports.PortalActionStarted, error) {
	f.started = append(f.started, req)

	return &agentextensionports.PortalActionStarted{
		RunID:  f.runID,
		Site:   f.site,
		Status: f.status,
	}, nil
}

func (f *fakePortals) TrackPortalRun(
	_ context.Context,
	_ pagination.TenantInfo,
	runID pulid.ID,
	_ string,
) error {
	f.tracked = append(f.tracked, runID)

	return f.trackErr
}

type fakeWaits struct {
	registered []*agentextensionports.RegisterPortalRunWaitRequest
	err        error
}

func (f *fakeWaits) RegisterPortalRunWait(
	_ context.Context,
	req *agentextensionports.RegisterPortalRunWaitRequest,
) error {
	f.registered = append(f.registered, req)

	return f.err
}

func (f *fakeWaits) NotifyPortalRun(
	context.Context,
	pagination.TenantInfo,
	*agentextension.PortalRun,
) {
}

func (f *fakeWaits) AnnouncePortalRun(
	context.Context,
	pagination.TenantInfo,
	*agentextension.PortalRun,
) {
}

func actArgs(siteID pulid.ID) map[string]any {
	return map[string]any{
		paramSiteID:      siteID.String(),
		paramInstruction: "Book the 8:00 AM dock appointment on October 20",
		paramDetails:     map[string]any{"PO number": "4512"},
	}
}

func TestActOnPortalAlwaysAsksAPerson(t *testing.T) {
	t.Parallel()

	policy := NewActOnPortalTool(newFakePortals(), &fakeWaits{}, nil).Policy()
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressExternalRecipient}, policy.Egress)
	assert.Equal(t, permission.ResourcePortalAutomation, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
}

func TestActOnPortalRefusesToRunWithoutAnApproval(t *testing.T) {
	t.Parallel()

	portals := newFakePortals()
	tool := NewActOnPortalTool(portals, &fakeWaits{}, nil).(*ActOnPortalTool)

	_, err := tool.ExecuteWithResult(t.Context(),
		agenttoolstest.AgentParamsFor(actArgs(portals.site.ID)))
	require.ErrorIs(t, err, ErrPortalActionNeedsAPerson)
	assert.Empty(t, portals.started)
}

func TestActOnPortalPreviewShowsWhatTheBrowserWillBeTold(t *testing.T) {
	t.Parallel()

	portals := newFakePortals()
	tool := NewActOnPortalTool(portals, &fakeWaits{}, nil).(*ActOnPortalTool)

	preview, err := tool.Preview(t.Context(),
		agenttoolstest.AgentParamsFor(actArgs(portals.site.ID)))
	require.NoError(t, err)
	assert.Empty(t, preview.Warnings)
	assert.Equal(t,
		"Would ask Browser Use's browser to do this on Acme DC: Book the 8:00 AM dock "+
			"appointment on October 20. It signs in with the site's stored sign-in.",
		preview.Summary)

	require.Len(t, preview.Changes, 1)
	change := preview.Changes[0]
	assert.Equal(t, agentpreview.PreviewOperationSend, change.Operation)
	assert.Equal(t, permission.ResourcePortalAutomation, change.Resource)
	require.NotNil(t, change.Message)
	assert.Equal(t, agentpreview.MessageChannelPortal, change.Message.Channel)
	assert.Equal(t, []string{"portal.acme.com"}, change.Message.To)
	assert.Equal(t, "Book the 8:00 AM dock appointment on October 20", change.Message.Subject)
	assert.Equal(t,
		"On the Acme DC portal, do this: Book the 8:00 AM dock appointment on October 20",
		change.Message.Body)

	require.Len(t, portals.planned, 1)
	assert.Equal(t, map[string]string{"PO number": "4512"}, portals.planned[0].Details)
	assert.Empty(t, portals.started, "a preview never reaches the vendor")
}

func TestActOnPortalPreviewWarnsWhenTheSiteIsReadOnly(t *testing.T) {
	t.Parallel()

	portals := newFakePortals()
	portals.planErr = &agentextensionservice.ToolError{
		Message: "Agents may only read Acme DC, not act on it.",
	}
	tool := NewActOnPortalTool(portals, &fakeWaits{}, nil).(*ActOnPortalTool)

	preview, err := tool.Preview(t.Context(),
		agenttoolstest.AgentParamsFor(actArgs(portals.site.ID)))
	require.NoError(t, err)
	require.Len(t, preview.Warnings, 1)
	assert.Equal(t, agentpreview.PreviewWarningWouldFail, preview.Warnings[0].Code)
	assert.Empty(t, preview.Changes)

	require.Error(t, tool.Validate(t.Context(),
		agenttoolstest.AgentParamsFor(actArgs(portals.site.ID))))
}

func TestActOnPortalStartsTheTaskParksTheWorkAndFollowsIt(t *testing.T) {
	t.Parallel()

	portals := newFakePortals()
	waits := &fakeWaits{}
	tool := NewActOnPortalTool(portals, waits, nil).(*ActOnPortalTool)
	params := agenttoolstest.Approved(agenttoolstest.AgentParamsFor(actArgs(portals.site.ID)))
	params.ThreadID = pulid.MustNew("athr_")

	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "started", result.Action)
	assert.Equal(t, "portal task", result.Kind)
	assert.Equal(t, "Acme DC", result.Name)
	assert.Equal(t, map[string]string{resultRunID: portals.runID.String()}, result.IDs)
	assert.Equal(t, &agentrecord.RecordRef{EntityType: "portal_run", ID: portals.runID.String()},
		result.Record)
	assert.Equal(t,
		"Running on Acme DC. The work picks up here with what the portal said when it finishes.",
		result.State)

	require.Len(t, portals.started, 1)
	assert.Equal(t, params.IdempotencyKey, portals.started[0].IdempotencyKey)
	assert.Equal(t, params.ProposalID, portals.started[0].Owner.ProposalID)
	require.Len(t, waits.registered, 1)
	assert.Equal(t, portals.runID, waits.registered[0].RunID)
	assert.Equal(t, params.ThreadID, waits.registered[0].Owner.ThreadID)
	assert.Equal(t, []pulid.ID{portals.runID}, portals.tracked)
}

func TestActOnPortalStillFollowsATaskWhoseWaitCouldNotBeSet(t *testing.T) {
	t.Parallel()

	portals := newFakePortals()
	tool := NewActOnPortalTool(portals, &fakeWaits{err: errors.New("db down")}, nil).(*ActOnPortalTool)

	result, err := tool.ExecuteWithResult(t.Context(),
		agenttoolstest.Approved(agenttoolstest.AgentParamsFor(actArgs(portals.site.ID))))
	require.NoError(t, err)
	assert.Equal(t,
		"Running on Acme DC. Its outcome appears under AI Control, Extensions when it finishes.",
		result.State)
	assert.Equal(t, []pulid.ID{portals.runID}, portals.tracked)
}

func TestActOnPortalRetriedAfterTheTaskEndedReportsHowItEnded(t *testing.T) {
	t.Parallel()

	portals := newFakePortals()
	portals.status = agentextension.PortalRunStatusSucceeded
	waits := &fakeWaits{}
	tool := NewActOnPortalTool(portals, waits, nil).(*ActOnPortalTool)

	result, err := tool.ExecuteWithResult(t.Context(),
		agenttoolstest.Approved(agenttoolstest.AgentParamsFor(actArgs(portals.site.ID))))
	require.NoError(t, err)
	assert.Equal(t, "ran", result.Action)
	assert.Equal(t, "Finished. Read what the portal said under AI Control, Extensions.",
		result.State)
	assert.Empty(t, waits.registered)
	assert.Empty(t, portals.tracked)
}

func TestActOnPortalRefusesDetailsThatAreNotText(t *testing.T) {
	t.Parallel()

	portals := newFakePortals()
	tool := NewActOnPortalTool(portals, &fakeWaits{}, nil).(*ActOnPortalTool)
	args := actArgs(portals.site.ID)
	args[paramDetails] = map[string]any{"Nested": map[string]any{"a": "b"}}

	err := tool.Validate(t.Context(), agenttoolstest.AgentParamsFor(args))
	require.Error(t, err)
	assert.Empty(t, portals.planned)
}

var _ agenttoolports.ToolPreviewer = (*ActOnPortalTool)(nil)

func TestActOnPortalPreviewSaysWhenItAnswersAnAuthenticatorCode(t *testing.T) {
	t.Parallel()

	portals := newFakePortals()
	portals.authenticator = true
	tool := NewActOnPortalTool(portals, &fakeWaits{}, nil).(*ActOnPortalTool)

	preview, err := tool.Preview(t.Context(),
		agenttoolstest.AgentParamsFor(actArgs(portals.site.ID)))
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(preview.Summary,
		"It signs in with the site's stored sign-in and answers its authenticator code."))
}
