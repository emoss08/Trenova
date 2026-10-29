package memorytools

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeProfileGuide struct {
	agentprofileports.Service
	added  *agentprofileports.AddGuidanceRequest
	offers bool
}

func (f *fakeProfileGuide) PlanGuidance(
	_ context.Context,
	req *agentprofileports.AddGuidanceRequest,
) (*agentprofile.Guidance, error) {
	status := agentprofile.GuidanceStatusActive
	if f.offers && req.Ref.Subject == agentprofile.SubjectOrganization {
		status = agentprofile.GuidanceStatusSuggested
	}

	return &agentprofile.Guidance{
		OrganizationID: req.Ref.OrganizationID,
		BusinessUnitID: req.Ref.BusinessUnitID,
		Subject:        req.Ref.Subject,
		UserID:         req.Ref.PersonPtr(),
		Content:        req.Content,
		Status:         status,
		Source:         req.Source,
	}, nil
}

func (f *fakeProfileGuide) AddGuidance(
	_ context.Context,
	req *agentprofileports.AddGuidanceRequest,
) (*agentprofile.Guidance, error) {
	f.added = req

	return &agentprofile.Guidance{ID: pulid.MustNew("aprg_"), Content: req.Content}, nil
}

func TestGuideProfile_WritesGuidanceForTheProfileItNames(t *testing.T) {
	t.Parallel()

	guide := &fakeProfileGuide{}
	tool := NewGuideProfileTool(guide)

	mine := agenttoolstest.MemoryParams(map[string]any{
		"about":    "me",
		"guidance": "  I only handle the Midwest accounts.  ",
	})
	require.NoError(t, tool.Execute(t.Context(), mine))
	require.NotNil(t, guide.added)
	assert.Equal(t, agentprofile.SubjectUser, guide.added.Ref.Subject)
	assert.Equal(t, mine.Actor.UserID, guide.added.Ref.PersonID)
	assert.Equal(t, "I only handle the Midwest accounts.", guide.added.Content)
	assert.Equal(t, agentprofile.GuidanceSourceAgent, guide.added.Source)
	assert.Equal(t, agent.EgressPersonal, tool.Policy().Classified(mine).Egress,
		"guidance about the person's own work reaches only their conversations")

	org := agenttoolstest.MemoryParams(map[string]any{
		"about":    "organization",
		"guidance": "The Laredo lane is seasonal.",
	})
	require.NoError(t, tool.Execute(t.Context(), org))
	assert.Equal(t, agentprofile.SubjectOrganization, guide.added.Ref.Subject)
	assert.Equal(t, agent.EgressInternal, tool.Policy().Classified(org).Egress)
}

func TestGuideProfile_RefusesWhatItCannotWriteDown(t *testing.T) {
	t.Parallel()

	tool := NewGuideProfileTool(&fakeProfileGuide{})

	for name, params := range map[string]map[string]any{
		"no guidance":  {"about": "me"},
		"too long":     {"about": "me", "guidance": strings.Repeat("a", agentprofile.MaxGuidanceChars+1)},
		"unknown kind": {"about": "team", "guidance": "We are a carrier."},
	} {
		err := tool.Execute(t.Context(), agenttoolstest.MemoryParams(params))
		assert.Error(t, err, name)
	}

	agentRun := agenttoolstest.MemoryParams(map[string]any{
		"about":    "organization",
		"guidance": "We are a carrier.",
	})
	agentRun.Actor = &permissionports.RequestActor{
		PrincipalType: permissionports.PrincipalTypeAgent,
		PrincipalID:   pulid.MustNew("adef_"),
	}
	require.ErrorIs(t, tool.Execute(t.Context(), agentRun), errGuidanceNeedsPerson)
}

func TestGuideProfile_PreviewSaysWhatItWouldAdd(t *testing.T) {
	t.Parallel()

	tool := NewGuideProfileTool(&fakeProfileGuide{})
	previewer, ok := tool.(interface {
		Preview(context.Context, *agenttoolports.ToolExecuteParams) (*agentpreview.ToolPreview, error)
	})
	require.True(t, ok)

	preview, err := previewer.Preview(t.Context(), agenttoolstest.MemoryParams(map[string]any{
		"about":    "organization",
		"guidance": "The Laredo lane is seasonal.",
	}))
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "organization's business")
}

// Guidance about the organization from someone who does not manage agent
// memory waits for an administrator, and the preview says so.
func TestGuideProfile_PreviewSaysWhenItWouldBeOffered(t *testing.T) {
	t.Parallel()

	tool := NewGuideProfileTool(&fakeProfileGuide{offers: true})
	previewer, ok := tool.(interface {
		Preview(context.Context, *agenttoolports.ToolExecuteParams) (*agentpreview.ToolPreview, error)
	})
	require.True(t, ok)

	preview, err := previewer.Preview(t.Context(), agenttoolstest.MemoryParams(map[string]any{
		"about":    "organization",
		"guidance": "The Laredo lane is seasonal.",
	}))
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "Would offer this for an administrator")

	own, err := previewer.Preview(t.Context(), agenttoolstest.MemoryParams(map[string]any{
		"about":    "me",
		"guidance": "I only handle the Midwest accounts.",
	}))
	require.NoError(t, err)
	assert.Contains(t, own.Summary, "your own work")
}
