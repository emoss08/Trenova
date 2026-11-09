package auditreviewservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var tenant = pagination.TenantInfo{OrgID: pulid.ID("org_1"), BuID: pulid.ID("bu_1")}

type fakeRepo struct {
	raised   []*audit.Review
	created  bool
	stored   *audit.Review
	setReqs  []*auditports.SetReviewedRequest
	setErr   error
	listReq  *auditports.ListReviewsRequest
	byEntry  []*audit.Review
	listResp *auditports.ReviewList
}

func (f *fakeRepo) Raise(_ context.Context, review *audit.Review) (*audit.Review, bool, error) {
	f.raised = append(f.raised, review)
	return review, f.created, nil
}

func (f *fakeRepo) List(
	_ context.Context,
	req *auditports.ListReviewsRequest,
) (*auditports.ReviewList, error) {
	f.listReq = req
	return f.listResp, nil
}

func (f *fakeRepo) GetByID(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (*audit.Review, error) {
	if f.stored == nil {
		return nil, errortypes.NewNotFoundError("Audit review not found")
	}
	copied := *f.stored
	return &copied, nil
}

func (f *fakeRepo) GetByEntryIDs(
	context.Context,
	pagination.TenantInfo,
	[]pulid.ID,
) ([]*audit.Review, error) {
	return f.byEntry, nil
}

func (f *fakeRepo) SetReviewed(
	_ context.Context,
	req *auditports.SetReviewedRequest,
) (*audit.Review, error) {
	f.setReqs = append(f.setReqs, req)
	if f.setErr != nil {
		return nil, f.setErr
	}
	updated := *f.stored
	if req.Reviewed {
		at := req.At
		updated.ReviewedAt = &at
		updated.ReviewedByID = req.ReviewerID
	} else {
		updated.ReviewedAt = nil
		updated.ReviewedByID = ""
	}
	return &updated, nil
}

type fakeTimezones struct{ zone string }

func (f fakeTimezones) OrganizationTimezone(
	context.Context,
	pagination.TenantInfo,
) (string, error) {
	return f.zone, nil
}

func (fakeTimezones) Overview(
	context.Context,
	*auditports.OverviewRequest,
) (*auditports.OverviewRollup, error) {
	return nil, errors.New("not used")
}

type recordingAuditor struct {
	changes []*permissionports.SecurityChange
}

func (a *recordingAuditor) RecordChange(_ context.Context, change *permissionports.SecurityChange) {
	a.changes = append(a.changes, change)
}

type recordingRealtime struct {
	published []*realtimeports.PublishResourceInvalidationRequest
}

func (r *recordingRealtime) PublishResourceInvalidation(
	_ context.Context,
	req *realtimeports.PublishResourceInvalidationRequest,
) error {
	r.published = append(r.published, req)
	return nil
}

const now = int64(1_800_000_000)

func newService(repo *fakeRepo) (*Service, *recordingAuditor, *recordingRealtime) {
	auditor := &recordingAuditor{}
	realtime := &recordingRealtime{}
	return New(Params{
		Repo:      repo,
		Timezones: fakeTimezones{zone: "America/Chicago"},
		Auditor:   auditor,
		Realtime:  realtime,
		Now:       func() int64 { return now },
		Logger:    zap.NewNop(),
	}), auditor, realtime
}

func payload() *auditports.AlertPayload {
	return &auditports.AlertPayload{
		TenantInfo:    tenant,
		Kind:          audit.AlertBulkDelete,
		DedupeKey:     "BulkDelete/usr_1:2900000",
		EntryID:       pulid.ID("ae_1"),
		Action:        "accessorial_rate.delete",
		Resource:      permission.Resource("accessorial_rate"),
		PrincipalID:   pulid.ID("usr_1"),
		PrincipalType: "session_user",
		ActorName:     "Dana Whitfield",
		Count:         50,
		Threshold:     50,
		WindowMinutes: 10,
		OccurredAt:    now - 3600,
	}
}

func TestRaiseKeepsTheAlertsFactsAndAnnouncesOnlyANewReview(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{created: true}
	svc, _, realtime := newService(repo)
	require.NoError(t, svc.Raise(t.Context(), payload()))

	require.Len(t, repo.raised, 1)
	got := repo.raised[0]
	assert.Equal(t, audit.AlertBulkDelete, got.Kind)
	assert.Equal(t, "BulkDelete/usr_1:2900000", got.DedupeKey)
	assert.Equal(t, pulid.ID("ae_1"), got.EntryID)
	assert.Equal(t, int64(50), got.Count)
	assert.Equal(t, 10, got.WindowMinutes)
	assert.Nil(t, got.ReviewedAt)
	require.Len(t, realtime.published, 1)
	assert.Equal(t, RealtimeResource, realtime.published[0].Resource)
	assert.Equal(t, "created", realtime.published[0].Action)

	repo.created = false
	require.NoError(t, svc.Raise(t.Context(), payload()))
	assert.Len(t, realtime.published, 1, "a replayed alert finds its review and announces nothing")
}

func TestRaiseNeverStoresAZeroCountOrWindow(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{created: true}
	svc, _, _ := newService(repo)
	p := payload()
	p.Count, p.Threshold, p.WindowMinutes = 0, 0, 0
	require.NoError(t, svc.Raise(t.Context(), p))

	assert.Equal(t, int64(1), repo.raised[0].Count)
	assert.Equal(t, 1, repo.raised[0].Threshold)
	assert.Equal(t, 1, repo.raised[0].WindowMinutes)
}

func TestMarkReviewedRecordsWhoAndAnnouncesIt(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{stored: &audit.Review{
		ID: pulid.ID("arv_1"), Kind: audit.AlertRoleChanged, EntryID: pulid.ID("ae_1"),
		OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID, ActorName: "Sam Patel",
	}}
	svc, auditor, realtime := newService(repo)
	actor := kernelports.AuditActor{UserID: pulid.ID("usr_9"), PrincipalID: pulid.ID("usr_9")}

	view, err := svc.SetReviewed(t.Context(), &SetReviewedRequest{
		TenantInfo: tenant, Actor: actor, ID: pulid.ID("arv_1"), Reviewed: true,
	})
	require.NoError(t, err)

	require.Len(t, repo.setReqs, 1)
	assert.Equal(t, pulid.ID("usr_9"), repo.setReqs[0].ReviewerID)
	assert.Equal(t, now, repo.setReqs[0].At)
	require.NotNil(t, view.ReviewedAt)
	assert.Equal(t, now, *view.ReviewedAt)
	assert.Equal(t, "Sam Patel changed a role", view.Title)

	require.Len(t, auditor.changes, 1)
	change := auditor.changes[0]
	assert.Equal(t, audit.ActionReviewReviewed, change.Action)
	assert.Equal(t, permission.ResourceSecurityAuditLog, change.Resource)
	assert.Equal(t, "arv_1", change.ResourceID)
	assert.Equal(t, actor, change.Actor)
	require.Len(t, realtime.published, 1)
	assert.Equal(t, pulid.ID("usr_9"), realtime.published[0].ActorUserID)
}

func TestMarkingAReviewedAlertAgainIsAConflictAndChangesNothing(t *testing.T) {
	t.Parallel()

	at := now - 60
	repo := &fakeRepo{stored: &audit.Review{ID: pulid.ID("arv_1"), ReviewedAt: &at}}
	svc, auditor, realtime := newService(repo)

	_, err := svc.SetReviewed(t.Context(), &SetReviewedRequest{
		TenantInfo: tenant, ID: pulid.ID("arv_1"), Reviewed: true,
	})

	var conflict *errortypes.ConflictError
	require.ErrorAs(t, err, &conflict)
	assert.Empty(t, repo.setReqs)
	assert.Empty(t, auditor.changes)
	assert.Empty(t, realtime.published)
}

func TestARaceLostToAnotherReviewerIsAConflict(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{
		stored: &audit.Review{ID: pulid.ID("arv_1")},
		setErr: auditports.ErrReviewUnchanged,
	}
	svc, auditor, _ := newService(repo)

	_, err := svc.SetReviewed(t.Context(), &SetReviewedRequest{
		TenantInfo: tenant, ID: pulid.ID("arv_1"), Reviewed: true,
	})

	var conflict *errortypes.ConflictError
	require.ErrorAs(t, err, &conflict)
	assert.Empty(t, auditor.changes)
}

func TestReopenClearsTheReviewAndIsAuditedAsAReopen(t *testing.T) {
	t.Parallel()

	at := now - 60
	repo := &fakeRepo{
		stored: &audit.Review{ID: pulid.ID("arv_1"), ReviewedAt: &at, ReviewedByID: "usr_2"},
	}
	svc, auditor, _ := newService(repo)

	view, err := svc.SetReviewed(t.Context(), &SetReviewedRequest{
		TenantInfo: tenant, ID: pulid.ID("arv_1"), Reviewed: false,
	})
	require.NoError(t, err)

	assert.Nil(t, view.ReviewedAt)
	require.Len(t, auditor.changes, 1)
	assert.Equal(t, audit.ActionReviewReopened, auditor.changes[0].Action)
}

func TestListReadsAWeekOfReviewedAlertsAndMarksOffHoursInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	ninePm := time.Date(2026, 11, 5, 21, 0, 0, 0, time.FixedZone("CST", -6*3600)).Unix()
	noon := time.Date(2026, 11, 5, 12, 0, 0, 0, time.FixedZone("CST", -6*3600)).Unix()
	repo := &fakeRepo{listResp: &auditports.ReviewList{
		Open: []*audit.Review{
			{ID: "arv_1", Kind: audit.AlertMalwareRefused, OccurredAt: ninePm},
			{ID: "arv_2", Kind: audit.AlertMalwareRefused, OccurredAt: noon},
		},
		OpenCount: 2,
	}}
	svc, _, _ := newService(repo)

	list, err := svc.List(t.Context(), &ListRequest{TenantInfo: tenant, Limit: 1_000})
	require.NoError(t, err)

	assert.Equal(t, MaxLimit, repo.listReq.Limit)
	assert.Equal(t, now-7*24*3600, repo.listReq.ReviewedSince)
	require.Len(t, list.Open, 2)
	assert.True(t, list.Open[0].OffHours)
	assert.False(t, list.Open[1].OffHours)
	assert.Equal(t, "Someone sent a file the malware scanner refused", list.Open[0].Title)
}

