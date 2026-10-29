package agentquerytoolservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPortals struct {
	sites   []agentextensionports.PortalSiteSummary
	read    *agentextensionports.PortalReadRequest
	outcome *agentextensionports.PortalReadOutcome
	err     error
}

func (s *stubPortals) ListPortalSites(
	context.Context,
	pagination.TenantInfo,
) ([]agentextensionports.PortalSiteSummary, error) {
	return s.sites, s.err
}

func (s *stubPortals) ReadPortal(
	_ context.Context,
	req *agentextensionports.PortalReadRequest,
) (*agentextensionports.PortalReadOutcome, error) {
	s.read = req

	return s.outcome, s.err
}

func (s *stubPortals) PlanPortalAction(
	context.Context,
	*agentextensionports.PortalActionRequest,
) (*agentextensionports.PortalActionPlan, error) {
	return nil, errors.New("not used")
}

func (s *stubPortals) StartPortalAction(
	context.Context,
	*agentextensionports.PortalActionRequest,
) (*agentextensionports.PortalActionStarted, error) {
	return nil, errors.New("not used")
}

func (s *stubPortals) TrackPortalRun(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	string,
) error {
	return errors.New("not used")
}

func TestListPortalSitesSaysWhichSitesAreReadOnly(t *testing.T) {
	t.Parallel()

	siteID := pulid.MustNew(agentextension.PortalSiteIDPrefix)
	portals := &stubPortals{sites: []agentextensionports.PortalSiteSummary{{
		ID:        siteID,
		Name:      "Acme DC",
		Purpose:   "dock appointments",
		Domains:   []string{"portal.acme.com"},
		HasSignIn: true,
	}}}

	result, err := newListPortalSitesTool(portals).Query(t.Context(), testParams(nil))
	require.NoError(t, err)

	answer, ok := result.(portalSitesAnswer)
	require.True(t, ok)
	assert.Equal(t, []portalSiteEntry{{
		ID:        siteID.String(),
		Name:      "Acme DC",
		Purpose:   "dock appointments",
		Addresses: []string{"portal.acme.com"},
		SignsIn:   true,
		ReadOnly:  true,
	}}, answer.Sites)
	assert.Equal(t, portalSitesGuidance, answer.Note)
}

func TestListPortalSitesWithNoneSetUpSaysWhoCanAddOne(t *testing.T) {
	t.Parallel()

	result, err := newListPortalSitesTool(&stubPortals{}).Query(t.Context(), testParams(nil))
	require.NoError(t, err)

	answer, ok := result.(portalSitesAnswer)
	require.True(t, ok)
	assert.Empty(t, answer.Sites)
	assert.Equal(t, portalNoSitesGuidance, answer.Note)
}

func TestReadPortalAsksTheQuestionAndMarksTheAnswerAsOutsideText(t *testing.T) {
	t.Parallel()

	siteID := pulid.MustNew(agentextension.PortalSiteIDPrefix)
	runID := pulid.MustNew(agentextension.PortalRunIDPrefix)
	portals := &stubPortals{outcome: &agentextensionports.PortalReadOutcome{
		RunID:       runID,
		Site:        agentextensionports.PortalSiteSummary{ID: siteID, Name: "Acme DC"},
		Status:      agentextension.PortalRunStatusSucceeded,
		Answer:      "Open slots: 8:00 AM and 2:00 PM.",
		VisitedURLs: []string{"https://portal.acme.com/appointments"},
		RetrievedAt: 1_790_150_400,
	}}
	params := testParams(map[string]any{
		"siteId":   siteID.String(),
		"question": " Which slots are open on October 20? ",
		"details":  map[string]any{"PO number": float64(4512)},
	})
	params.ThreadID = pulid.MustNew("athr_")

	result, err := newReadPortalTool(portals, nil).Query(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, portals.read)
	assert.Equal(t, siteID, portals.read.SiteID)
	assert.Equal(t, map[string]string{"PO number": "4512"}, portals.read.Details)
	assert.Equal(t, params.Actor.TenantInfo(), portals.read.TenantInfo)
	assert.Equal(t, params.ThreadID, portals.read.Owner.ThreadID)

	answer, ok := result.(portalReadAnswer)
	require.True(t, ok)
	assert.Equal(t, "Acme DC", answer.Site)
	assert.Equal(t, "Open slots: 8:00 AM and 2:00 PM.", answer.Answer)
	assert.Equal(t, []string{"https://portal.acme.com/appointments"}, answer.PagesRead)
	assert.Equal(t, runID.String(), answer.PortalRunID)
	assert.Equal(t, "2026-09-23", answer.RetrievedOn)
	assert.Equal(t, portalUntrustedReminder, answer.Note)
}

