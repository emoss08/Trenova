package performancereviewservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/performancereviewservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRepo struct {
	repositories.PerformanceReviewRepository
	templates []*worker.PerformanceReviewTemplate
	reviews   []*worker.PerformanceReview
}

func (f *fakeRepo) GetTemplateByID(_ context.Context, req *repositories.GetReviewTemplateByIDRequest) (*worker.PerformanceReviewTemplate, error) {
	for _, t := range f.templates {
		if t.ID == req.ID {
			return t, nil
		}
	}
	return nil, errortypes.NewNotFoundError("PerformanceReviewTemplate not found")
}

func (f *fakeRepo) TemplateCodeExists(context.Context, *repositories.ReviewTemplateCodeExistsRequest) (bool, error) {
	return false, nil
}

func (f *fakeRepo) ClearDefaultTemplate(_ context.Context, _ pagination.TenantInfo, exceptID pulid.ID) error {
	for _, t := range f.templates {
		if t.ID != exceptID {
			t.IsDefault = false
		}
	}
	return nil
}

func (f *fakeRepo) CreateTemplate(_ context.Context, t *worker.PerformanceReviewTemplate) (*worker.PerformanceReviewTemplate, error) {
	t.ID = pulid.MustNew("prt_")
	f.templates = append(f.templates, t)
	return t, nil
}

func (f *fakeRepo) ListReviews(_ context.Context, req *repositories.ListPerformanceReviewsRequest) ([]*worker.PerformanceReview, error) {
	out := make([]*worker.PerformanceReview, 0, len(f.reviews))
	for _, r := range f.reviews {
		if r.WorkerID != req.WorkerID {
			continue
		}
		if len(req.Statuses) > 0 {
			match := false
			for _, status := range req.Statuses {
				if r.Status == status {
					match = true
				}
			}
			if !match {
				continue
			}
		}
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeRepo) GetReviewByID(_ context.Context, req *repositories.GetPerformanceReviewByIDRequest) (*worker.PerformanceReview, error) {
	for _, r := range f.reviews {
		if r.ID == req.ID {
			copied := *r
			if req.IncludeTemplate {
				for _, t := range f.templates {
					if t.ID == r.TemplateID {
						copied.Template = t
					}
				}
			}
			return &copied, nil
		}
	}
	return nil, errortypes.NewNotFoundError("PerformanceReview not found")
}

func (f *fakeRepo) CreateReview(_ context.Context, r *worker.PerformanceReview) (*worker.PerformanceReview, error) {
	for _, existing := range f.reviews {
		if existing.WorkerID == r.WorkerID && existing.TemplateID == r.TemplateID && existing.Status.IsOpen() {
			return nil, errortypes.NewValidationError("templateId", errortypes.ErrDuplicate, "open")
		}
	}
	r.ID = pulid.MustNew("prev_")
	f.reviews = append(f.reviews, r)
	return r, nil
}

func (f *fakeRepo) UpdateReview(_ context.Context, r *worker.PerformanceReview) (*worker.PerformanceReview, error) {
	for i, existing := range f.reviews {
		if existing.ID == r.ID {
			r.Version = existing.Version + 1
			f.reviews[i] = r
			return r, nil
		}
	}
	return nil, errortypes.NewNotFoundError("PerformanceReview not found")
}

func (f *fakeRepo) DeleteReview(_ context.Context, req *repositories.GetPerformanceReviewByIDRequest) error {
	for i, r := range f.reviews {
		if r.ID == req.ID {
			f.reviews = append(f.reviews[:i], f.reviews[i+1:]...)
			return nil
		}
	}
	return errortypes.NewNotFoundError("PerformanceReview not found")
}

type harness struct {
	svc      *performancereviewservice.Service
	repo     *fakeRepo
	tenant   pagination.TenantInfo
	wrk      *worker.Worker
	template *worker.PerformanceReviewTemplate
	userID   pulid.ID
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	wrk := &worker.Worker{ID: pulid.MustNew("wrk_"), OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID, Status: domaintypes.StatusActive}
	workerRepo := mocks.NewMockWorkerRepository(t)
	workerRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(wrk, nil).Maybe()
	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Maybe()

	cadence := int32(6)
	repo := &fakeRepo{templates: []*worker.PerformanceReviewTemplate{{
		ID: pulid.MustNew("prt_"), OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
		Code: "DRIVER-ANNUAL", Name: "Driver Review", Status: domaintypes.StatusActive, CadenceMonths: &cadence,
		Items: []worker.ReviewItem{
			{Key: "safety", Label: "Safety", Weight: 3},
			{Key: "service", Label: "Customer service", Weight: 1},
		},
	}}}
	svc := performancereviewservice.New(performancereviewservice.Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		WorkerRepo:   workerRepo,
		AuditService: audit,
	})
	return &harness{svc: svc, repo: repo, tenant: tenant, wrk: wrk, template: repo.templates[0], userID: pulid.MustNew("usr_")}
}