func TestTitlesSayWhoDidWhatForEveryKind(t *testing.T) {
	t.Parallel()

	cases := []struct {
		review audit.Review
		want   string
	}{
		{
			audit.Review{
				Kind:      audit.AlertRoleChanged,
				ActorName: "Sam Patel",
				EntityRef: "Marcus Lee",
			},
			"Sam Patel changed a role: Marcus Lee",
		},
		{
			audit.Review{
				Kind:      audit.AlertAPIKeyChanged,
				ActorName: "Sam Patel",
				EntityRef: "EDI 214 key",
			},
			"Sam Patel changed the API key EDI 214 key",
		},
		{
			audit.Review{Kind: audit.AlertMFAReset, ActorName: "Sam Patel"},
			"Sam Patel turned off or reset two-factor",
		},
		{
			audit.Review{Kind: audit.AlertIdentityConfigChanged, ActorName: "Sam Patel"},
			"Sam Patel changed sign-in settings",
		},
		{
			audit.Review{Kind: audit.AlertCrossTenantRefused, ActorName: "EDI 214 key"},
			"EDI 214 key reached for another organization's records",
		},
		{
			audit.Review{
				Kind:      audit.AlertLegalHoldChanged,
				ActorName: "Sam Patel",
				Action:    audit.ActionLegalHoldReleased,
			},
			"Sam Patel released a legal hold",
		},
		{
			audit.Review{
				Kind:      audit.AlertLegalHoldChanged,
				ActorName: "Sam Patel",
				Action:    audit.ActionLegalHoldPlaced,
			},
			"Sam Patel placed a legal hold",
		},
		{
			audit.Review{
				Kind:          audit.AlertBulkDelete,
				ActorName:     "Dana Whitfield",
				Count:         50,
				WindowMinutes: 10,
			},
			"Dana Whitfield deleted 50 records within 10 minutes",
		},
		{
			audit.Review{
				Kind:          audit.AlertBulkExport,
				ActorName:     "EDI 214 key",
				Count:         1,
				WindowMinutes: 1,
			},
			"EDI 214 key exported or downloaded 1 time within 1 minute",
		},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, Title(i18n.EN, &tc.review))
	}
}
