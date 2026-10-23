package ratematrixservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulatemplateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/ratematrixportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type testDBConnection struct{}

func (testDBConnection) DB() *bun.DB                          { return nil }
func (testDBConnection) DBForContext(context.Context) bun.IDB { return nil }
func (testDBConnection) HealthCheck(context.Context) error    { return nil }
func (testDBConnection) IsHealthy(context.Context) bool       { return true }
func (testDBConnection) Close() error                         { return nil }
func (testDBConnection) WithTx(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

type testDeps struct {
	repo        *ratematrixportsmock.MockRateMatrixRepository
	versionRepo *ratematrixportsmock.MockRateMatrixVersionRepository
	reviewRepo  *ratematrixportsmock.MockRateMatrixReviewRepository
	usages      *formulatemplateportsmock.MockFormulaTemplateVersionRepository
	audit       *auditportsmock.MockAuditService
	svc         *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()

	deps := &testDeps{
		repo:        ratematrixportsmock.NewMockRateMatrixRepository(t),
		versionRepo: ratematrixportsmock.NewMockRateMatrixVersionRepository(t),
		reviewRepo:  ratematrixportsmock.NewMockRateMatrixReviewRepository(t),
		usages:      formulatemplateportsmock.NewMockFormulaTemplateVersionRepository(t),
		audit:       auditportsmock.NewMockAuditService(t),
	}
	deps.svc = &Service{
		l:            zap.NewNop(),
		db:           testDBConnection{},
		repo:         deps.repo,
		versionRepo:  deps.versionRepo,
		reviewRepo:   deps.reviewRepo,
		usages:       deps.usages,
		validator:    NewTestValidator(),
		auditService: deps.audit,
	}
	deps.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Maybe()
	return deps
}

func tenantFor(userID pulid.ID) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: userID,
	}
}

func zoneMatrix(tenant pagination.TenantInfo) *ratematrix.RateMatrix {
	id := pulid.MustNew("rmx_")
	return &ratematrix.RateMatrix{
		ID:             id,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Code:           "ZONES",
		Name:           "Zones",
		ValueType:      ratematrix.ValueTypeMoney,
		Status:         formulatemplate.StatusDraft,
		DraftEditorIDs: []pulid.ID{},
		Dimensions: []*ratematrix.RateMatrixDimension{{
			RateMatrixID: id,
			Position:     0,
			Kind:         ratematrix.DimensionKindZone,
			MatchMode:    ratematrix.MatchModeExact,
		}},
	}
}

func expectLoad(deps *testDeps, matrix *ratematrix.RateMatrix) {
	deps.repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *ratematrixports.GetRateMatrixByIDRequest) bool {
			return req.RateMatrixID == matrix.ID
		})).
		RunAndReturn(func(
			context.Context,
			*ratematrixports.GetRateMatrixByIDRequest,
		) (*ratematrix.RateMatrix, error) {
			loaded := *matrix
			return &loaded, nil
		})
}

func expectHeaderSave(deps *testDeps) {
	deps.repo.EXPECT().
		UpdateHeader(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			entity *ratematrix.RateMatrix,
		) (*ratematrix.RateMatrix, error) {
			entity.Version++
			return entity, nil
		})
}

func expectReviews(deps *testDeps) {
	deps.reviewRepo.EXPECT().Latest(mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	deps.reviewRepo.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			review *ratematrix.RateMatrixReview,
		) (*ratematrix.RateMatrixReview, error) {
			return review, nil
		}).Maybe()
}

func TestCreateStartsADraftWithNoPublishedVersion(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	author := pulid.MustNew("usr_")
	tenant := tenantFor(author)
	entity := zoneMatrix(tenant)
	entity.Status = formulatemplate.StatusActive
	entity.PublishedVersionNumber = 7

	deps.repo.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			created *ratematrix.RateMatrix,
		) (*ratematrix.RateMatrix, error) {
			return created, nil
		})

	created, err := deps.svc.Create(t.Context(), entity, author)
	require.NoError(t, err)
	assert.Equal(t, formulatemplate.StatusDraft, created.Status)
	assert.Zero(t, created.PublishedVersionNumber)
	assert.True(t, created.HasDraftChanges)
	assert.Equal(t, []pulid.ID{author}, created.DraftEditorIDs)
}

