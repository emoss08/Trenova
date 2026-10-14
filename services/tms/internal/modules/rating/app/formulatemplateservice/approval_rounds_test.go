package formulatemplateservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	notificationdomain "github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// stubReviewRepo remembers what the service recorded and answers the latest
// entry it was primed with.
type stubReviewRepo struct {
	latest  *formulatemplate.Review
	stale   []*formulatemplate.FormulaTemplate
	created []*formulatemplate.Review
}

func (s *stubReviewRepo) Create(
	_ context.Context,
	entity *formulatemplate.Review,
) (*formulatemplate.Review, error) {
	s.created = append(s.created, entity)
	return entity, nil
}

func (s *stubReviewRepo) ListByTemplate(
	context.Context,
	*formulatemplateports.ListTemplateReviewsRequest,
) ([]*formulatemplate.Review, error) {
	return s.created, nil
}

func (s *stubReviewRepo) Latest(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*formulatemplate.Review, error) {
	return s.latest, nil
}

func (s *stubReviewRepo) ListStaleSubmissions(
	context.Context,
	*formulatemplateports.ListStaleSubmissionsRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return s.stale, nil
}

func primeReviews(deps *testDeps, latest *formulatemplate.Review) *stubReviewRepo {
	repo := &stubReviewRepo{latest: latest}
	deps.svc.reviewRepo = repo
	return repo
}

func TestSubmit_OpensARoundAgainstTheApprovedBase(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	reviews := primeReviews(deps, nil)

	template := newTestTemplate()
	deps.testCaseRepo.cases = append(deps.testCaseRepo.cases, newTestCase("exact match", 250, 0.01))
	template.Status = formulatemplate.StatusDraft

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{VersionNumber: 3}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Submit(
		t.Context(),
		&SubmitRequest{ApprovalActionRequest: ApprovalActionRequest{
			TenantInfo: newTenantInfo(),
			EntityID:   template.ID,
			Comment:    "please review",
		}},
	)
	require.NoError(t, err)

	require.Len(t, reviews.created, 1)
	entry := reviews.created[0]
	assert.Equal(t, formulatemplate.ReviewDecisionSubmitted, entry.Decision)
	assert.EqualValues(t, 1, entry.Round)
	assert.EqualValues(t, 3, entry.BaseVersionNumber, "the diff a reviewer sees is against v3")
	assert.Equal(t, "please review", entry.Comment)
	require.NotNil(t, entry.ActorID)
}

func TestSubmit_AfterChangesRequestedStaysInTheSameRound(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	reviews := primeReviews(deps, &formulatemplate.Review{
		Round: 2, Decision: formulatemplate.ReviewDecisionChangesRequested,
	})

	template := newTestTemplate()
	deps.testCaseRepo.cases = append(deps.testCaseRepo.cases, newTestCase("exact match", 250, 0.01))
	template.Status = formulatemplate.StatusDraft

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).Return(nil, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Submit(
		t.Context(),
		&SubmitRequest{ApprovalActionRequest: ApprovalActionRequest{
			TenantInfo: newTenantInfo(),
			EntityID:   template.ID,
		}},
	)
	require.NoError(t, err)

	require.Len(t, reviews.created, 1)
	assert.EqualValues(t, 2, reviews.created[0].Round, "the conversation continues")
	assert.EqualValues(t, 0, reviews.created[0].BaseVersionNumber, "never approved yet")
}

func TestRequestChanges_KeepsTheSubmitterAndRecordsTheRound(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	captured := withNotifications(t, deps)
	reviews := primeReviews(deps, &formulatemplate.Review{
		Round: 1, Decision: formulatemplate.ReviewDecisionSubmitted, BaseVersionNumber: 2,
	})

	submitterID := pulid.MustNew("usr_")
	submittedAt := timeutils.NowUnix() - 3600
	template := newTestTemplate()
	template.Status = formulatemplate.StatusInReview
	template.SubmittedByID = &submitterID
	template.SubmittedAt = &submittedAt

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.RequestChanges(t.Context(), &ApprovalActionRequest{
		TenantInfo: newTenantInfo(),
		EntityID:   template.ID,
		Comment:    "guard the weight",
	})
	require.NoError(t, err)

	assert.Equal(t, formulatemplate.StatusDraft, result.Status)
	require.NotNil(t, result.SubmittedByID, "unlike a rejection, the author keeps the floor")
	assert.Equal(t, submitterID, *result.SubmittedByID)
	assert.Equal(t, "guard the weight", result.ReviewComment)

	require.Len(t, reviews.created, 1)
	assert.Equal(t, formulatemplate.ReviewDecisionChangesRequested, reviews.created[0].Decision)
	assert.EqualValues(t, 1, reviews.created[0].Round)
	assert.EqualValues(t, 2, reviews.created[0].BaseVersionNumber, "the round keeps its base")

	notified := *captured
	require.NotNil(t, notified)
	assert.Equal(t, eventTemplateChangesRequested, notified.EventType)
	assert.Equal(t, notificationdomain.ChannelUser, notified.Channel)
	assert.Contains(t, notified.Message, "guard the weight")
}

