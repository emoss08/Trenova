package driverportalresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/driverportalservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documenttypeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/portalaccessportsmock"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type portalDocumentsFixture struct {
	resolver  *QueryResolver
	portal    *portalaccessportsmock.MockPortalAccessRepository
	docTypes  *documenttypeportsmock.MockDocumentTypeRepository
	documents *documentportsmock.MockDocumentRepository
	auth      *authctx.AuthContext
}

func newPortalDocumentsFixture(t *testing.T) *portalDocumentsFixture {
	t.Helper()

	portal := portalaccessportsmock.NewMockPortalAccessRepository(t)
	docTypes := documenttypeportsmock.NewMockDocumentTypeRepository(t)
	documents := documentportsmock.NewMockDocumentRepository(t)

	service := driverportalservice.New(driverportalservice.Params{
		Logger:           zap.NewNop(),
		PortalRepo:       portal,
		DocumentTypeRepo: docTypes,
		DocumentService: documentservice.New(documentservice.Params{
			Logger: zap.NewNop(),
			Repo:   documents,
		}),
	})

	userID := pulid.MustNew("usr_")
	return &portalDocumentsFixture{
		resolver: &QueryResolver{&Deps{
			Core:                &resolvercore.Core{},
			DriverPortalService: service,
		}},
		portal:    portal,
		docTypes:  docTypes,
		documents: documents,
		auth: &authctx.AuthContext{
			PrincipalType:  authctx.PrincipalTypeUser,
			PrincipalID:    userID,
			UserID:         userID,
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
	}
}

func (f *portalDocumentsFixture) ctx(t *testing.T) context.Context {
	t.Helper()
	return gqlctx.WithAuthContext(t.Context(), f.auth)
}

func (f *portalDocumentsFixture) tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  f.auth.OrganizationID,
		BuID:   f.auth.BusinessUnitID,
		UserID: f.auth.UserID,
	}
}

func (f *portalDocumentsFixture) expectWorker(workerID pulid.ID) {
	f.portal.EXPECT().
		GetWorkerByUserID(mock.Anything, f.tenantInfo()).
		Return(&worker.Worker{ID: workerID}, nil).
		Once()
}

func portalDocumentCalls(shipmentID string) map[string]func(context.Context, *QueryResolver) error {
	return map[string]func(context.Context, *QueryResolver) error{
		"myLoadDocuments": func(ctx context.Context, r *QueryResolver) error {
			_, err := r.MyLoadDocuments(ctx, shipmentID)
			return err
		},
		"myLoadDocumentTypes": func(ctx context.Context, r *QueryResolver) error {
			_, err := r.MyLoadDocumentTypes(ctx)
			return err
		},
		"myProfileDocuments": func(ctx context.Context, r *QueryResolver) error {
			_, err := r.MyProfileDocuments(ctx)
			return err
		},
		"myProfileDocumentTypes": func(ctx context.Context, r *QueryResolver) error {
			_, err := r.MyProfileDocumentTypes(ctx)
			return err
		},
	}
}

func TestPortalDocumentQueries_RequireASignedInUser(t *testing.T) {
	t.Parallel()

	for name, call := range portalDocumentCalls(pulid.MustNew("shp_").String()) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newPortalDocumentsFixture(t)
			err := call(t.Context(), f.resolver)
			require.Error(t, err)
			assert.True(t, errortypes.IsAuthenticationError(err))
		})
	}
}

func TestPortalDocumentQueries_RefuseAgents(t *testing.T) {
	t.Parallel()

	for name, call := range portalDocumentCalls(pulid.MustNew("shp_").String()) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newPortalDocumentsFixture(t)
			f.auth.PrincipalType = authctx.PrincipalTypeAgent
			err := call(f.ctx(t), f.resolver)
			require.Error(t, err)
			assert.True(t, errortypes.IsAuthorizationError(err))
		})
	}
}

func TestPortalDocumentQueries_RefuseUsersWithoutADriverProfile(t *testing.T) {
	t.Parallel()

	for name, call := range portalDocumentCalls(pulid.MustNew("shp_").String()) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newPortalDocumentsFixture(t)
			f.portal.EXPECT().
				GetWorkerByUserID(mock.Anything, f.tenantInfo()).
				Return(nil, errortypes.NewNotFoundError("worker not found")).
				Once()

			err := call(f.ctx(t), f.resolver)
			require.Error(t, err)
			assert.True(t, errortypes.IsAuthorizationError(err))
		})
	}
}

func TestMyLoadDocuments_RejectsAnInvalidShipmentID(t *testing.T) {
	t.Parallel()

	f := newPortalDocumentsFixture(t)
	documents, err := f.resolver.MyLoadDocuments(f.ctx(t), "not-an-id")
	require.Error(t, err)
	assert.Nil(t, documents)

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "shipmentId", validation.Field)
}

