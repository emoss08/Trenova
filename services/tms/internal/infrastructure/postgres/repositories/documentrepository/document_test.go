//go:build integration

package documentrepository_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
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

func createTestSchema(t *testing.T, db *bun.DB, tc *testutil.TestContext) {
	t.Helper()

	queries := []string{
		`CREATE TABLE IF NOT EXISTS organizations (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS business_units (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			organization_id VARCHAR(100) REFERENCES organizations(id)
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			organization_id VARCHAR(100) REFERENCES organizations(id)
		)`,
		`DO $$ BEGIN
			CREATE TYPE document_status_enum AS ENUM (
				'Draft', 'Active', 'Archived', 'Expired', 'Pending', 'Rejected', 'PendingApproval'
			);
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$`,
		`DO $$ BEGIN
			CREATE TYPE document_preview_status_enum AS ENUM (
				'Pending', 'Ready', 'Failed', 'Unsupported'
			);
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$`,
		`DO $$ BEGIN
			CREATE TYPE document_content_status_enum AS ENUM (
				'Pending', 'Extracting', 'Extracted', 'Indexed', 'Failed'
			);
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$`,
		`DO $$ BEGIN
			CREATE TYPE document_shipment_draft_status_enum AS ENUM (
				'Unavailable', 'Pending', 'Ready', 'Failed'
			);
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$`,
		`CREATE TABLE IF NOT EXISTS documents (
			id VARCHAR(100) NOT NULL,
			organization_id VARCHAR(100) NOT NULL REFERENCES organizations(id),
			business_unit_id VARCHAR(100) NOT NULL REFERENCES business_units(id),
			lineage_id VARCHAR(100) NOT NULL,
			version_number BIGINT NOT NULL DEFAULT 1,
			is_current_version BOOLEAN NOT NULL DEFAULT TRUE,
			file_name VARCHAR(255) NOT NULL,
			original_name VARCHAR(255) NOT NULL,
			file_size BIGINT NOT NULL,
			file_type VARCHAR(100) NOT NULL,
			storage_path VARCHAR(500) NOT NULL,
			checksum_sha256 VARCHAR(64),
			storage_version_id VARCHAR(255),
			storage_retention_mode VARCHAR(50),
			storage_retention_until BIGINT,
			storage_legal_hold BOOLEAN NOT NULL DEFAULT FALSE,
			status document_status_enum NOT NULL DEFAULT 'Active',
			description TEXT,
			resource_id VARCHAR(100) NOT NULL,
			resource_type VARCHAR(100) NOT NULL,
			processing_profile VARCHAR(64) NOT NULL DEFAULT 'none',
			expiration_date BIGINT,
			tags VARCHAR(100)[] DEFAULT '{}',
			is_public BOOLEAN NOT NULL DEFAULT FALSE,
			uploaded_by_id VARCHAR(100) NOT NULL REFERENCES users(id),
			approved_by_id VARCHAR(100) REFERENCES users(id),
			approved_at BIGINT,
			preview_storage_path VARCHAR(500),
			preview_status document_preview_status_enum NOT NULL DEFAULT 'Unsupported',
			content_status document_content_status_enum NOT NULL DEFAULT 'Pending',
			content_error TEXT,
			detected_kind VARCHAR(100),
			has_extracted_text BOOLEAN NOT NULL DEFAULT FALSE,
			shipment_draft_status document_shipment_draft_status_enum NOT NULL DEFAULT 'Unavailable',
			document_type_id VARCHAR(100),
			version BIGINT NOT NULL DEFAULT 0,
			created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL,
			PRIMARY KEY (id, organization_id, business_unit_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_documents_resource ON documents(resource_type, resource_id)`,
		`CREATE INDEX IF NOT EXISTS idx_documents_tenant ON documents(business_unit_id, organization_id)`,
	}

	for _, q := range queries {
		_, err := db.ExecContext(tc.Ctx, q)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			require.NoError(t, err, "failed to execute: %s", q)
		}
	}
}

type testFixtures struct {
	orgID  pulid.ID
	buID   pulid.ID
	userID pulid.ID
}

