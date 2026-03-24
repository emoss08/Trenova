//go:build integration

package documentservice_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	minioadapter "github.com/emoss08/trenova/internal/infrastructure/minio"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	storageutil "github.com/emoss08/trenova/shared/testutil/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *testRepository) List(
	ctx context.Context,
	req *repositories.ListDocumentsRequest,
) (*pagination.ListResult[*document.Document], error) {
	entities := make([]*document.Document, 0, req.Filter.Pagination.Limit)
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
	req *repositories.GetDocumentsByResourceRequest,
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
			return sq.Where("doc.id IN (?)", bun.In(req.IDs)).
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
			return dq.Where("id IN (?)", bun.In(req.IDs)).
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

var _ repositories.DocumentRepository = (*testRepository)(nil)

func createTestSchema(t *testing.T, db *bun.DB, ctx context.Context) {
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
		`CREATE TABLE IF NOT EXISTS documents (
			id VARCHAR(100) NOT NULL,
			organization_id VARCHAR(100) NOT NULL REFERENCES organizations(id),
			business_unit_id VARCHAR(100) NOT NULL REFERENCES business_units(id),
			file_name VARCHAR(255) NOT NULL,
			original_name VARCHAR(255) NOT NULL,
			file_size BIGINT NOT NULL,
			file_type VARCHAR(100) NOT NULL,
			storage_path VARCHAR(500) NOT NULL,
			status document_status_enum NOT NULL DEFAULT 'Active',
			description TEXT,
			resource_id VARCHAR(100) NOT NULL,
			resource_type VARCHAR(100) NOT NULL,
			expiration_date BIGINT,
			tags VARCHAR(100)[] DEFAULT '{}',
			is_public BOOLEAN NOT NULL DEFAULT FALSE,
			uploaded_by_id VARCHAR(100) NOT NULL REFERENCES users(id),
			approved_by_id VARCHAR(100) REFERENCES users(id),
			approved_at BIGINT,
			preview_storage_path VARCHAR(500),
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
		_, err := db.ExecContext(ctx, q)
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

func createTestFixtures(t *testing.T, db *bun.DB, ctx context.Context) *testFixtures {
	t.Helper()

	orgID := pulid.MustNew("org_")
	_, err := db.ExecContext(ctx,
		`INSERT INTO organizations (id, name) VALUES (?, ?)`,
		orgID.String(), "Test Org",
	)
	require.NoError(t, err)

	buID := pulid.MustNew("bu_")
	_, err = db.ExecContext(ctx,
		`INSERT INTO business_units (id, name, organization_id) VALUES (?, ?, ?)`,
		buID.String(), "Test BU", orgID.String(),
	)
	require.NoError(t, err)

	userID := pulid.MustNew("usr_")
	_, err = db.ExecContext(ctx,
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

func setupTestService(t *testing.T, db *bun.DB, cfg *config.Config) *documentservice.Service {
	t.Helper()

	repo := newTestRepository(db)
	logger := zap.NewNop()

	storageClient, err := minioadapter.New(minioadapter.Params{
		Config: cfg,
		Logger: logger,
	})
	require.NoError(t, err)

	validator := documentservice.NewValidator(documentservice.ValidatorParams{Config: cfg})
	thumbnailGen := thumbnailservice.NewGenerator()
	cacheRepo := mocks.NewMockDocumentCacheRepository(t)
	cacheRepo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(nil, repositories.ErrCacheMiss)

	service := documentservice.NewTestService(
		logger,
		repo,
		cacheRepo,
		storageClient,
		validator,
		&mocks.NoopAuditService{},
		cfg.GetStorageConfig(),
		thumbnailGen,
		nil,
	)

	return service
}

func TestService_Upload_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires Temporal client setup")
	t.Skip("Skipping integration test - requires Temporal client setup")
	dbCtx, db := testutil.SetupTestDB(t)
	minioCtx, mc := testutil.SetupTestMinio(t)
	defer dbCtx.Cancel()
	defer minioCtx.Cancel()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			Endpoint:  mc.Endpoint(),
			AccessKey: mc.AccessKey(),
			SecretKey: mc.SecretKey(),
			Bucket:    mc.Bucket(),
			UseSSL:    false,
			AllowedMIMETypes: []string{
				"application/pdf",
				"image/jpeg",
				"image/png",
				"text/plain",
			},
			MaxFileSize: 50 * 1024 * 1024,
		},
	}

	createTestSchema(t, db, dbCtx.Ctx)
	fixtures := createTestFixtures(t, db, dbCtx.Ctx)

	service := setupTestService(t, db, cfg)

	t.Run("upload file successfully", func(t *testing.T) {
		fileData := []byte("This is a test PDF content")
		fileHeader := storageutil.NewMockFileHeader(
			"test-document.pdf",
			fileData,
			"application/pdf",
		)

		result, err := service.Upload(dbCtx.Ctx, &documentservice.UploadRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  fixtures.orgID,
				BuID:   fixtures.buID,
				UserID: fixtures.userID,
			},
			File:         fileHeader,
			ResourceID:   pulid.MustNew("tr_").String(),
			ResourceType: "trailer",
			Description:  "Test document",
			Tags:         []string{"test", "integration"},
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Document)
		assert.Equal(t, "test-document.pdf", result.Document.OriginalName)
		assert.Equal(t, int64(len(fileData)), result.Document.FileSize)
		assert.Equal(t, document.StatusActive, result.Document.Status)
		assert.Contains(t, result.Document.StoragePath, fixtures.orgID.String())
		assert.Contains(t, result.Document.StoragePath, "trailer")

		storageClient, _ := minioadapter.New(minioadapter.Params{
			Config: cfg,
			Logger: zap.NewNop(),
		})
		exists, err := storageClient.Exists(dbCtx.Ctx, result.Document.StoragePath)
		require.NoError(t, err)
		assert.True(t, exists, "file should exist in MinIO")
	})

	t.Run("upload creates database record", func(t *testing.T) {
		fileData := []byte("Database record test")
		fileHeader := storageutil.NewMockFileHeader("db-test.txt", fileData, "text/plain")

		resourceID := pulid.MustNew("tr_").String()
		result, err := service.Upload(dbCtx.Ctx, &documentservice.UploadRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  fixtures.orgID,
				BuID:   fixtures.buID,
				UserID: fixtures.userID,
			},
			File:         fileHeader,
			ResourceID:   resourceID,
			ResourceType: "trailer",
		})
		require.NoError(t, err)

		retrieved, err := service.Get(dbCtx.Ctx, repositories.GetDocumentByIDRequest{
			ID: result.Document.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, result.Document.ID, retrieved.ID)
		assert.Equal(t, resourceID, retrieved.ResourceID)
	})
}

func TestService_BulkUpload_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires Temporal client setup")
	t.Skip("Skipping integration test - requires Temporal client setup")
	dbCtx, db := testutil.SetupTestDB(t)
	minioCtx, mc := testutil.SetupTestMinio(t)
	defer dbCtx.Cancel()
	defer minioCtx.Cancel()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			Endpoint:  mc.Endpoint(),
			AccessKey: mc.AccessKey(),
			SecretKey: mc.SecretKey(),
			Bucket:    mc.Bucket(),
			UseSSL:    false,
			AllowedMIMETypes: []string{
				"application/pdf",
				"image/jpeg",
				"text/plain",
			},
			MaxFileSize: 50 * 1024 * 1024,
		},
	}

	createTestSchema(t, db, dbCtx.Ctx)
	fixtures := createTestFixtures(t, db, dbCtx.Ctx)

	service := setupTestService(t, db, cfg)

	t.Run("bulk upload multiple files", func(t *testing.T) {
		files := storageutil.NewMockFileHeaders([]storageutil.MockFileHeader{
			{Filename: "doc1.pdf", Data: []byte("PDF content 1"), ContentType: "application/pdf"},
			{Filename: "doc2.txt", Data: []byte("Text content 2"), ContentType: "text/plain"},
			{Filename: "image.jpg", Data: []byte("JPEG data"), ContentType: "image/jpeg"},
		})

		resourceID := pulid.MustNew("tr_").String()
		result, err := service.BulkUpload(dbCtx.Ctx, &documentservice.BulkUploadRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  fixtures.orgID,
				BuID:   fixtures.buID,
				UserID: fixtures.userID,
			},
			Files:        files,
			ResourceID:   resourceID,
			ResourceType: "trailer",
		})
		require.NoError(t, err)
		assert.Len(t, result.Documents, 3)
		assert.Len(t, result.Errors, 0)

		docs, err := service.GetByResource(dbCtx.Ctx, &repositories.GetDocumentsByResourceRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
			ResourceID:   resourceID,
			ResourceType: "trailer",
		})
		require.NoError(t, err)
		assert.Len(t, docs, 3)
	})
}

func TestService_GetDownloadURL_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires Temporal client setup")
	dbCtx, db := testutil.SetupTestDB(t)
	minioCtx, mc := testutil.SetupTestMinio(t)
	defer dbCtx.Cancel()
	defer minioCtx.Cancel()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			Endpoint:           mc.Endpoint(),
			AccessKey:          mc.AccessKey(),
			SecretKey:          mc.SecretKey(),
			Bucket:             mc.Bucket(),
			UseSSL:             false,
			AllowedMIMETypes:   []string{"application/pdf"},
			MaxFileSize:        50 * 1024 * 1024,
			PresignedURLExpiry: time.Duration(15 * time.Minute),
		},
	}

	createTestSchema(t, db, dbCtx.Ctx)
	fixtures := createTestFixtures(t, db, dbCtx.Ctx)

	service := setupTestService(t, db, cfg)

	t.Run("get presigned download URL", func(t *testing.T) {
		fileData := []byte("Download test content")
		fileHeader := storageutil.NewMockFileHeader(
			"download-test.pdf",
			fileData,
			"application/pdf",
		)

		uploadResult, err := service.Upload(dbCtx.Ctx, &documentservice.UploadRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  fixtures.orgID,
				BuID:   fixtures.buID,
				UserID: fixtures.userID,
			},
			File:         fileHeader,
			ResourceID:   pulid.MustNew("tr_").String(),
			ResourceType: "trailer",
		})
		require.NoError(t, err)

		downloadURL, err := service.GetDownloadURL(dbCtx.Ctx, repositories.GetDocumentByIDRequest{
			ID: uploadResult.Document.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, downloadURL)
		assert.Contains(t, downloadURL, mc.Bucket())
		assert.Contains(t, downloadURL, "X-Amz-Signature")
		assert.Contains(t, downloadURL, "response-content-disposition")
	})

	t.Run("get download URL for non-existent document", func(t *testing.T) {
		_, err := service.GetDownloadURL(dbCtx.Ctx, repositories.GetDocumentByIDRequest{
			ID: pulid.MustNew("doc_"),
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		assert.Error(t, err)
	})
}

func TestService_Delete_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires Temporal client setup")
	dbCtx, db := testutil.SetupTestDB(t)
	minioCtx, mc := testutil.SetupTestMinio(t)
	defer dbCtx.Cancel()
	defer minioCtx.Cancel()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			Endpoint:         mc.Endpoint(),
			AccessKey:        mc.AccessKey(),
			SecretKey:        mc.SecretKey(),
			Bucket:           mc.Bucket(),
			UseSSL:           false,
			AllowedMIMETypes: []string{"application/pdf"},
			MaxFileSize:      50 * 1024 * 1024,
		},
	}

	createTestSchema(t, db, dbCtx.Ctx)
	fixtures := createTestFixtures(t, db, dbCtx.Ctx)

	service := setupTestService(t, db, cfg)

	t.Run("delete removes from storage and database", func(t *testing.T) {
		fileData := []byte("Delete test content")
		fileHeader := storageutil.NewMockFileHeader("delete-test.pdf", fileData, "application/pdf")

		uploadResult, err := service.Upload(dbCtx.Ctx, &documentservice.UploadRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  fixtures.orgID,
				BuID:   fixtures.buID,
				UserID: fixtures.userID,
			},
			File:         fileHeader,
			ResourceID:   pulid.MustNew("tr_").String(),
			ResourceType: "trailer",
		})
		require.NoError(t, err)

		storagePath := uploadResult.Document.StoragePath

		storageClient, _ := minioadapter.New(minioadapter.Params{
			Config: cfg,
			Logger: zap.NewNop(),
		})
		exists, err := storageClient.Exists(dbCtx.Ctx, storagePath)
		require.NoError(t, err)
		assert.True(t, exists)

		err = service.Delete(dbCtx.Ctx, repositories.DeleteDocumentRequest{
			ID: uploadResult.Document.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		}, fixtures.userID)
		require.NoError(t, err)

		_, err = service.Get(dbCtx.Ctx, repositories.GetDocumentByIDRequest{
			ID: uploadResult.Document.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		assert.Error(t, err)

		exists, err = storageClient.Exists(dbCtx.Ctx, storagePath)
		require.NoError(t, err)
		assert.False(t, exists, "file should be deleted from MinIO")
	})

	t.Run("delete non-existent document", func(t *testing.T) {
		err := service.Delete(dbCtx.Ctx, repositories.DeleteDocumentRequest{
			ID: pulid.MustNew("doc_"),
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		}, fixtures.userID)
		assert.Error(t, err)
	})
}

func TestService_MultiTenancy_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires Temporal client setup")
	dbCtx, db := testutil.SetupTestDB(t)
	minioCtx, mc := testutil.SetupTestMinio(t)
	defer dbCtx.Cancel()
	defer minioCtx.Cancel()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			Endpoint:         mc.Endpoint(),
			AccessKey:        mc.AccessKey(),
			SecretKey:        mc.SecretKey(),
			Bucket:           mc.Bucket(),
			UseSSL:           false,
			AllowedMIMETypes: []string{"application/pdf"},
			MaxFileSize:      50 * 1024 * 1024,
		},
	}

	createTestSchema(t, db, dbCtx.Ctx)

	org1 := pulid.MustNew("org_")
	_, err := db.ExecContext(
		dbCtx.Ctx,
		`INSERT INTO organizations (id, name) VALUES (?, ?)`,
		org1.String(),
		"Org 1",
	)
	require.NoError(t, err)

	bu1 := pulid.MustNew("bu_")
	_, err = db.ExecContext(
		dbCtx.Ctx,
		`INSERT INTO business_units (id, name, organization_id) VALUES (?, ?, ?)`,
		bu1.String(),
		"BU 1",
		org1.String(),
	)
	require.NoError(t, err)

	user1 := pulid.MustNew("usr_")
	_, err = db.ExecContext(
		dbCtx.Ctx,
		`INSERT INTO users (id, name, organization_id) VALUES (?, ?, ?)`,
		user1.String(),
		"User 1",
		org1.String(),
	)
	require.NoError(t, err)

	org2 := pulid.MustNew("org_")
	_, err = db.ExecContext(
		dbCtx.Ctx,
		`INSERT INTO organizations (id, name) VALUES (?, ?)`,
		org2.String(),
		"Org 2",
	)
	require.NoError(t, err)

	bu2 := pulid.MustNew("bu_")
	_, err = db.ExecContext(
		dbCtx.Ctx,
		`INSERT INTO business_units (id, name, organization_id) VALUES (?, ?, ?)`,
		bu2.String(),
		"BU 2",
		org2.String(),
	)
	require.NoError(t, err)

	user2 := pulid.MustNew("usr_")
	_, err = db.ExecContext(
		dbCtx.Ctx,
		`INSERT INTO users (id, name, organization_id) VALUES (?, ?, ?)`,
		user2.String(),
		"User 2",
		org2.String(),
	)
	require.NoError(t, err)

	service := setupTestService(t, db, cfg)

	fileData := []byte("Org 1 document")
	fileHeader := storageutil.NewMockFileHeader("org1-doc.pdf", fileData, "application/pdf")

	org1UploadResult, err := service.Upload(dbCtx.Ctx, &documentservice.UploadRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  org1,
			BuID:   bu1,
			UserID: user1,
		},
		File:         fileHeader,
		ResourceID:   pulid.MustNew("tr_").String(),
		ResourceType: "trailer",
	})
	require.NoError(t, err)

	t.Run("org2 cannot access org1 document", func(t *testing.T) {
		_, err := service.Get(dbCtx.Ctx, repositories.GetDocumentByIDRequest{
			ID: org1UploadResult.Document.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: org2,
				BuID:  bu2,
			},
		})
		assert.Error(t, err)
	})

	t.Run("org2 cannot get download URL for org1 document", func(t *testing.T) {
		_, err := service.GetDownloadURL(dbCtx.Ctx, repositories.GetDocumentByIDRequest{
			ID: org1UploadResult.Document.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: org2,
				BuID:  bu2,
			},
		})
		assert.Error(t, err)
	})

	t.Run("org2 cannot delete org1 document", func(t *testing.T) {
		err := service.Delete(dbCtx.Ctx, repositories.DeleteDocumentRequest{
			ID: org1UploadResult.Document.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: org2,
				BuID:  bu2,
			},
		}, user2)
		assert.Error(t, err)
	})

	t.Run("storage path includes org isolation", func(t *testing.T) {
		assert.Contains(t, org1UploadResult.Document.StoragePath, org1.String())
	})

	t.Run("documents are isolated by org in list", func(t *testing.T) {
		org1Docs, err := service.List(dbCtx.Ctx, &repositories.ListDocumentsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{
					OrgID: org1,
					BuID:  bu1,
				},
				Pagination: pagination.Info{Limit: 10},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, org1Docs.Total)

		org2Docs, err := service.List(dbCtx.Ctx, &repositories.ListDocumentsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{
					OrgID: org2,
					BuID:  bu2,
				},
				Pagination: pagination.Info{Limit: 10},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 0, org2Docs.Total)
	})
}

func TestService_List_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires Temporal client setup")
	dbCtx, db := testutil.SetupTestDB(t)
	minioCtx, mc := testutil.SetupTestMinio(t)
	defer dbCtx.Cancel()
	defer minioCtx.Cancel()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			Endpoint:         mc.Endpoint(),
			AccessKey:        mc.AccessKey(),
			SecretKey:        mc.SecretKey(),
			Bucket:           mc.Bucket(),
			UseSSL:           false,
			AllowedMIMETypes: []string{"application/pdf"},
			MaxFileSize:      50 * 1024 * 1024,
		},
	}

	createTestSchema(t, db, dbCtx.Ctx)
	fixtures := createTestFixtures(t, db, dbCtx.Ctx)

	service := setupTestService(t, db, cfg)

	for i := range 5 {
		fileData := []byte("Test content " + string(rune('0'+i)))
		fileHeader := storageutil.NewMockFileHeader(
			"doc-"+string(rune('0'+i))+".pdf",
			fileData,
			"application/pdf",
		)

		_, err := service.Upload(dbCtx.Ctx, &documentservice.UploadRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  fixtures.orgID,
				BuID:   fixtures.buID,
				UserID: fixtures.userID,
			},
			File:         fileHeader,
			ResourceID:   pulid.MustNew("tr_").String(),
			ResourceType: "trailer",
		})
		require.NoError(t, err)
	}

	t.Run("list all documents", func(t *testing.T) {
		result, err := service.List(dbCtx.Ctx, &repositories.ListDocumentsRequest{
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
		result, err := service.List(dbCtx.Ctx, &repositories.ListDocumentsRequest{
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
}

func TestService_GetByResource_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires Temporal client setup")
	dbCtx, db := testutil.SetupTestDB(t)
	minioCtx, mc := testutil.SetupTestMinio(t)
	defer dbCtx.Cancel()
	defer minioCtx.Cancel()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			Endpoint:         mc.Endpoint(),
			AccessKey:        mc.AccessKey(),
			SecretKey:        mc.SecretKey(),
			Bucket:           mc.Bucket(),
			UseSSL:           false,
			AllowedMIMETypes: []string{"application/pdf"},
			MaxFileSize:      50 * 1024 * 1024,
		},
	}

	createTestSchema(t, db, dbCtx.Ctx)
	fixtures := createTestFixtures(t, db, dbCtx.Ctx)

	service := setupTestService(t, db, cfg)

	resourceID := pulid.MustNew("tr_").String()
	otherResourceID := pulid.MustNew("tr_").String()

	for i := range 3 {
		fileData := []byte("Resource content " + string(rune('0'+i)))
		fileHeader := storageutil.NewMockFileHeader(
			"resource-doc-"+string(rune('0'+i))+".pdf",
			fileData,
			"application/pdf",
		)

		_, err := service.Upload(dbCtx.Ctx, &documentservice.UploadRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  fixtures.orgID,
				BuID:   fixtures.buID,
				UserID: fixtures.userID,
			},
			File:         fileHeader,
			ResourceID:   resourceID,
			ResourceType: "trailer",
		})
		require.NoError(t, err)
	}

	fileData := []byte("Other resource content")
	fileHeader := storageutil.NewMockFileHeader("other-doc.pdf", fileData, "application/pdf")
	_, err := service.Upload(dbCtx.Ctx, &documentservice.UploadRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  fixtures.orgID,
			BuID:   fixtures.buID,
			UserID: fixtures.userID,
		},
		File:         fileHeader,
		ResourceID:   otherResourceID,
		ResourceType: "trailer",
	})
	require.NoError(t, err)

	t.Run("get documents for specific resource", func(t *testing.T) {
		docs, err := service.GetByResource(dbCtx.Ctx, &repositories.GetDocumentsByResourceRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
			ResourceID:   resourceID,
			ResourceType: "trailer",
		})
		require.NoError(t, err)
		assert.Len(t, docs, 3)
	})

	t.Run("get documents for resource with no documents", func(t *testing.T) {
		docs, err := service.GetByResource(dbCtx.Ctx, &repositories.GetDocumentsByResourceRequest{
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
