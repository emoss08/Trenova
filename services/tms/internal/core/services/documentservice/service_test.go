//go:build integration

package documentservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	minioadapter "github.com/emoss08/trenova/internal/infrastructure/minio"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/documentpacketrulerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/documentrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/documenttyperepository"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	storageutil "github.com/emoss08/trenova/shared/testutil/storage"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type serviceHarness struct {
	ctx              context.Context
	db               *bun.DB
	tenantInfo       pagination.TenantInfo
	service          *documentservice.Service
	documentRepo     repositories.DocumentRepository
	documentTypeRepo repositories.DocumentTypeRepository
	packetRuleRepo   repositories.DocumentPacketRuleRepository
	storage          storage.Client
	cacheRepo        *mocks.MockDocumentCacheRepository
	sessionRepo      *mocks.MockDocumentUploadSessionRepository
	contentService   *mocks.MockDocumentContentService
	searchProjection *mocks.MockDocumentSearchProjectionService
}

func setupDocumentServiceHarness(t *testing.T) *serviceHarness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	minioCtx, mc := sharedtestutil.SetupTestMinio(t)
	t.Cleanup(minioCtx.Cancel)

	fixtures := createTestFixtures(t, db, ctx)
	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			Endpoint:  mc.Endpoint(),
			AccessKey: mc.AccessKey(),
			SecretKey: mc.SecretKey(),
			Bucket:    mc.Bucket(),
			UseSSL:    false,
			AllowedMIMETypes: []string{
				"application/pdf",
				"text/plain",
				"image/jpeg",
				"image/png",
			},
			MaxFileSize:        50 * 1024 * 1024,
			PresignedURLExpiry: 15 * time.Minute,
		},
	}

	storageClient, err := minioadapter.New(minioadapter.Params{
		Config: cfg,
		Logger: logger,
	})
	require.NoError(t, err)

	cacheRepo := mocks.NewMockDocumentCacheRepository(t)

	sessionRepo := mocks.NewMockDocumentUploadSessionRepository(t)
	contentService := mocks.NewMockDocumentContentService(t)
	searchProjection := mocks.NewMockDocumentSearchProjectionService(t)
	// Every scenario in this suite uploads at least one document, and each
	// upload projects it into search.
	searchProjection.On("Upsert", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	searchProjection.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	docRepo := documentrepository.New(documentrepository.Params{
		DB:     conn,
		Logger: logger,
	})
	docTypeRepo := documenttyperepository.New(documenttyperepository.Params{
		DB:     conn,
		Logger: logger,
	})
	packetRuleRepo := documentpacketrulerepository.New(documentpacketrulerepository.Params{
		DB:     conn,
		Logger: logger,
	})

	encryption := encryptionservice.NewWithKeyManager(
		encryptionservice.NewLocalKeyManager("unit-test-encryption-key-with-at-least-32-bytes"),
	)

	service := documentservice.New(documentservice.Params{
		Logger:           logger,
		DB:               conn,
		Repo:             docRepo,
		PacketRuleRepo:   packetRuleRepo,
		DocumentTypeRepo: docTypeRepo,
		CacheRepo:        cacheRepo,
		SessionRepo:      sessionRepo,
		Storage:          storageClient,
		Validator: documentservice.NewValidator(
			documentservice.ValidatorParams{Config: cfg},
		),
		AuditService:         &mocks.NoopAuditService{},
		DocumentIntelligence: contentService,
		SearchProjection:     searchProjection,
		Config:               cfg,
		ThumbnailGenerator:   thumbnailservice.NewGenerator(),
		Encryption:           encryption,
	})

	return &serviceHarness{
		ctx: ctx,
		db:  db,
		tenantInfo: pagination.TenantInfo{
			OrgID:  fixtures.orgID,
			BuID:   fixtures.buID,
			UserID: fixtures.userID,
		},
		service:          service,
		documentRepo:     docRepo,
		documentTypeRepo: docTypeRepo,
		packetRuleRepo:   packetRuleRepo,
		storage:          storageClient,
		cacheRepo:        cacheRepo,
		sessionRepo:      sessionRepo,
		contentService:   contentService,
		searchProjection: searchProjection,
	}
}

type testFixtures struct {
	orgID  pulid.ID
	buID   pulid.ID
	userID pulid.ID
}

func createTestFixtures(t *testing.T, db *bun.DB, ctx context.Context) *testFixtures {
	t.Helper()

	data := seedtest.SeedFullTestData(t, ctx, db)

	return &testFixtures{
		orgID:  data.Organization.ID,
		buID:   data.BusinessUnit.ID,
		userID: data.User.ID,
	}
}