func createTestFixtures(t *testing.T, db *bun.DB, tc *testutil.TestContext) *testFixtures {
	t.Helper()

	orgID := pulid.MustNew("org_")
	_, err := db.ExecContext(tc.Ctx,
		`INSERT INTO organizations (id, name) VALUES (?, ?)`,
		orgID.String(), "Test Org",
	)
	require.NoError(t, err)

	buID := pulid.MustNew("bu_")
	_, err = db.ExecContext(tc.Ctx,
		`INSERT INTO business_units (id, name, organization_id) VALUES (?, ?, ?)`,
		buID.String(), "Test BU", orgID.String(),
	)
	require.NoError(t, err)

	userID := pulid.MustNew("usr_")
	_, err = db.ExecContext(tc.Ctx,
		`INSERT INTO users (id, name, organization_id) VALUES (?, ?, ?)`,
		userID.String(), "Test User", orgID.String(),
	)
	require.NoError(t, err)

	return &testFixtures{
		orgID:  orgID,
		buID:   buID,
		userID: userID,
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
	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, tc)
	fixtures := createTestFixtures(t, db, tc)

	repo := newTestRepository(db)

	t.Run("create document successfully", func(t *testing.T) {
		doc := createTestDocument(fixtures)

		created, err := repo.Create(tc.Ctx, doc)
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

		created, err := repo.Create(tc.Ctx, doc)
		require.NoError(t, err)
		assert.NotNil(t, created.ExpirationDate)
		assert.True(t, created.IsPublic)
	})
}