func TestMyLoadDocuments_HidesLoadsNotAssignedToTheDriver(t *testing.T) {
	t.Parallel()

	f := newPortalDocumentsFixture(t)
	workerID := pulid.MustNew("wrk_")
	shipmentID := pulid.MustNew("shp_")
	f.expectWorker(workerID)
	f.portal.EXPECT().
		WorkerAssignedToShipment(mock.Anything, f.tenantInfo(), workerID, shipmentID).
		Return(false, nil).
		Once()

	documents, err := f.resolver.MyLoadDocuments(f.ctx(t), shipmentID.String())
	require.Error(t, err)
	assert.Nil(t, documents)
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestMyLoadDocuments_ListsTheLoadsLiveDocuments(t *testing.T) {
	t.Parallel()

	f := newPortalDocumentsFixture(t)
	workerID := pulid.MustNew("wrk_")
	shipmentID := pulid.MustNew("shp_")
	docID := pulid.MustNew("doc_")
	f.expectWorker(workerID)
	f.portal.EXPECT().
		WorkerAssignedToShipment(mock.Anything, f.tenantInfo(), workerID, shipmentID).
		Return(true, nil).
		Once()
	f.documents.EXPECT().
		GetByResourceID(mock.Anything, &documentports.GetDocumentsByResourceRequest{
			TenantInfo:          f.tenantInfo(),
			ResourceID:          shipmentID.String(),
			ResourceType:        "shipment",
			IncludeDocumentType: true,
		}).
		Return([]*document.Document{
			{
				ID:           docID,
				FileName:     "stored.pdf",
				OriginalName: "pod.pdf",
				FileSize:     2048,
				Status:       document.StatusActive,
				CreatedAt:    1_800_000_000,
				DocumentType: &documenttype.DocumentType{Name: "Proof of Delivery"},
			},
			{ID: pulid.MustNew("doc_"), FileName: "old.pdf", Status: document.StatusArchived},
		}, nil).
		Once()

	documents, err := f.resolver.MyLoadDocuments(f.ctx(t), shipmentID.String())
	require.NoError(t, err)
	assert.Equal(t, []*driverportalservice.PortalDocument{{
		ID:               docID,
		FileName:         "pod.pdf",
		FileSize:         2048,
		Status:           string(document.StatusActive),
		DocumentTypeName: "Proof of Delivery",
		CreatedAt:        1_800_000_000,
	}}, documents)
}

func TestMyProfileDocuments_ListsTheDriversOwnDocuments(t *testing.T) {
	t.Parallel()

	f := newPortalDocumentsFixture(t)
	workerID := pulid.MustNew("wrk_")
	docID := pulid.MustNew("doc_")
	f.expectWorker(workerID)
	f.documents.EXPECT().
		GetByResourceID(mock.Anything, &documentports.GetDocumentsByResourceRequest{
			TenantInfo:          f.tenantInfo(),
			ResourceID:          workerID.String(),
			ResourceType:        "worker",
			IncludeDocumentType: true,
		}).
		Return([]*document.Document{
			{ID: docID, FileName: "cdl.jpg", FileSize: 512, Status: document.StatusPending},
		}, nil).
		Once()

	documents, err := f.resolver.MyProfileDocuments(f.ctx(t))
	require.NoError(t, err)
	assert.Equal(t, []*driverportalservice.PortalDocument{{
		ID:       docID,
		FileName: "cdl.jpg",
		FileSize: 512,
		Status:   string(document.StatusPending),
	}}, documents)
}

func TestPortalDocumentTypeQueries_ReturnOnlyTheirCategory(t *testing.T) {
	t.Parallel()

	shipmentType := &documenttype.DocumentType{
		ID:               pulid.MustNew("dt_"),
		Code:             "POD",
		Name:             "Proof of Delivery",
		Color:            "#16a34a",
		DocumentCategory: documenttype.CategoryShipment,
	}
	workerType := &documenttype.DocumentType{
		ID:               pulid.MustNew("dt_"),
		Code:             "CDL",
		Name:             "Commercial Driver's License",
		Color:            "#2563eb",
		DocumentCategory: documenttype.CategoryWorker,
	}
	otherType := &documenttype.DocumentType{
		ID:               pulid.MustNew("dt_"),
		Code:             "INV",
		Name:             "Invoice",
		DocumentCategory: documenttype.CategoryInvoice,
	}

	cases := []struct {
		name string
		call func(context.Context, *QueryResolver) ([]*driverportalservice.PortalDocumentType, error)
		want *documenttype.DocumentType
	}{
		{
			name: "myLoadDocumentTypes",
			call: func(
				ctx context.Context,
				r *QueryResolver,
			) ([]*driverportalservice.PortalDocumentType, error) {
				return r.MyLoadDocumentTypes(ctx)
			},
			want: shipmentType,
		},
		{
			name: "myProfileDocumentTypes",
			call: func(
				ctx context.Context,
				r *QueryResolver,
			) ([]*driverportalservice.PortalDocumentType, error) {
				return r.MyProfileDocumentTypes(ctx)
			},
			want: workerType,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newPortalDocumentsFixture(t)
			f.expectWorker(pulid.MustNew("wrk_"))
			f.docTypes.EXPECT().
				List(mock.Anything, &documenttypeports.ListDocumentTypesRequest{
					Filter: &pagination.QueryOptions{
						TenantInfo: f.tenantInfo(),
						Pagination: pagination.Info{Limit: 100},
					},
				}).
				Return(&pagination.ListResult[*documenttype.DocumentType]{
					Items: []*documenttype.DocumentType{shipmentType, workerType, otherType},
				}, nil).
				Once()

			types, err := tc.call(f.ctx(t), f.resolver)
			require.NoError(t, err)
			assert.Equal(t, []*driverportalservice.PortalDocumentType{{
				ID:    tc.want.ID,
				Code:  tc.want.Code,
				Name:  tc.want.Name,
				Color: tc.want.Color,
			}}, types)
		})
	}
}