type stubPortalWaits struct {
	registered []*agentextensionports.RegisterPortalRunWaitRequest
	err        error
}

func (s *stubPortalWaits) RegisterPortalRunWait(
	_ context.Context,
	req *agentextensionports.RegisterPortalRunWaitRequest,
) error {
	s.registered = append(s.registered, req)

	return s.err
}

func (s *stubPortalWaits) NotifyPortalRun(
	context.Context,
	pagination.TenantInfo,
	*agentextension.PortalRun,
) {
}

func (s *stubPortalWaits) AnnouncePortalRun(
	context.Context,
	pagination.TenantInfo,
	*agentextension.PortalRun,
) {
}

func awaitingCodePortals(siteID, runID pulid.ID) *stubPortals {
	return &stubPortals{outcome: &agentextensionports.PortalReadOutcome{
		RunID:       runID,
		Site:        agentextensionports.PortalSiteSummary{ID: siteID, Name: "Acme DC"},
		Status:      agentextension.PortalRunStatusAwaitingCode,
		VisitedURLs: []string{"https://portal.acme.com/login"},
		CodePrompt:  "SIGN-IN CODE NEEDED: the portal texted a six-digit code to the phone on file.",
		RetrievedAt: 1_790_150_400,
	}}
}

func TestAReadStoppedForASignInCodeParksTheConversationOnIt(t *testing.T) {
	t.Parallel()

	siteID := pulid.MustNew(agentextension.PortalSiteIDPrefix)
	runID := pulid.MustNew(agentextension.PortalRunIDPrefix)
	waits := &stubPortalWaits{}
	params := testParams(map[string]any{
		"siteId":   siteID.String(),
		"question": "Which slots are open on October 20?",
	})
	params.ThreadID = pulid.MustNew("athr_")

	result, err := newReadPortalTool(awaitingCodePortals(siteID, runID), waits).
		Query(t.Context(), params)
	require.NoError(t, err)

	require.Len(t, waits.registered, 1)
	registered := waits.registered[0]
	assert.Equal(t, runID, registered.RunID)
	assert.Equal(t, "Acme DC", registered.SiteName)
	assert.Equal(t, params.ThreadID, registered.Owner.ThreadID)
	assert.Equal(t, params.Actor.TenantInfo(), registered.TenantInfo)

	answer, ok := result.(portalCodeAnswer)
	require.True(t, ok)
	assert.Equal(t, portalCodeAnswer{
		Site:               "Acme DC",
		Question:           "Which slots are open on October 20?",
		AwaitingSignInCode: true,
		PortalSaid: "SIGN-IN CODE NEEDED: the portal texted a six-digit code to the phone " +
			"on file.",
		PortalRunID: runID.String(),
		Note:        portalCodeGuidance,
	}, answer)
}

func TestAReadStoppedForASignInCodeThatCannotWaitSaysSo(t *testing.T) {
	t.Parallel()

	siteID := pulid.MustNew(agentextension.PortalSiteIDPrefix)
	runID := pulid.MustNew(agentextension.PortalRunIDPrefix)
	params := testParams(map[string]any{
		"siteId":   siteID.String(),
		"question": "Which slots are open on October 20?",
	})
	params.ThreadID = pulid.MustNew("athr_")

	_, err := newReadPortalTool(awaitingCodePortals(siteID, runID), &stubPortalWaits{
		err: errors.New("the conversation is gone"),
	}).Query(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), portalCodeUnwatched)
}

func TestReadPortalNeedsASite(t *testing.T) {
	t.Parallel()

	portals := &stubPortals{}
	_, err := newReadPortalTool(portals, nil).Query(t.Context(), testParams(map[string]any{
		"question": "Which slots are open on October 20?",
	}))
	require.Error(t, err)
	assert.Nil(t, portals.read)
}

func TestReadPortalDeclaresWhatItBringsIn(t *testing.T) {
	t.Parallel()

	policy := newReadPortalTool(&stubPortals{}, nil).Policy()
	assert.Equal(t, agenttooltrust.ExternalReadAlways, policy.ReadsExternal)
	assert.True(t, policy.CapsLaterWrites)

	listed := newListPortalSitesTool(&stubPortals{}).Policy()
	assert.NotEqual(t, agenttooltrust.ExternalReadAlways, listed.ReadsExternal)
}