func TestReviewLifecycle(t *testing.T) {
	h := newHarness(t)
	now := timeutils.NowUnix()

	draft, err := h.svc.CreateReview(context.Background(), &performancereviewservice.CreateReviewRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, TemplateID: h.template.ID,
		PeriodStart: now - 180*86400, PeriodEnd: now, UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.ReviewStatusDraft, draft.Status)
	assert.Equal(t, h.userID, draft.ReviewerID)
	require.Len(t, draft.Ratings, 2, "items are copied from the template")
	assert.Contains(t, draft.Title, "Driver Review")

	_, err = h.svc.CreateReview(context.Background(), &performancereviewservice.CreateReviewRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, TemplateID: h.template.ID,
		PeriodStart: now - 10*86400, PeriodEnd: now, UserID: h.userID,
	})
	require.Error(t, err, "one open review per worker and template")

	_, err = h.svc.SubmitReview(context.Background(), &performancereviewservice.ReviewStatusRequest{
		ID: draft.ID, TenantInfo: h.tenant, UserID: h.userID,
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr, "unrated items block submission")

	five, three := int32(5), int32(3)
	saved, err := h.svc.UpdateReview(context.Background(), &performancereviewservice.UpdateReviewRequest{
		TenantInfo: h.tenant, ID: draft.ID,
		Ratings: []worker.ReviewRating{
			{Key: "safety", Score: &five, Comment: "Spotless"},
			{Key: "service", Score: &three},
			{Key: "bogus", Score: &five},
		},
		Summary: "Strong year on the road.",
		Goals:   []worker.ReviewGoal{{Title: "Complete hazmat refresher"}, {Title: "  "}},
		UserID:  h.userID,
	})
	require.NoError(t, err)
	require.Len(t, saved.Ratings, 2, "unknown keys are ignored")
	assert.Equal(t, "Spotless", saved.Ratings[0].Comment)
	require.True(t, saved.OverallScore.Valid)
	assert.Equal(t, "4.50", saved.OverallScore.Decimal.StringFixed(2))
	require.Len(t, saved.Goals, 1, "blank goals are dropped")
	assert.NotEmpty(t, saved.Goals[0].ID)
	assert.Equal(t, worker.ReviewGoalStatusOpen, saved.Goals[0].Status)

	submitted, err := h.svc.SubmitReview(context.Background(), &performancereviewservice.ReviewStatusRequest{
		ID: draft.ID, TenantInfo: h.tenant, UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.ReviewStatusSubmitted, submitted.Status)
	require.NotNil(t, submitted.SubmittedAt)

	_, err = h.svc.UpdateReview(context.Background(), &performancereviewservice.UpdateReviewRequest{
		TenantInfo: h.tenant, ID: draft.ID, UserID: h.userID,
	})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr, "submitted reviews are read-only")

	_, err = h.svc.AcknowledgeReview(context.Background(), h.tenant, draft.ID, pulid.MustNew("wrk_"), "")
	require.Error(t, err, "another worker cannot sign it")
	acked, err := h.svc.AcknowledgeReview(context.Background(), h.tenant, draft.ID, h.wrk.ID, "Thanks — agreed on the goal.")
	require.NoError(t, err)
	assert.Equal(t, worker.ReviewStatusAcknowledged, acked.Status)
	assert.True(t, acked.IsAcknowledged())

	closed, err := h.svc.CloseReview(context.Background(), &performancereviewservice.ReviewStatusRequest{
		ID: draft.ID, TenantInfo: h.tenant, UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.ReviewStatusClosed, closed.Status)
	require.NotNil(t, closed.NextReviewAt)
	assert.Equal(t, timeutils.AddMonthsUTC(closed.PeriodEnd, 6), *closed.NextReviewAt, "next review comes from the cadence")

	visible, err := h.svc.ListReviews(context.Background(), h.tenant, h.wrk.ID, []worker.ReviewStatus{worker.ReviewStatusSubmitted, worker.ReviewStatusClosed})
	require.NoError(t, err)
	assert.Len(t, visible, 1)
}

func TestReopenAndDelete(t *testing.T) {
	h := newHarness(t)
	now := timeutils.NowUnix()
	draft, err := h.svc.CreateReview(context.Background(), &performancereviewservice.CreateReviewRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, TemplateID: h.template.ID,
		Title: "Probation review", PeriodStart: now - 90*86400, PeriodEnd: now, UserID: h.userID,
	})
	require.NoError(t, err)

	four := int32(4)
	_, err = h.svc.UpdateReview(context.Background(), &performancereviewservice.UpdateReviewRequest{
		TenantInfo: h.tenant, ID: draft.ID, Summary: "Solid start.",
		Ratings: []worker.ReviewRating{{Key: "safety", Score: &four}, {Key: "service", Score: &four}},
		UserID:  h.userID,
	})
	require.NoError(t, err)
	_, err = h.svc.SubmitReview(context.Background(), &performancereviewservice.ReviewStatusRequest{ID: draft.ID, TenantInfo: h.tenant, UserID: h.userID})
	require.NoError(t, err)

	reopened, err := h.svc.ReopenReview(context.Background(), &performancereviewservice.ReviewStatusRequest{ID: draft.ID, TenantInfo: h.tenant, UserID: h.userID})
	require.NoError(t, err)
	assert.Equal(t, worker.ReviewStatusDraft, reopened.Status)
	assert.Nil(t, reopened.SubmittedAt)

	require.NoError(t, h.svc.DeleteReview(context.Background(), h.tenant, draft.ID, h.userID))
}

func TestCreateTemplate_DefaultSwapsAndValidates(t *testing.T) {
	h := newHarness(t)
	h.template.IsDefault = true

	_, err := h.svc.CreateTemplate(context.Background(), &worker.PerformanceReviewTemplate{
		OrganizationID: h.tenant.OrgID, BusinessUnitID: h.tenant.BuID, Code: "empty", Name: "Empty",
		Status: domaintypes.StatusActive,
	}, h.userID)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr, "a template needs items")

	created, err := h.svc.CreateTemplate(context.Background(), &worker.PerformanceReviewTemplate{
		OrganizationID: h.tenant.OrgID, BusinessUnitID: h.tenant.BuID, Code: "otr-quarterly", Name: "OTR Quarterly",
		Status: domaintypes.StatusActive, IsDefault: true,
		Items: []worker.ReviewItem{{Key: "safety", Label: "Safety", Weight: 1}},
	}, h.userID)
	require.NoError(t, err)
	assert.Equal(t, "OTR-QUARTERLY", created.Code)
	assert.True(t, created.IsDefault)
	assert.False(t, h.template.IsDefault, "only one default at a time")
}