func TestReplacingCellsOfAnApprovedMatrixReturnsItToDraft(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	editor := pulid.MustNew("usr_")
	tenant := tenantFor(editor)
	matrix := zoneMatrix(tenant)
	matrix.Status = formulatemplate.StatusActive
	matrix.PublishedVersionNumber = 2
	matrix.HasDraftChanges = false
	approver := pulid.MustNew("usr_")
	matrix.ApprovedByID = &approver

	expectLoad(deps, matrix)
	deps.repo.EXPECT().ReplaceCells(mock.Anything, mock.Anything).Return(nil)
	var saved *ratematrix.RateMatrix
	deps.repo.EXPECT().UpdateHeader(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			entity *ratematrix.RateMatrix,
		) (*ratematrix.RateMatrix, error) {
			saved = entity
			return entity, nil
		})

	err := deps.svc.ReplaceCells(t.Context(), &ratematrixports.ReplaceRateMatrixCellsRequest{
		TenantInfo:   tenant,
		RateMatrixID: matrix.ID,
		Cells: []*ratematrix.RateMatrixCell{
			{D0Key: "SE", Value: decimal.RequireFromString("2.35")},
			nil,
			{D0Key: "NE", Value: decimal.RequireFromString("2.60")},
		},
	}, editor)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, formulatemplate.StatusDraft, saved.Status)
	assert.True(t, saved.HasDraftChanges)
	assert.EqualValues(t, 2, saved.DraftCellCount)
	assert.Nil(t, saved.ApprovedByID)
	assert.EqualValues(t, 2, saved.PublishedVersionNumber, "the published version stays served")
	assert.Contains(t, saved.DraftEditorIDs, editor)
}

func TestReplacingCellsRefusesOverlappingPostalRanges(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	tenant := tenantFor(pulid.MustNew("usr_"))
	matrix := zoneMatrix(tenant)
	matrix.Dimensions[0].Kind = ratematrix.DimensionKindZipRange
	matrix.Dimensions[0].MatchMode = ratematrix.MatchModeRange
	expectLoad(deps, matrix)

	err := deps.svc.ReplaceCells(t.Context(), &ratematrixports.ReplaceRateMatrixCellsRequest{
		TenantInfo:   tenant,
		RateMatrixID: matrix.ID,
		Cells: []*ratematrix.RateMatrixCell{
			{D0Key: "30000", D0KeyTo: "31999", Value: decimal.NewFromInt(1)},
			{D0Key: "31500", D0KeyTo: "32999", Value: decimal.NewFromInt(2)},
		},
	}, tenant.UserID)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "cells[1].d0Key", multiErr.Errors[0].Field)
}

func submitted(matrix *ratematrix.RateMatrix, submitter pulid.ID) *ratematrix.RateMatrix {
	now := timeutils.NowUnix()
	matrix.Status = formulatemplate.StatusInReview
	matrix.SubmittedByID = &submitter
	matrix.SubmittedAt = &now
	matrix.HasDraftChanges = true
	matrix.DraftCellCount = 3
	matrix.DraftEditorIDs = []pulid.ID{submitter}
	return matrix
}