func TestRequestChanges_RequiresAComment(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)

	_, err := deps.svc.RequestChanges(t.Context(), &ApprovalActionRequest{
		TenantInfo: newTenantInfo(),
		EntityID:   pulid.MustNew("ft_"),
	})
	require.Error(t, err)
}

func TestApprove_RefusesAStaleSubmission(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)

	submitterID := pulid.MustNew("usr_")
	submittedAt := timeutils.NowUnix() - formulatemplate.SubmissionExpiry - 60
	template := newTestTemplate()
	template.Status = formulatemplate.StatusInReview
	template.SubmittedByID = &submitterID
	template.SubmittedAt = &submittedAt

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	deps.versionRepo.On("ListAuthorIDsSince", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	_, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: newTenantInfo(),
		EntityID:   template.ID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "resubmit")
	assert.Equal(t, formulatemplate.StatusInReview, template.Status, "nothing was saved")
}

func TestExpireStaleSubmissions_ReturnsTemplatesToDraftAndRecordsIt(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	captured := withNotifications(t, deps)

	now := timeutils.NowUnix()
	submitterID := pulid.MustNew("usr_")
	oldSubmission := now - formulatemplate.SubmissionExpiry - 3600
	stale := newTestTemplate()
	stale.Status = formulatemplate.StatusInReview
	stale.SubmittedByID = &submitterID
	stale.SubmittedAt = &oldSubmission

	freshSubmission := now - 60
	fresh := newTestTemplate()
	fresh.Status = formulatemplate.StatusInReview
	fresh.SubmittedAt = &freshSubmission

	reviews := primeReviews(deps, &formulatemplate.Review{
		Round: 4, Decision: formulatemplate.ReviewDecisionSubmitted, BaseVersionNumber: 1,
	})
	reviews.stale = []*formulatemplate.FormulaTemplate{stale, fresh}

	deps.repo.On("Update", mock.Anything, mock.MatchedBy(func(t *formulatemplate.FormulaTemplate) bool {
		return t.ID == stale.ID
	})).
		Return(stale, nil).
		Once()
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.ExpireStaleSubmissions(
		t.Context(),
		&ExpireStaleSubmissionsRequest{Now: now},
	)
	require.NoError(t, err)

	assert.Equal(
		t,
		[]pulid.ID{stale.ID},
		result.Expired,
		"a fresh submission the repo returned by mistake is left alone",
	)
	assert.Equal(t, formulatemplate.StatusDraft, stale.Status)
	assert.Nil(t, stale.SubmittedByID)
	assert.Nil(t, stale.SubmittedAt)

	require.Len(t, reviews.created, 1)
	assert.Equal(t, formulatemplate.ReviewDecisionExpired, reviews.created[0].Decision)
	assert.EqualValues(t, 4, reviews.created[0].Round)
	assert.Nil(t, reviews.created[0].ActorID, "nobody decided; time did")

	notified := *captured
	require.NotNil(t, notified)
	assert.Equal(t, eventTemplateSubmissionExpired, notified.EventType)
	require.NotNil(t, notified.TargetUserID)
	assert.Equal(t, submitterID, *notified.TargetUserID)
}

