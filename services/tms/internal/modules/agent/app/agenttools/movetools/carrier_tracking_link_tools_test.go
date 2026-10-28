package movetools

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const issuedLinkURL = "https://track.example.test/carrier-tracking/secret-token"

type fakeTrackingLinks struct {
	guard    agenttoolstest.WriteGuard
	live     *checkcall.TrackingLink
	refuse   error
	issued   []*checkcallports.IssueTrackingLinkRequest
	revoked  []*checkcallports.RevokeTrackingLinkRequest
	emailErr string
}

func (f *fakeTrackingLinks) PreviewIssueLink(
	_ context.Context,
	req *checkcallports.IssueTrackingLinkRequest,
) (*checkcallports.TrackingLinkPreview, error) {
	if f.refuse != nil {
		return nil, f.refuse
	}
	return &checkcallports.TrackingLinkPreview{
		MoveID:            req.MoveID,
		ShipmentID:        pulid.MustNew("shp_"),
		ShipmentProNumber: "PRO-4410",
		CarrierID:         pulid.MustNew("car_"),
		CarrierName:       "Lone Star Hauling",
		RecipientEmail:    req.RecipientEmail,
		ExpiresAt:         checkCallNow + checkcall.DefaultLinkTTLSeconds,
		Replaces:          f.live,
		Email: &emailports.PlannedEmail{
			To:      []string{req.RecipientEmail},
			Subject: "Tracking link for PRO-4410",
			Text:    "Post check calls here: LINK-MADE-WHEN-ISSUED",
		},
	}, nil
}

func (f *fakeTrackingLinks) IssueLink(
	_ context.Context,
	req *checkcallports.IssueTrackingLinkRequest,
) (*checkcallports.IssuedTrackingLink, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	if f.refuse != nil {
		return nil, f.refuse
	}
	f.issued = append(f.issued, req)
	return &checkcallports.IssuedTrackingLink{
		Link: &checkcall.TrackingLink{
			ID:             pulid.MustNew("ctl_"),
			ShipmentID:     pulid.MustNew("shp_"),
			ShipmentMoveID: req.MoveID,
			RecipientEmail: req.RecipientEmail,
		},
		URL:        issuedLinkURL,
		Emailed:    f.emailErr == "",
		EmailError: f.emailErr,
	}, nil
}

func (f *fakeTrackingLinks) PreviewRevokeLink(
	_ context.Context,
	_ *checkcallports.RevokeTrackingLinkRequest,
) (*checkcall.TrackingLink, error) {
	if f.live == nil {
		return nil, errortypes.NewNotFoundError("This load has no active carrier tracking link")
	}
	return f.live, nil
}

func (f *fakeTrackingLinks) RevokeLink(
	_ context.Context,
	req *checkcallports.RevokeTrackingLinkRequest,
) error {
	if err := f.guard.Write(); err != nil {
		return err
	}
	if f.live == nil {
		return errortypes.NewNotFoundError("This load has no active carrier tracking link")
	}
	f.revoked = append(f.revoked, req)
	return nil
}

func liveTrackingLink() *checkcall.TrackingLink {
	return &checkcall.TrackingLink{
		ID:             pulid.MustNew("ctl_"),
		RecipientEmail: "old-dispatch@lonestar.test",
		ExpiresAt:      checkCallNow + 3_600,
	}
}

func TestIssueCarrierTrackingLink_EmailsTheCarrierAndNeverHandsTheLinkBack(t *testing.T) {
	t.Parallel()

	links := &fakeTrackingLinks{}
	tool := newIssueCarrierTrackingLinkTool(links)
	moveID := pulid.MustNew("sm_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		"moveId":         moveID.String(),
		"recipientEmail": "dispatch@lonestar.test",
	})

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)

	require.Len(t, links.issued, 1)
	issued := links.issued[0]
	assert.Equal(t, moveID, issued.MoveID)
	assert.Equal(t, "dispatch@lonestar.test", issued.RecipientEmail)
	assert.Equal(t, params.Actor.UserID, issued.IssuedByID)
	assert.Equal(t, params.OrganizationID, issued.TenantInfo.OrgID)
	assert.Equal(t, "Emailed to dispatch@lonestar.test", result.State)
	require.NotNil(t, result.Record)
	assert.Equal(t, "shipment", result.Record.EntityType)
	for _, text := range []string{result.Name, result.State} {
		assert.NotContains(t, text, "secret-token", "the bearer link never reaches the model")
	}
	for _, id := range result.IDs {
		assert.False(t, strings.Contains(id, "secret-token"))
	}
}