func createDocumentType(
	t *testing.T,
	h *serviceHarness,
	code, name string,
	category documenttype.DocumentCategory,
) *documenttype.DocumentType {
	t.Helper()

	entity, err := h.documentTypeRepo.Create(h.ctx, &documenttype.DocumentType{
		OrganizationID:         h.tenantInfo.OrgID,
		BusinessUnitID:         h.tenantInfo.BuID,
		Code:                   code,
		Name:                   name,
		DocumentClassification: documenttype.ClassificationPublic,
		DocumentCategory:       category,
	})
	require.NoError(t, err)
	return entity
}

func createPacketRule(
	t *testing.T,
	h *serviceHarness,
	resourceType string,
	docTypeID pulid.ID,
	required bool,
	expirationRequired bool,
	expirationWarningDays int,
) *documentpacketrule.DocumentPacketRule {
	t.Helper()

	rule, err := h.packetRuleRepo.Create(h.ctx, &documentpacketrule.DocumentPacketRule{
		OrganizationID:        h.tenantInfo.OrgID,
		BusinessUnitID:        h.tenantInfo.BuID,
		ResourceType:          resourceType,
		DocumentTypeID:        docTypeID,
		Required:              required,
		DisplayOrder:          10,
		ExpirationRequired:    expirationRequired,
		ExpirationWarningDays: expirationWarningDays,
	})
	require.NoError(t, err)
	return rule
}