func TestSubmissionAgeCheck(t *testing.T) {
	t.Parallel()

	now := timeutils.NowUnix()
	fresh := now - 24*3600
	aging := now - 8*24*3600
	stale := now - formulatemplate.SubmissionExpiry - 1

	inReview := func(at *int64) *formulatemplate.FormulaTemplate {
		template := newTestTemplate()
		template.Status = formulatemplate.StatusInReview
		template.SubmittedAt = at
		return template
	}

	assert.Equal(t, ReadinessPass, submissionAgeCheck(inReview(&fresh), now).Status)
	assert.Equal(t, ReadinessWarn, submissionAgeCheck(inReview(&aging), now).Status)
	assert.Equal(t, ReadinessFail, submissionAgeCheck(inReview(&stale), now).Status)

	draft := newTestTemplate()
	draft.Status = formulatemplate.StatusDraft
	assert.Equal(t, ReadinessPass, submissionAgeCheck(draft, now).Status,
		"age only matters while a decision is pending")
}

func approvableTemplate(deps *testDeps) *formulatemplate.FormulaTemplate {
	submitterID := pulid.MustNew("usr_")
	submittedAt := timeutils.NowUnix() - 3600
	template := newTestTemplate()
	template.Status = formulatemplate.StatusInReview
	template.SubmittedByID = &submitterID
	template.SubmittedAt = &submittedAt
	deps.testCaseRepo.cases = append(deps.testCaseRepo.cases, newTestCase("exact match", 250, 0.01))
	return template
}

func TestApprove_TheApprovedSnapshotTakesEffectWhenApproved(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)
	template := approvableTemplate(deps)

	var snapshot *formulatemplate.FormulaTemplateVersion
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			snapshot = args.Get(1).(*formulatemplate.FormulaTemplateVersion)
		}).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	before := timeutils.NowUnix()
	deps.versionRepo.On("ListAuthorIDsSince", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	_, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: newTenantInfo(),
		EntityID:   template.ID,
	})
	require.NoError(t, err)

	require.NotNil(t, snapshot)
	assert.Equal(t, formulatemplate.StatusActive, snapshot.Status)
	require.NotNil(t, snapshot.EffectiveFrom, "approval closes the previous window")
	assert.GreaterOrEqual(t, *snapshot.EffectiveFrom, before)
}

func TestApproveFrom_AChosenDateStartsTheWindowThen(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)
	template := approvableTemplate(deps)

	var snapshot *formulatemplate.FormulaTemplateVersion
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			snapshot = args.Get(1).(*formulatemplate.FormulaTemplateVersion)
		}).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	firstOfNextMonth := timeutils.NowUnix() + 30*86_400
	deps.versionRepo.On("ListAuthorIDsSince", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	_, err := deps.svc.ApproveFrom(t.Context(), &ApproveRequest{
		ApprovalActionRequest: ApprovalActionRequest{
			TenantInfo: newTenantInfo(),
			EntityID:   template.ID,
		},
		EffectiveFrom: &firstOfNextMonth,
	})
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.NotNil(t, snapshot.EffectiveFrom)
	assert.Equal(t, firstOfNextMonth, *snapshot.EffectiveFrom)
}

func TestApproveFrom_RefusesADateInThePast(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)

	lastWeek := timeutils.NowUnix() - 7*86_400
	deps.versionRepo.On("ListAuthorIDsSince", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	_, err := deps.svc.ApproveFrom(t.Context(), &ApproveRequest{
		ApprovalActionRequest: ApprovalActionRequest{
			TenantInfo: newTenantInfo(),
			EntityID:   pulid.MustNew("ft_"),
		},
		EffectiveFrom: &lastWeek,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "past")
	deps.repo.AssertNotCalled(t, "Update")
}

func TestApprove_RefusesAnyoneWhoAuthoredTheRevision(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, &formulatemplate.Review{
		Round: 2, Decision: formulatemplate.ReviewDecisionSubmitted, BaseVersionNumber: 4,
	})
	template := approvableTemplate(deps)
	approver := newTenantInfo()

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("ListAuthorIDsSince", mock.Anything, mock.MatchedBy(
		func(req *formulatemplateports.ListVersionAuthorsRequest) bool {
			return req.TemplateID == template.ID && req.AfterVersion == 4
		},
	)).Return([]pulid.ID{pulid.MustNew("usr_"), approver.UserID}, nil)

	_, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: approver,
		EntityID:   template.ID,
	})
	require.Error(t, err, "the approver edited the revision even though someone else submitted it")
	assert.Contains(t, err.Error(), "authored")
	deps.repo.AssertNotCalled(t, "Update")
}
