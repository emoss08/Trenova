//go:build integration

package documentrepository_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type testRepository struct {
	db *bun.DB
	l  *zap.Logger
}

func newTestRepository(db *bun.DB) *testRepository {
	return &testRepository{
		db: db,
		l:  zap.NewNop(),
	}
}

func (r *testRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDocumentsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"doc",
		req.Filter,
		(*document.Document)(nil),
	)

	if req.ResourceID != "" {
		q = q.Where("doc.resource_id = ?", req.ResourceID)
	}

	if req.ResourceType != "" {
		q = q.Where("doc.resource_type = ?", req.ResourceType)
	}

	if req.Status != "" {
		q = q.Where("doc.status = ?", req.Status)
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *testRepository) List(
	ctx context.Context,
	req *repositories.ListDocumentsRequest,
) (*pagination.ListResult[*document.Document], error) {
	entities := make([]*document.Document, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*document.Document]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *testRepository) GetByID(
	ctx context.Context,
	req repositories.GetDocumentByIDRequest,
) (*document.Document, error) {
	entity := new(document.Document)
	err := r.db.
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("doc.id = ?", req.ID).
				Where("doc.organization_id = ?", req.TenantInfo.OrgID).
				Where("doc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Document")
	}

	return entity, nil
}

func (r *testRepository) GetByResourceID(
	ctx context.Context,
	req repositories.GetDocumentsByResourceRequest,
) ([]*document.Document, error) {
	entities := make([]*document.Document, 0)
	err := r.db.
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("doc.resource_id = ?", req.ResourceID).
				Where("doc.resource_type = ?", req.ResourceType).
				Where("doc.organization_id = ?", req.TenantInfo.OrgID).
				Where("doc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Order("doc.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *testRepository) Create(
	ctx context.Context,
	entity *document.Document,
) (*document.Document, error) {
	if _, err := r.db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *testRepository) Update(
	ctx context.Context,
	entity *document.Document,
) (*document.Document, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.
		NewUpdate().
		Model(entity).WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Document", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *testRepository) GetByIDs(
	ctx context.Context,
	req repositories.BulkDeleteDocumentRequest,
) ([]*document.Document, error) {
	entities := make([]*document.Document, 0, len(req.IDs))
	err := r.db.
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("doc.id IN (?)", bun.List(req.IDs)).
				Where("doc.organization_id = ?", req.TenantInfo.OrgID).
				Where("doc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *testRepository) Delete(
	ctx context.Context,
	req repositories.DeleteDocumentRequest,
) error {
	results, err := r.db.
		NewDelete().
		Model((*document.Document)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("id = ?", req.ID).
				Where("organization_id = ?", req.TenantInfo.OrgID).
				Where("business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Exec(ctx)
	if err != nil {
		return err
	}

	rowsAffected, _ := results.RowsAffected()
	if rowsAffected == 0 {
		return errortypes.NewNotFoundError("Document not found within your organization")
	}

	return nil
}

func (r *testRepository) BulkDelete(
	ctx context.Context,
	req repositories.BulkDeleteDocumentRequest,
) error {
	results, err := r.db.
		NewDelete().
		Model((*document.Document)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("id IN (?)", bun.List(req.IDs)).
				Where("organization_id = ?", req.TenantInfo.OrgID).
				Where("business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Exec(ctx)
	if err != nil {
		return err
	}

	rowsAffected, _ := results.RowsAffected()
	if rowsAffected == 0 {
		return dberror.HandleNotFoundError(nil, "Document")
	}

	return nil
}

type testFixtures struct {
	orgID  pulid.ID
	buID   pulid.ID
	userID pulid.ID
}

func newTestFixtures(t *testing.T, ctx context.Context, db *bun.DB) *testFixtures {
	t.Helper()

	data := seedtest.SeedFullTestData(t, ctx, db)

	return &testFixtures{
		orgID:  data.Organization.ID,
		buID:   data.BusinessUnit.ID,
		userID: data.User.ID,
	}
}

func createTestDocument(
	fixtures *testFixtures,
	opts ...func(*document.Document),
) *document.Document {
	now := timeutils.NowUnix()
	doc := &document.Document{
		ID:             pulid.MustNew("doc_"),
		OrganizationID: fixtures.orgID,
		BusinessUnitID: fixtures.buID,
		FileName:       "test-file.pdf",
		OriginalName:   "Original Test File.pdf",
		FileSize:       1024,
		FileType:       "application/pdf",
		StoragePath:    fixtures.orgID.String() + "/trailer/test-file.pdf",
		Status:         document.StatusActive,
		Description:    "Test document description",
		ResourceID:     pulid.MustNew("tr_").String(),
		ResourceType:   "trailer",
		Tags:           []string{"test", "document"},
		UploadedByID:   fixtures.userID,
		Version:        0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	for _, opt := range opts {
		opt(doc)
	}

	return doc
}

func TestDocumentRepository_Create_Integration(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	fixtures := newTestFixtures(t, ctx, db)

	repo := newTestRepository(db)

	t.Run("create document successfully", func(t *testing.T) {
		doc := createTestDocument(fixtures)

		created, err := repo.Create(ctx, doc)
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.Equal(t, doc.FileName, created.FileName)
		assert.Equal(t, doc.OriginalName, created.OriginalName)
		assert.Equal(t, doc.FileSize, created.FileSize)
		assert.Equal(t, doc.Status, created.Status)
	})

	t.Run("create document with all fields", func(t *testing.T) {
		expDate := time.Now().Add(30 * 24 * time.Hour).Unix()
		doc := createTestDocument(fixtures, func(d *document.Document) {
			d.ExpirationDate = &expDate
			d.IsPublic = true
			d.Description = "Full document with all fields"
		})

		created, err := repo.Create(ctx, doc)
		require.NoError(t, err)
		assert.NotNil(t, created.ExpirationDate)
		assert.True(t, created.IsPublic)
	})
}

func TestDocumentRepository_GetByID_Integration(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	fixtures := newTestFixtures(t, ctx, db)

	repo := newTestRepository(db)

	doc := createTestDocument(fixtures)
	_, err := repo.Create(ctx, doc)
	require.NoError(t, err)

	t.Run("get existing document", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, repositories.GetDocumentByIDRequest{
			ID: doc.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, doc.ID, retrieved.ID)
		assert.Equal(t, doc.FileName, retrieved.FileName)
		assert.Equal(t, doc.ResourceType, retrieved.ResourceType)
	})

	t.Run("get non-existent document", func(t *testing.T) {
		_, err := repo.GetByID(ctx, repositories.GetDocumentByIDRequest{
			ID: pulid.MustNew("doc_"),
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		assert.Error(t, err)
	})

	t.Run("get document with wrong tenant", func(t *testing.T) {
		_, err := repo.GetByID(ctx, repositories.GetDocumentByIDRequest{
			ID: doc.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
		})
		assert.Error(t, err)
	})
}

func TestDocumentRepository_GetByResourceID_Integration(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	fixtures := newTestFixtures(t, ctx, db)

	repo := newTestRepository(db)

	resourceID := pulid.MustNew("tr_").String()

	doc1 := createTestDocument(fixtures, func(d *document.Document) {
		d.ResourceID = resourceID
		d.FileName = "doc1.pdf"
	})
	doc2 := createTestDocument(fixtures, func(d *document.Document) {
		d.ResourceID = resourceID
		d.FileName = "doc2.pdf"
	})
	doc3 := createTestDocument(fixtures, func(d *document.Document) {
		d.ResourceID = pulid.MustNew("tr_").String()
		d.FileName = "doc3.pdf"
	})

	_, err := repo.Create(ctx, doc1)
	require.NoError(t, err)
	_, err = repo.Create(ctx, doc2)
	require.NoError(t, err)
	_, err = repo.Create(ctx, doc3)
	require.NoError(t, err)

	t.Run("get documents by resource", func(t *testing.T) {
		docs, err := repo.GetByResourceID(ctx, repositories.GetDocumentsByResourceRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
			ResourceID:   resourceID,
			ResourceType: "trailer",
		})
		require.NoError(t, err)
		assert.Len(t, docs, 2)
	})

	t.Run("get documents for resource with no documents", func(t *testing.T) {
		docs, err := repo.GetByResourceID(ctx, repositories.GetDocumentsByResourceRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
			ResourceID:   pulid.MustNew("tr_").String(),
			ResourceType: "trailer",
		})
		require.NoError(t, err)
		assert.Len(t, docs, 0)
	})
}

func TestDocumentRepository_List_Integration(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	fixtures := newTestFixtures(t, ctx, db)

	repo := newTestRepository(db)

	for i := range 5 {
		doc := createTestDocument(fixtures, func(d *document.Document) {
			d.FileName = "document-" + string(rune('0'+i)) + ".pdf"
			if i%2 == 0 {
				d.Status = document.StatusActive
			} else {
				d.Status = document.StatusArchived
			}
		})
		_, err := repo.Create(ctx, doc)
		require.NoError(t, err)
	}

	t.Run("list all documents", func(t *testing.T) {
		result, err := repo.List(ctx, &repositories.ListDocumentsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{
					OrgID: fixtures.orgID,
					BuID:  fixtures.buID,
				},
				Pagination: pagination.Info{
					Limit:  10,
					Offset: 0,
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 5, result.Total)
		assert.Len(t, result.Items, 5)
	})

	t.Run("list with pagination", func(t *testing.T) {
		result, err := repo.List(ctx, &repositories.ListDocumentsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{
					OrgID: fixtures.orgID,
					BuID:  fixtures.buID,
				},
				Pagination: pagination.Info{
					Limit:  2,
					Offset: 0,
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 5, result.Total)
		assert.Len(t, result.Items, 2)
	})

	t.Run("list with status filter", func(t *testing.T) {
		result, err := repo.List(ctx, &repositories.ListDocumentsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{
					OrgID: fixtures.orgID,
					BuID:  fixtures.buID,
				},
				Pagination: pagination.Info{
					Limit:  10,
					Offset: 0,
				},
			},
			Status: "Active",
		})
		require.NoError(t, err)
		assert.Equal(t, 3, result.Total)
	})
}

func TestDocumentRepository_Update_Integration(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	fixtures := newTestFixtures(t, ctx, db)

	repo := newTestRepository(db)

	doc := createTestDocument(fixtures)
	created, err := repo.Create(ctx, doc)
	require.NoError(t, err)

	t.Run("update document successfully", func(t *testing.T) {
		created.Description = "Updated description"
		created.Status = document.StatusArchived

		updated, err := repo.Update(ctx, created)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", updated.Description)
		assert.Equal(t, document.StatusArchived, updated.Status)
		assert.Equal(t, int64(1), updated.Version)
	})

	t.Run("update with version conflict", func(t *testing.T) {
		created.Version = 0
		created.Description = "Should fail"

		_, err := repo.Update(ctx, created)
		assert.Error(t, err)
	})
}

func TestDocumentRepository_Delete_Integration(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	fixtures := newTestFixtures(t, ctx, db)

	repo := newTestRepository(db)

	doc := createTestDocument(fixtures)
	_, err := repo.Create(ctx, doc)
	require.NoError(t, err)

	t.Run("delete existing document", func(t *testing.T) {
		err := repo.Delete(ctx, repositories.DeleteDocumentRequest{
			ID: doc.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, repositories.GetDocumentByIDRequest{
			ID: doc.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		assert.Error(t, err)
	})

	t.Run("delete non-existent document", func(t *testing.T) {
		err := repo.Delete(ctx, repositories.DeleteDocumentRequest{
			ID: pulid.MustNew("doc_"),
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		assert.Error(t, err)
	})
}

func TestDocumentRepository_MultiTenancy_Integration(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	fixtures1 := newTestFixtures(t, ctx, db)
	second := seedtest.SeedAdditionalTenant(t, ctx, db, "TWO")
	fixtures2 := &testFixtures{
		orgID:  second.Organization.ID,
		buID:   second.BusinessUnit.ID,
		userID: second.User.ID,
	}
	org1, bu1 := fixtures1.orgID, fixtures1.buID
	org2, bu2 := fixtures2.orgID, fixtures2.buID

	repo := newTestRepository(db)

	doc1 := createTestDocument(fixtures1, func(d *document.Document) {
		d.FileName = "org1-doc.pdf"
	})
	doc2 := createTestDocument(fixtures2, func(d *document.Document) {
		d.FileName = "org2-doc.pdf"
	})

	_, err := repo.Create(ctx, doc1)
	require.NoError(t, err)
	_, err = repo.Create(ctx, doc2)
	require.NoError(t, err)

	t.Run("org1 can only see org1 documents", func(t *testing.T) {
		result, err := repo.List(ctx, &repositories.ListDocumentsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{
					OrgID: org1,
					BuID:  bu1,
				},
				Pagination: pagination.Info{Limit: 10},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, result.Total)
		assert.Equal(t, "org1-doc.pdf", result.Items[0].FileName)
	})

	t.Run("org2 can only see org2 documents", func(t *testing.T) {
		result, err := repo.List(ctx, &repositories.ListDocumentsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{
					OrgID: org2,
					BuID:  bu2,
				},
				Pagination: pagination.Info{Limit: 10},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, result.Total)
		assert.Equal(t, "org2-doc.pdf", result.Items[0].FileName)
	})

	t.Run("org1 cannot access org2 document by ID", func(t *testing.T) {
		_, err := repo.GetByID(ctx, repositories.GetDocumentByIDRequest{
			ID: doc2.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: org1,
				BuID:  bu1,
			},
		})
		assert.Error(t, err)
	})

	t.Run("org1 cannot delete org2 document", func(t *testing.T) {
		err := repo.Delete(ctx, repositories.DeleteDocumentRequest{
			ID: doc2.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: org1,
				BuID:  bu1,
			},
		})
		assert.Error(t, err)
	})
}
