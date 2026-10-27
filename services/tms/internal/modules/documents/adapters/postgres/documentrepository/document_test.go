//go:build integration

package documentrepository_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentrepository"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testFixtures struct {
	ctx    context.Context
	orgID  pulid.ID
	buID   pulid.ID
	userID pulid.ID
}

func fixturesOf(ctx context.Context, data *seedtest.TestData) *testFixtures {
	return &testFixtures{
		ctx:    data.Ctx(ctx),
		orgID:  data.Organization.ID,
		buID:   data.BusinessUnit.ID,
		userID: data.User.ID,
	}
}

func setupRepository(t *testing.T) (documentports.DocumentRepository, *testFixtures) {
	t.Helper()

	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	fixtures := fixturesOf(t.Context(), seedtest.SeedFullTestData(t, ctx, admin))
	conn := dbtest.NewRLSConnection(t, admin, dsn)

	return documentrepository.New(documentrepository.Params{DB: conn, Logger: zap.NewNop()}), fixtures
}

func createTestDocument(
	fixtures *testFixtures,
	opts ...func(*document.Document),
) *document.Document {
	now := timeutils.NowUnix()
	doc := &document.Document{
		ID:               pulid.MustNew("doc_"),
		OrganizationID:   fixtures.orgID,
		BusinessUnitID:   fixtures.buID,
		FileName:         "test-file.pdf",
		OriginalName:     "Original Test File.pdf",
		FileSize:         1024,
		FileType:         "application/pdf",
		StoragePath:      fixtures.orgID.String() + "/trailer/test-file.pdf",
		Status:           document.StatusActive,
		IsCurrentVersion: true,
		Description:      "Test document description",
		ResourceID:       pulid.MustNew("tr_").String(),
		ResourceType:     "trailer",
		Tags:             []string{"test", "document"},
		UploadedByID:     fixtures.userID,
		Version:          0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	for _, opt := range opts {
		opt(doc)
	}

	return doc
}

func TestDocumentRepository_Create_Integration(t *testing.T) {
	repo, fixtures := setupRepository(t)
	ctx := fixtures.ctx

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
	repo, fixtures := setupRepository(t)
	ctx := fixtures.ctx

	doc := createTestDocument(fixtures)
	_, err := repo.Create(ctx, doc)
	require.NoError(t, err)

	t.Run("get existing document", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, documentports.GetDocumentByIDRequest{
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
		_, err := repo.GetByID(ctx, documentports.GetDocumentByIDRequest{
			ID: pulid.MustNew("doc_"),
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		assert.Error(t, err)
	})

	t.Run("get document with wrong tenant", func(t *testing.T) {
		other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
		_, err := repo.GetByID(dbscope.WithTenant(t.Context(), other.DBTenant()), documentports.GetDocumentByIDRequest{
			ID:         doc.ID,
			TenantInfo: other,
		})
		assert.Error(t, err)
	})
}

func TestDocumentRepository_GetByResourceID_Integration(t *testing.T) {
	repo, fixtures := setupRepository(t)
	ctx := fixtures.ctx

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
		docs, err := repo.GetByResourceID(ctx, &documentports.GetDocumentsByResourceRequest{
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
		docs, err := repo.GetByResourceID(ctx, &documentports.GetDocumentsByResourceRequest{
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
	repo, fixtures := setupRepository(t)
	ctx := fixtures.ctx

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
		result, err := repo.List(ctx, &documentports.ListDocumentsRequest{
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
		result, err := repo.List(ctx, &documentports.ListDocumentsRequest{
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
		result, err := repo.List(ctx, &documentports.ListDocumentsRequest{
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
	repo, fixtures := setupRepository(t)
	ctx := fixtures.ctx

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
	repo, fixtures := setupRepository(t)
	ctx := fixtures.ctx

	doc := createTestDocument(fixtures)
	_, err := repo.Create(ctx, doc)
	require.NoError(t, err)

	t.Run("delete existing document", func(t *testing.T) {
		err := repo.Delete(ctx, documentports.DeleteDocumentRequest{
			ID: doc.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, documentports.GetDocumentByIDRequest{
			ID: doc.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: fixtures.orgID,
				BuID:  fixtures.buID,
			},
		})
		assert.Error(t, err)
	})

	t.Run("delete non-existent document", func(t *testing.T) {
		err := repo.Delete(ctx, documentports.DeleteDocumentRequest{
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
	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	fixtures1 := fixturesOf(t.Context(), seedtest.SeedFullTestData(t, ctx, admin))
	fixtures2 := fixturesOf(t.Context(), seedtest.SeedAdditionalTenant(t, ctx, admin, "TWO"))
	org1, bu1 := fixtures1.orgID, fixtures1.buID
	org2, bu2 := fixtures2.orgID, fixtures2.buID
	ctx1, ctx2 := fixtures1.ctx, fixtures2.ctx

	repo := documentrepository.New(documentrepository.Params{
		DB:     dbtest.NewRLSConnection(t, admin, dsn),
		Logger: zap.NewNop(),
	})

	doc1 := createTestDocument(fixtures1, func(d *document.Document) {
		d.FileName = "org1-doc.pdf"
	})
	doc2 := createTestDocument(fixtures2, func(d *document.Document) {
		d.FileName = "org2-doc.pdf"
	})

	_, err := repo.Create(ctx1, doc1)
	require.NoError(t, err)
	_, err = repo.Create(ctx2, doc2)
	require.NoError(t, err)

	t.Run("org1 can only see org1 documents", func(t *testing.T) {
		result, err := repo.List(ctx1, &documentports.ListDocumentsRequest{
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
		result, err := repo.List(ctx2, &documentports.ListDocumentsRequest{
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
		_, err := repo.GetByID(ctx1, documentports.GetDocumentByIDRequest{
			ID: doc2.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: org1,
				BuID:  bu1,
			},
		})
		assert.Error(t, err)
	})

	t.Run("org1 cannot delete org2 document", func(t *testing.T) {
		err := repo.Delete(ctx1, documentports.DeleteDocumentRequest{
			ID: doc2.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: org1,
				BuID:  bu1,
			},
		})
		assert.Error(t, err)
	})

	t.Run("org2 document survives org1's delete", func(t *testing.T) {
		kept, err := repo.GetByID(ctx2, documentports.GetDocumentByIDRequest{
			ID:         doc2.ID,
			TenantInfo: pagination.TenantInfo{OrgID: org2, BuID: bu2},
		})
		require.NoError(t, err)
		assert.Equal(t, doc2.ID, kept.ID)
	})
}