func TestApprovePublishesTheDraftAsTheNextVersion(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	submitter := pulid.MustNew("usr_")
	approver := pulid.MustNew("usr_")
	tenant := tenantFor(approver)
	matrix := submitted(zoneMatrix(tenant), submitter)
	matrix.PublishedVersionNumber = 1

	expectLoad(deps, matrix)
	expectHeaderSave(deps)
	expectReviews(deps)
	deps.repo.EXPECT().CountMisfitCells(mock.Anything, mock.Anything).Return(int64(0), nil)
	var published *ratematrix.RateMatrixVersion
	deps.versionRepo.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			version *ratematrix.RateMatrixVersion,
		) (*ratematrix.RateMatrixVersion, error) {
			published = version
			return version, nil
		})
	deps.repo.EXPECT().
		CopyVersion(mock.Anything, mock.MatchedBy(func(req *ratematrixports.CopyVersionRequest) bool {
			return req.From == ratematrix.DraftVersion && req.To == 2
		})).
		Return(int64(3), nil)

	approved, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: tenant,
		EntityID:   matrix.ID,
		Comment:    "New fuel season",
	})
	require.NoError(t, err)
	assert.Equal(t, formulatemplate.StatusActive, approved.Status)
	assert.EqualValues(t, 2, approved.PublishedVersionNumber)
	assert.False(t, approved.HasDraftChanges)
	assert.Empty(t, approved.DraftEditorIDs)
	require.NotNil(t, published)
	assert.EqualValues(t, 2, published.VersionNumber)
	assert.Equal(t, "ZONES", published.Code)
	assert.EqualValues(t, 3, published.CellCount)
	assert.Equal(t, approver, *published.PublishedByID)
}

func TestNobodyPublishesADraftTheyWroteOrSubmitted(t *testing.T) {
	t.Parallel()

	for name, actor := range map[string]func(submitter, editor pulid.ID) pulid.ID{
		"the submitter": func(submitter, _ pulid.ID) pulid.ID { return submitter },
		"an editor":     func(_, editor pulid.ID) pulid.ID { return editor },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			deps := setupTest(t)
			submitter := pulid.MustNew("usr_")
			editor := pulid.MustNew("usr_")
			tenant := tenantFor(actor(submitter, editor))
			matrix := submitted(zoneMatrix(tenant), submitter)
			matrix.DraftEditorIDs = []pulid.ID{submitter, editor}
			expectLoad(deps, matrix)

			_, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
				TenantInfo: tenant,
				EntityID:   matrix.ID,
			})
			var validation *errortypes.Error
			require.ErrorAs(t, err, &validation)
			assert.Equal(t, "approvedById", validation.Field)
		})
	}
}

func TestApproveOwnLetsTheSubmitterAndEditorPublishTheirOwnDraft(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	author := pulid.MustNew("usr_")
	tenant := tenantFor(author)
	matrix := submitted(zoneMatrix(tenant), author)
	matrix.DraftEditorIDs = []pulid.ID{author}
	matrix.PublishedVersionNumber = 1

	expectLoad(deps, matrix)
	expectHeaderSave(deps)
	expectReviews(deps)
	deps.repo.EXPECT().CountMisfitCells(mock.Anything, mock.Anything).Return(int64(0), nil)
	deps.versionRepo.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			version *ratematrix.RateMatrixVersion,
		) (*ratematrix.RateMatrixVersion, error) {
			return version, nil
		})
	deps.repo.EXPECT().CopyVersion(mock.Anything, mock.Anything).Return(int64(3), nil)

	approved, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo:    tenant,
		EntityID:      matrix.ID,
		MayApproveOwn: true,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 2, approved.PublishedVersionNumber)
}