func TestDocumentRepository_GetByID_Integration(t *testing.T) {
	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, tc)
	fixtures := createTestFixtures(t, db, tc)

	repo := newTestRepository(db)

	doc := createTestDocument(fixtures)
	_, err := repo.Create(tc.Ctx, doc)
	require.NoError(t, err)

	t.Run("get existing document", func(t *testing.T) {
		retrieved, err := repo.GetByID(tc.Ctx, repositories.GetDocumentByIDRequest{
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
		_, err := repo.GetByID(tc.Ctx, repositories.GetDocumentByIDRequest{
			ID: pulid.MustNew("doc_"),
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		assert.Error(t, err)
	})

	t.Run("get document with wrong tenant", func(t *testing.T) {
		_, err := repo.GetByID(tc.Ctx, repositories.GetDocumentByIDRequest{
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
	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, tc)
	fixtures := createTestFixtures(t, db, tc)

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

	_, err := repo.Create(tc.Ctx, doc1)
	require.NoError(t, err)
	_, err = repo.Create(tc.Ctx, doc2)
	require.NoError(t, err)
	_, err = repo.Create(tc.Ctx, doc3)
	require.NoError(t, err)

	t.Run("get documents by resource", func(t *testing.T) {
		docs, err := repo.GetByResourceID(tc.Ctx, repositories.GetDocumentsByResourceRequest{
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
		docs, err := repo.GetByResourceID(tc.Ctx, repositories.GetDocumentsByResourceRequest{
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
	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, tc)
	fixtures := createTestFixtures(t, db, tc)

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
		_, err := repo.Create(tc.Ctx, doc)
		require.NoError(t, err)
	}

	t.Run("list all documents", func(t *testing.T) {
		result, err := repo.List(tc.Ctx, &repositories.ListDocumentsRequest{
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
		result, err := repo.List(tc.Ctx, &repositories.ListDocumentsRequest{
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
		result, err := repo.List(tc.Ctx, &repositories.ListDocumentsRequest{
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
	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, tc)
	fixtures := createTestFixtures(t, db, tc)

	repo := newTestRepository(db)

	doc := createTestDocument(fixtures)
	created, err := repo.Create(tc.Ctx, doc)
	require.NoError(t, err)

	t.Run("update document successfully", func(t *testing.T) {
		created.Description = "Updated description"
		created.Status = document.StatusArchived

		updated, err := repo.Update(tc.Ctx, created)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", updated.Description)
		assert.Equal(t, document.StatusArchived, updated.Status)
		assert.Equal(t, int64(1), updated.Version)
	})

	t.Run("update with version conflict", func(t *testing.T) {
		created.Version = 0
		created.Description = "Should fail"

		_, err := repo.Update(tc.Ctx, created)
		assert.Error(t, err)
	})
}

func TestDocumentRepository_Delete_Integration(t *testing.T) {
	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, tc)
	fixtures := createTestFixtures(t, db, tc)

	repo := newTestRepository(db)

	doc := createTestDocument(fixtures)
	_, err := repo.Create(tc.Ctx, doc)
	require.NoError(t, err)

	t.Run("delete existing document", func(t *testing.T) {
		err := repo.Delete(tc.Ctx, repositories.DeleteDocumentRequest{
			ID: doc.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		require.NoError(t, err)

		_, err = repo.GetByID(tc.Ctx, repositories.GetDocumentByIDRequest{
			ID: doc.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		assert.Error(t, err)
	})

	t.Run("delete non-existent document", func(t *testing.T) {
		err := repo.Delete(tc.Ctx, repositories.DeleteDocumentRequest{
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
	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, tc)

	org1 := pulid.MustNew("org_")
	_, err := db.ExecContext(
		tc.Ctx,
		`INSERT INTO organizations (id, name) VALUES (?, ?)`,
		org1.String(),
		"Org 1",
	)
	require.NoError(t, err)

	bu1 := pulid.MustNew("bu_")
	_, err = db.ExecContext(
		tc.Ctx,
		`INSERT INTO business_units (id, name, organization_id) VALUES (?, ?, ?)`,
		bu1.String(),
		"BU 1",
		org1.String(),
	)
	require.NoError(t, err)

	user1 := pulid.MustNew("usr_")
	_, err = db.ExecContext(
		tc.Ctx,
		`INSERT INTO users (id, name, organization_id) VALUES (?, ?, ?)`,
		user1.String(),
		"User 1",
		org1.String(),
	)
	require.NoError(t, err)

	org2 := pulid.MustNew("org_")
	_, err = db.ExecContext(
		tc.Ctx,
		`INSERT INTO organizations (id, name) VALUES (?, ?)`,
		org2.String(),
		"Org 2",
	)
	require.NoError(t, err)

	bu2 := pulid.MustNew("bu_")
	_, err = db.ExecContext(
		tc.Ctx,
		`INSERT INTO business_units (id, name, organization_id) VALUES (?, ?, ?)`,
		bu2.String(),
		"BU 2",
		org2.String(),
	)
	require.NoError(t, err)

	user2 := pulid.MustNew("usr_")
	_, err = db.ExecContext(
		tc.Ctx,
		`INSERT INTO users (id, name, organization_id) VALUES (?, ?, ?)`,
		user2.String(),
		"User 2",
		org2.String(),
	)
	require.NoError(t, err)

	repo := newTestRepository(db)

	fixtures1 := &testFixtures{orgID: org1, buID: bu1, userID: user1}
	fixtures2 := &testFixtures{orgID: org2, buID: bu2, userID: user2}

	doc1 := createTestDocument(fixtures1, func(d *document.Document) {
		d.FileName = "org1-doc.pdf"
	})
	doc2 := createTestDocument(fixtures2, func(d *document.Document) {
		d.FileName = "org2-doc.pdf"
	})

	_, err = repo.Create(tc.Ctx, doc1)
	require.NoError(t, err)
	_, err = repo.Create(tc.Ctx, doc2)
	require.NoError(t, err)

	t.Run("org1 can only see org1 documents", func(t *testing.T) {
		result, err := repo.List(tc.Ctx, &repositories.ListDocumentsRequest{
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
		result, err := repo.List(tc.Ctx, &repositories.ListDocumentsRequest{
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
		_, err := repo.GetByID(tc.Ctx, repositories.GetDocumentByIDRequest{
			ID: doc2.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: org1,
				BuID:  bu1,
			},
		})
		assert.Error(t, err)
	})

	t.Run("org1 cannot delete org2 document", func(t *testing.T) {
		err := repo.Delete(tc.Ctx, repositories.DeleteDocumentRequest{
			ID: doc2.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: org1,
				BuID:  bu1,
			},
		})
		assert.Error(t, err)
	})
}