func uploadDocument(
	t *testing.T,
	h *serviceHarness,
	filename string,
	resourceID string,
	resourceType string,
	documentTypeID string,
	lineageID string,
) *document.Document {
	t.Helper()

	fileHeader := storageutil.NewMockFileHeader(
		filename,
		[]byte("test "+filename),
		"application/pdf",
	)
	result, err := h.service.Upload(h.ctx, &documentservice.UploadRequest{
		TenantInfo:     h.tenantInfo,
		File:           fileHeader,
		ResourceID:     resourceID,
		ResourceType:   resourceType,
		DocumentTypeID: documentTypeID,
		LineageID:      lineageID,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Document)

	return result.Document
}

func updateDocumentExpiration(
	t *testing.T,
	h *serviceHarness,
	documentID pulid.ID,
	expiration *int64,
) {
	t.Helper()

	// The schema refuses to store an expiration in the past, which is right for
	// real writes but leaves no way to age a document. Dropping the check on the
	// throwaway test database lets the suite simulate elapsed time.
	_, err := h.db.ExecContext(
		h.ctx,
		`ALTER TABLE documents DROP CONSTRAINT IF EXISTS chk_documents_expiration_date`,
	)
	require.NoError(t, err)

	_, err = h.db.NewUpdate().
		Table("documents").
		Set("expiration_date = ?", expiration).
		Where("id = ?", documentID).
		Where("organization_id = ?", h.tenantInfo.OrgID).
		Where("business_unit_id = ?", h.tenantInfo.BuID).
		Exec(h.ctx)
	require.NoError(t, err)
}

func fetchDocument(t *testing.T, h *serviceHarness, id pulid.ID) *document.Document {
	t.Helper()

	entity, err := h.documentRepo.GetByID(h.ctx, repositories.GetDocumentByIDRequest{
		ID:         id,
		TenantInfo: h.tenantInfo,
	})
	require.NoError(t, err)
	return entity
}

func TestService_DocumentVersioningLifecycle_Integration(t *testing.T) {
	h := setupDocumentServiceHarness(t)
	// Versioning re-reads the current version, which misses the cache, and pulls
	// the previous version's extracted content to carry lineage metadata forward.
	h.cacheRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, repositories.ErrCacheMiss)
	h.contentService.On("GetContent", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	resourceID := pulid.MustNew("sh_").String()
	docType := createDocumentType(t, h, "BOL", "Bill of Lading", documenttype.CategoryShipment)

	firstVersion := uploadDocument(
		t,
		h,
		"v1.pdf",
		resourceID,
		"shipment",
		docType.ID.String(),
		"",
	)
	secondVersion := uploadDocument(
		t,
		h,
		"v2.pdf",
		resourceID,
		"shipment",
		docType.ID.String(),
		firstVersion.LineageID.String(),
	)

	require.Equal(t, firstVersion.LineageID, secondVersion.LineageID)
	require.EqualValues(t, 1, firstVersion.VersionNumber)
	require.EqualValues(t, 2, secondVersion.VersionNumber)

	versions, err := h.service.ListVersions(h.ctx, secondVersion.ID, h.tenantInfo)
	require.NoError(t, err)
	require.Len(t, versions, 2)
	assert.ElementsMatch(
		t,
		[]pulid.ID{firstVersion.ID, secondVersion.ID},
		[]pulid.ID{versions[0].ID, versions[1].ID},
	)

	refreshedFirstVersion := fetchDocument(t, h, firstVersion.ID)
	refreshedSecondVersion := fetchDocument(t, h, secondVersion.ID)
	assert.False(t, refreshedFirstVersion.IsCurrentVersion)
	assert.True(t, refreshedSecondVersion.IsCurrentVersion)
	assert.EqualValues(t, 1, refreshedFirstVersion.VersionNumber)
	assert.EqualValues(t, 2, refreshedSecondVersion.VersionNumber)

	resourceDocs, err := h.service.GetByResource(h.ctx, &repositories.GetDocumentsByResourceRequest{
		TenantInfo:   h.tenantInfo,
		ResourceID:   resourceID,
		ResourceType: "shipment",
	})
	require.NoError(t, err)
	require.Len(t, resourceDocs, 1)
	assert.Equal(t, secondVersion.ID, resourceDocs[0].ID)

	restored, err := h.service.RestoreVersion(
		h.ctx,
		firstVersion.ID,
		h.tenantInfo,
		h.tenantInfo.UserID,
	)
	require.NoError(t, err)
	assert.Equal(t, firstVersion.ID, restored.ID)
	assert.True(t, restored.IsCurrentVersion)

	restoredDocs, err := h.service.GetByResource(h.ctx, &repositories.GetDocumentsByResourceRequest{
		TenantInfo:   h.tenantInfo,
		ResourceID:   resourceID,
		ResourceType: "shipment",
	})
	require.NoError(t, err)
	require.Len(t, restoredDocs, 1)
	assert.Equal(t, firstVersion.ID, restoredDocs[0].ID)

	firstStoragePath := firstVersion.StoragePath
	secondStoragePath := secondVersion.StoragePath

	err = h.service.Delete(h.ctx, repositories.DeleteDocumentRequest{
		ID:         restored.ID,
		TenantInfo: h.tenantInfo,
	}, h.tenantInfo.UserID)
	require.NoError(t, err)

	_, err = h.service.Get(h.ctx, repositories.GetDocumentByIDRequest{
		ID:         firstVersion.ID,
		TenantInfo: h.tenantInfo,
	})
	require.Error(t, err)

	lineageCount, err := h.db.NewSelect().
		Table("documents").
		Where("lineage_id = ?", firstVersion.LineageID).
		Where("organization_id = ?", h.tenantInfo.OrgID).
		Where("business_unit_id = ?", h.tenantInfo.BuID).
		Count(h.ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, lineageCount)

	firstExists, err := h.storage.Exists(h.ctx, firstStoragePath)
	require.NoError(t, err)
	assert.False(t, firstExists)

	secondExists, err := h.storage.Exists(h.ctx, secondStoragePath)
	require.NoError(t, err)
	assert.False(t, secondExists)
}

func TestService_BulkDeleteDeletesLineages_Integration(t *testing.T) {
	h := setupDocumentServiceHarness(t)

	docType := createDocumentType(t, h, "POD", "Proof of Delivery", documenttype.CategoryShipment)

	resourceOne := pulid.MustNew("sh_").String()
	firstV1 := uploadDocument(
		t,
		h,
		"lineage-one-v1.pdf",
		resourceOne,
		"shipment",
		docType.ID.String(),
		"",
	)
	firstV2 := uploadDocument(
		t,
		h,
		"lineage-one-v2.pdf",
		resourceOne,
		"shipment",
		docType.ID.String(),
		firstV1.LineageID.String(),
	)

	resourceTwo := pulid.MustNew("sh_").String()
	secondV1 := uploadDocument(
		t,
		h,
		"lineage-two-v1.pdf",
		resourceTwo,
		"shipment",
		docType.ID.String(),
		"",
	)
	secondV2 := uploadDocument(
		t,
		h,
		"lineage-two-v2.pdf",
		resourceTwo,
		"shipment",
		docType.ID.String(),
		secondV1.LineageID.String(),
	)

	result, err := h.service.BulkDelete(h.ctx, &documentservice.BulkDeleteRequest{
		IDs:        []pulid.ID{firstV2.ID, secondV2.ID},
		TenantInfo: h.tenantInfo,
		UserID:     h.tenantInfo.UserID,
	})
	require.NoError(t, err)
	assert.Equal(t, 4, result.DeletedCount)

	for _, lineageID := range []pulid.ID{firstV1.LineageID, secondV1.LineageID} {
		count, countErr := h.db.NewSelect().
			Table("documents").
			Where("lineage_id = ?", lineageID).
			Where("organization_id = ?", h.tenantInfo.OrgID).
			Where("business_unit_id = ?", h.tenantInfo.BuID).
			Count(h.ctx)
		require.NoError(t, countErr)
		assert.Equal(t, 0, count)
	}
}

func TestService_GetPacketSummary_Integration(t *testing.T) {
	h := setupDocumentServiceHarness(t)

	missingDocType := createDocumentType(t, h, "INS", "Insurance", documenttype.CategoryRegulatory)
	needsReviewDocType := createDocumentType(
		t,
		h,
		"REG",
		"Registration",
		documenttype.CategoryRegulatory,
	)
	expiredDocType := createDocumentType(t, h, "PERM", "Permit", documenttype.CategoryRegulatory)
	expiringSoonDocType := createDocumentType(
		t,
		h,
		"ANNU",
		"Annual Inspection",
		documenttype.CategoryRegulatory,
	)
	currentOnlyDocType := createDocumentType(
		t,
		h,
		"BOL",
		"Bill of Lading",
		documenttype.CategoryShipment,
	)

	createPacketRule(
		t,
		h,
		"Trailer",
		missingDocType.ID,
		true,
		false,
		0,
	)
	createPacketRule(
		t,
		h,
		"Trailer",
		needsReviewDocType.ID,
		true,
		true,
		30,
	)
	createPacketRule(
		t,
		h,
		"Trailer",
		expiredDocType.ID,
		true,
		true,
		30,
	)
	createPacketRule(
		t,
		h,
		"Trailer",
		expiringSoonDocType.ID,
		true,
		true,
		30,
	)
	createPacketRule(
		t,
		h,
		"Trailer",
		currentOnlyDocType.ID,
		true,
		false,
		0,
	)

	resourceID := pulid.MustNew("tr_").String()
	uploadDocument(
		t,
		h,
		"registration.pdf",
		resourceID,
		"trailer",
		needsReviewDocType.ID.String(),
		"",
	)
	expiredDoc := uploadDocument(
		t,
		h,
		"permit.pdf",
		resourceID,
		"trailer",
		expiredDocType.ID.String(),
		"",
	)
	expiringSoonDoc := uploadDocument(
		t,
		h,
		"inspection.pdf",
		resourceID,
		"trailer",
		expiringSoonDocType.ID.String(),
		"",
	)
	currentV1 := uploadDocument(
		t,
		h,
		"bol-v1.pdf",
		resourceID,
		"trailer",
		currentOnlyDocType.ID.String(),
		"",
	)
	currentV2 := uploadDocument(
		t,
		h,
		"bol-v2.pdf",
		resourceID,
		"trailer",
		currentOnlyDocType.ID.String(),
		currentV1.LineageID.String(),
	)

	now := timeutils.NowUnix()
	expiredAt := now - int64(24*time.Hour.Seconds())
	expiringSoonAt := now + int64(7*24*time.Hour.Seconds())

	updateDocumentExpiration(t, h, expiredDoc.ID, &expiredAt)
	updateDocumentExpiration(t, h, expiringSoonDoc.ID, &expiringSoonAt)
	updateDocumentExpiration(t, h, currentV1.ID, &expiredAt)

	summary, err := h.service.GetPacketSummary(h.ctx, "trailer", resourceID, h.tenantInfo)
	require.NoError(t, err)

	assert.Equal(t, documentpacketrule.PacketStatusExpired, summary.Status)
	assert.Equal(t, 5, summary.TotalRules)
	assert.Equal(t, 1, summary.MissingRequired)
	assert.Equal(t, 1, summary.NeedsReview)
	assert.Equal(t, 1, summary.Expired)
	assert.Equal(t, 1, summary.ExpiringSoon)
	assert.Equal(t, 1, summary.SatisfiedRules)

	itemByType := make(map[pulid.ID]documentpacketrule.PacketItemSummary, len(summary.Items))
	for _, item := range summary.Items {
		itemByType[item.DocumentTypeID] = item
	}

	assert.Equal(t, documentpacketrule.ItemStatusMissing, itemByType[missingDocType.ID].Status)
	assert.Equal(
		t,
		documentpacketrule.ItemStatusNeedsReview,
		itemByType[needsReviewDocType.ID].Status,
	)
	assert.Equal(t, documentpacketrule.ItemStatusExpired, itemByType[expiredDocType.ID].Status)
	assert.Equal(
		t,
		documentpacketrule.ItemStatusExpiringSoon,
		itemByType[expiringSoonDocType.ID].Status,
	)
	assert.Equal(t, documentpacketrule.ItemStatusComplete, itemByType[currentOnlyDocType.ID].Status)
	assert.Equal(t, 1, itemByType[currentOnlyDocType.ID].DocumentCount)
	assert.Equal(t, []pulid.ID{currentV2.ID}, itemByType[currentOnlyDocType.ID].CurrentDocumentIDs)

	refreshedFirstVersion := fetchDocument(t, h, currentV1.ID)
	refreshedSecondVersion := fetchDocument(t, h, currentV2.ID)
	assert.False(t, refreshedFirstVersion.IsCurrentVersion)
	assert.True(t, refreshedSecondVersion.IsCurrentVersion)
}
