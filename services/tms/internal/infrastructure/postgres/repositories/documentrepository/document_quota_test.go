package documentrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func quotaDocument(size int64) *document.Document {
	return &document.Document{
		ID:             pulid.MustNew("doc_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		FileName:       "bol.pdf",
		FileSize:       size,
	}
}

func TestDocumentQuotaRequestsCoverFileSizeCountAndStorage(t *testing.T) {
	t.Parallel()

	entity := quotaDocument(4096)
	tenant := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}

	assert.Equal(t, []services.QuotaRequest{
		{TenantInfo: tenant, Meter: platformcatalog.MeterDocumentFileBytes, Quantity: 4096},
		{TenantInfo: tenant, Meter: platformcatalog.MeterDocumentUploads, Quantity: 1},
		{TenantInfo: tenant, Meter: platformcatalog.MeterDocumentStorageBytes, Quantity: 4096},
	}, documentQuotaRequests(entity))

	negative := documentQuotaRequests(quotaDocument(-1))
	assert.Zero(t, negative[0].Quantity)
	assert.Zero(t, negative[2].Quantity)
}

func TestCreateRefusesADocumentPastTheStorageQuota(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	guard := mocks.NewMockQuotaGuard(t)
	repo := &repository{db: conn, l: zap.NewNop(), quota: guard}
	entity := quotaDocument(2048)
	requests := documentQuotaRequests(entity)

	guard.EXPECT().Enforce(mock.Anything, &requests[0]).Return(nil).Once()
	guard.EXPECT().Enforce(mock.Anything, &requests[1]).Return(nil).Once()
	guard.EXPECT().Enforce(mock.Anything, &requests[2]).Return(errortypes.NewQuotaExceededError(
		string(platformcatalog.MeterDocumentStorageBytes), 100, 100, "free_demo",
	)).Once()
	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()

	created, err := repo.Create(t.Context(), entity)

	require.Nil(t, created)
	require.True(t, errortypes.IsQuotaExceededError(err))
	require.NoError(t, sqlMock.ExpectationsWereMet())
}