func TestSubmitRefusesADraftThatMatchesThePublishedVersion(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	tenant := tenantFor(pulid.MustNew("usr_"))
	matrix := zoneMatrix(tenant)
	matrix.PublishedVersionNumber = 3
	matrix.HasDraftChanges = false
	expectLoad(deps, matrix)

	_, err := deps.svc.Submit(t.Context(), &SubmitRequest{
		ApprovalActionRequest: ApprovalActionRequest{TenantInfo: tenant, EntityID: matrix.ID},
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "hasDraftChanges", multiErr.Errors[0].Field)
}

func TestSubmitRefusesCellsThatNoLongerFitTheAxes(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	tenant := tenantFor(pulid.MustNew("usr_"))
	matrix := zoneMatrix(tenant)
	matrix.HasDraftChanges = true
	matrix.DraftCellCount = 10
	expectLoad(deps, matrix)
	deps.repo.EXPECT().CountMisfitCells(mock.Anything, mock.Anything).Return(int64(4), nil)

	_, err := deps.svc.Submit(t.Context(), &SubmitRequest{
		ApprovalActionRequest: ApprovalActionRequest{TenantInfo: tenant, EntityID: matrix.ID},
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "cells", multiErr.Errors[0].Field)
}

func TestUpdateRefusesANewCodeOncePublished(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	tenant := tenantFor(pulid.MustNew("usr_"))
	matrix := zoneMatrix(tenant)
	matrix.PublishedVersionNumber = 1
	expectLoad(deps, matrix)

	payload := *matrix
	payload.Code = "ZONES_2026"
	_, err := deps.svc.Update(t.Context(), &payload, tenant.UserID)

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "code", validation.Field)
}

func TestDeleteRefusesAPublishedMatrix(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	tenant := tenantFor(pulid.MustNew("usr_"))
	matrix := zoneMatrix(tenant)
	matrix.PublishedVersionNumber = 1
	expectLoad(deps, matrix)

	err := deps.svc.Delete(t.Context(), &ratematrixports.GetRateMatrixByIDRequest{
		RateMatrixID: matrix.ID,
		TenantInfo:   tenant,
	}, tenant.UserID)
	require.Error(t, err)
}

func TestArchiveAsksBeforeArchivingAPinnedMatrix(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	tenant := tenantFor(pulid.MustNew("usr_"))
	matrix := zoneMatrix(tenant)
	matrix.Status = formulatemplate.StatusActive
	matrix.PublishedVersionNumber = 2
	expectLoad(deps, matrix)
	deps.usages.EXPECT().ListMatrixUsages(mock.Anything, mock.Anything).
		Return([]*formulatemplateports.MatrixUsage{{MatrixVersionNumber: 2}}, nil)

	_, err := deps.svc.Archive(t.Context(), &ArchiveRequest{
		TenantInfo:   tenant,
		RateMatrixID: matrix.ID,
	})
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "confirmInUse", validation.Field)

	expectHeaderSave(deps)
	archived, err := deps.svc.Archive(t.Context(), &ArchiveRequest{
		TenantInfo:   tenant,
		RateMatrixID: matrix.ID,
		ConfirmInUse: true,
	})
	require.NoError(t, err)
	assert.Equal(t, formulatemplate.StatusInactive, archived.Status)
}

func TestRestoringThePublishedVersionDiscardsTheDraft(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	editor := pulid.MustNew("usr_")
	tenant := tenantFor(editor)
	matrix := zoneMatrix(tenant)
	matrix.PublishedVersionNumber = 4
	matrix.HasDraftChanges = true
	matrix.DraftEditorIDs = []pulid.ID{editor}
	expectLoad(deps, matrix)
	deps.versionRepo.EXPECT().Get(mock.Anything, mock.Anything).Return(&ratematrix.RateMatrixVersion{
		RateMatrixID:  matrix.ID,
		VersionNumber: 4,
		ValueType:     ratematrix.ValueTypePercent,
	}, nil)
	deps.repo.EXPECT().CopyVersion(mock.Anything, mock.MatchedBy(func(req *ratematrixports.CopyVersionRequest) bool {
		return req.From == 4 && req.To == ratematrix.DraftVersion
	})).Return(int64(12), nil)
	var saved *ratematrix.RateMatrix
	deps.repo.EXPECT().UpdateHeader(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			entity *ratematrix.RateMatrix,
		) (*ratematrix.RateMatrix, error) {
			saved = entity
			return entity, nil
		})

	_, err := deps.svc.RestoreVersion(t.Context(), &RestoreVersionRequest{
		TenantInfo:    tenant,
		RateMatrixID:  matrix.ID,
		VersionNumber: 4,
	})
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.False(t, saved.HasDraftChanges)
	assert.Equal(t, formulatemplate.StatusActive, saved.Status)
	assert.EqualValues(t, 12, saved.DraftCellCount)
	assert.Equal(t, ratematrix.ValueTypePercent, saved.ValueType)
	assert.Empty(t, saved.DraftEditorIDs)
}