func TestIssueCarrierTrackingLink_SaysWhenTheEmailFailed(t *testing.T) {
	t.Parallel()

	links := &fakeTrackingLinks{emailErr: "mailbox unavailable"}
	tool := newIssueCarrierTrackingLinkTool(links)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"moveId":         pulid.MustNew("sm_").String(),
		"recipientEmail": "dispatch@lonestar.test",
	})

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, result.State, "mailbox unavailable")
	assert.NotContains(t, result.State, "secret-token")
}

func TestIssueCarrierTrackingLink_NeedsAnAddressToSendTo(t *testing.T) {
	t.Parallel()

	links := &fakeTrackingLinks{}
	tool := newIssueCarrierTrackingLinkTool(links)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"moveId": pulid.MustNew("sm_").String(),
	})

	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Empty(t, links.issued)
}

func TestIssueCarrierTrackingLink_PreviewIsTheEmailAndTheLinkItReplaces(t *testing.T) {
	t.Parallel()

	links := &fakeTrackingLinks{live: liveTrackingLink()}
	tool := newIssueCarrierTrackingLinkTool(links)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"moveId":         pulid.MustNew("sm_").String(),
		"recipientEmail": "dispatch@lonestar.test",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, &links.guard, func() (
		*agent.ToolPreview,
		error,
	) {
		return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	})

	require.Len(t, preview.Changes, 2)
	created := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)
	assert.Equal(t, permission.ResourceShipmentMove, created.Resource)
	require.NotNil(t, created.Message)
	assert.Equal(t, agent.MessageChannelEmail, created.Message.Channel)
	assert.Equal(t, []string{"dispatch@lonestar.test"}, created.Message.To)
	assert.True(t, agenttoolstest.FieldByPath(t, created, "expiresAt").Volatile)

	replaced := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, agent.PreviewOperationUpdate, replaced.Operation)
	assert.Contains(t, preview.Summary, "Lone Star Hauling")
	assert.Contains(t, preview.Summary, "stops working")
	assert.Empty(t, links.issued)
}

func TestIssueCarrierTrackingLink_APreviewTheServiceRefusesWarnsItWouldFail(t *testing.T) {
	t.Parallel()

	links := &fakeTrackingLinks{refuse: errortypes.NewBusinessError(
		"Only a load covered by an outside carrier gets a carrier tracking link",
	)}
	tool := newIssueCarrierTrackingLinkTool(links)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"moveId":         pulid.MustNew("sm_").String(),
		"recipientEmail": "dispatch@lonestar.test",
	})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.NotEmpty(t, preview.Warnings)
	assert.Equal(t, agent.PreviewWarningWouldFail, preview.Warnings[0].Code)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
}

func TestCarrierTrackingLinks_ReachOutsideAndStopAtAskFirst(t *testing.T) {
	t.Parallel()

	for _, tool := range []agenttoolports.AgentTool{
		newIssueCarrierTrackingLinkTool(nil),
		newRevokeCarrierTrackingLinkTool(nil),
	} {
		policy := tool.Policy()
		assert.Equal(t, []agent.EgressClass{agent.EgressExternalRecipient}, policy.Egress,
			tool.Name())
		assert.Equal(t, permission.ResourceShipmentMove, policy.Resource, tool.Name())
		assert.Equal(t, permission.OpUpdate, policy.Operation, tool.Name())
		assert.Equal(t, agent.TierActWithApproval, policy.MaxTier, tool.Name())
		assert.Equal(t, agent.TierPropose, policy.DefaultTier, tool.Name())
	}
}

func TestRevokeCarrierTrackingLink_PreviewsTheLiveLinkThenRevokesIt(t *testing.T) {
	t.Parallel()

	links := &fakeTrackingLinks{live: liveTrackingLink()}
	tool := newRevokeCarrierTrackingLinkTool(links)
	moveID := pulid.MustNew("sm_")
	params := agenttoolstest.ExecuteParams(map[string]any{"moveId": moveID.String()})

	preview := agenttoolstest.PreviewWithoutWrites(t, &links.guard, func() (
		*agent.ToolPreview,
		error,
	) {
		return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "old-dispatch@lonestar.test")
	assert.Empty(t, links.revoked)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.Len(t, links.revoked, 1)
	assert.Equal(t, moveID, links.revoked[0].MoveID)
}

func TestRevokeCarrierTrackingLink_RefusesAMoveWithNoLiveLink(t *testing.T) {
	t.Parallel()

	links := &fakeTrackingLinks{}
	tool := newRevokeCarrierTrackingLinkTool(links)
	params := agenttoolstest.ExecuteParams(map[string]any{"moveId": pulid.MustNew("sm_").String()})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.NotEmpty(t, preview.Warnings)
	assert.Equal(t, agent.PreviewWarningWouldFail, preview.Warnings[0].Code)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	assert.Empty(t, links.revoked)
}
