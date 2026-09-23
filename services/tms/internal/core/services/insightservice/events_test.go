package insightservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detector"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingEvents struct {
	published []services.AgentEvent
}

func (r *recordingEvents) Publish(_ context.Context, event services.AgentEvent) {
	r.published = append(r.published, event)
}

// detectingRepo stores what it is given and reports a chosen subset as new,
// the way the real repository reports the findings that were not active
// before the run.
type detectingRepo struct {
	stubRepo

	newKeys map[string]bool
}

func (r *detectingRepo) ReplaceDetectorFindings(
	_ context.Context,
	req repositories.ReplaceDetectorFindingsRequest,
) (repositories.ReplaceDetectorFindingsResult, error) {
	r.replaced = append(r.replaced, req)

	result := repositories.ReplaceDetectorFindingsResult{Created: len(req.Insights)}
	for _, entity := range req.Insights {
		if r.newKeys[entity.DedupeKey] {
			entity.ID = pulid.MustNew("inst_")
			result.Detected = append(result.Detected, entity)
		}
	}

	return result, nil
}

// A finding is announced once, when it first appears. The refresh that
// confirms it a day later supersedes the old row and says nothing, so an
// agent subscribed to the event is not woken every six hours for the same
// detention problem.
func TestRefresh_AnnouncesOnlyTheFindingsThatAreNew(t *testing.T) {
	t.Parallel()

	repo := &detectingRepo{newKeys: map[string]bool{"ontime-decline:cus_2": true}}
	events := &recordingEvents{}
	service := detectingService(repo, &stubDetector{
		key:      "ontime-decline",
		category: insight.CategoryServiceQuality,
		findings: []detector.Finding{
			testFinding("ontime-decline:cus_1"),
			testFinding("ontime-decline:cus_2"),
		},
	})
	service.events = events

	tenantInfo := tenant()
	_, err := service.Refresh(
		t.Context(),
		RefreshRequest{TenantInfo: tenantInfo, Now: 1_800_000_000},
	)
	require.NoError(t, err)

	require.Len(t, events.published, 1)
	published := events.published[0]
	assert.Equal(t, agent.EventInsightDetected, published.Kind)
	assert.Equal(t, tenantInfo, published.TenantInfo)
	assert.Equal(t, repo.replaced[0].Insights[1].ID, published.SubjectID)
	assert.True(t, published.SubjectID.IsNotNil())
}

func TestRefresh_RunsWithNobodyToAnnounceTo(t *testing.T) {
	t.Parallel()

	repo := &detectingRepo{newKeys: map[string]bool{"ontime-decline:cus_1": true}}
	service := detectingService(repo, &stubDetector{
		key:      "ontime-decline",
		category: insight.CategoryServiceQuality,
		findings: []detector.Finding{testFinding("ontime-decline:cus_1")},
	})

	result, err := service.Refresh(
		t.Context(),
		RefreshRequest{TenantInfo: tenant(), Now: 1_800_000_000},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
}

func detectingService(repo *detectingRepo, ds ...detector.Detector) *Service {
	service := svc(&stubRepo{}, ds...)
	service.repo = repo

	return service
}

type principalRecordingPermissions struct {
	stubPermissions

	principals []services.PrincipalType
}

func (p *principalRecordingPermissions) Check(
	ctx context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	p.principals = append(p.principals, req.PrincipalType)

	return p.stubPermissions.Check(ctx, req)
}

// An agent reading insights is checked as an agent, against what agents may
// see, rather than as a session user with no user id, which would either
// fail or, worse, be answered for the wrong principal.
func TestList_ChecksAnAgentActorAsAnAgent(t *testing.T) {
	t.Parallel()

	perms := &principalRecordingPermissions{stubPermissions: stubPermissions{
		allowedResources: map[permission.Resource]bool{permission.ResourceShipment: true},
	}}
	repo := &stubRepo{}
	service := newService(repo, &stubPermissions{}, &stubDetector{key: "ontime-decline"})
	service.permissions = perms

	_, err := service.List(t.Context(), services.BrowseInsightsRequest{
		TenantInfo: tenant(),
		Actor: &services.RequestActor{
			PrincipalType: services.PrincipalTypeAgent,
			PrincipalID:   services.AgentPrincipalID,
		},
	})
	require.NoError(t, err)

	require.Len(t, perms.principals, 1)
	assert.Equal(t, services.PrincipalTypeAgent, perms.principals[0])
	assert.Equal(
		t,
		repositories.AllowedDetectorKeys{"ontime-decline"},
		repo.lastBrowse.AllowedDetectorKeys,
	)
}

func TestGetDetail_ChecksAnAgentActorAsAnAgent(t *testing.T) {
	t.Parallel()

	perms := &principalRecordingPermissions{stubPermissions: stubPermissions{
		allowedResources: map[permission.Resource]bool{permission.ResourceShipment: true},
	}}
	found := &insight.Insight{
		ID:          pulid.MustNew("inst_"),
		DetectorKey: "ontime-decline",
		DedupeKey:   "k",
	}
	service := newService(
		&stubRepo{byID: found},
		&stubPermissions{},
		&stubDetector{key: "ontime-decline"},
	)
	service.permissions = perms

	detail, err := service.GetDetail(t.Context(), services.GetInsightDetailRequest{
		ID:         found.ID,
		TenantInfo: tenant(),
		Actor: &services.RequestActor{
			PrincipalType: services.PrincipalTypeAgent,
			PrincipalID:   services.AgentPrincipalID,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, found, detail.Insight)
	assert.Equal(t, []services.PrincipalType{services.PrincipalTypeAgent}, perms.principals)
}
