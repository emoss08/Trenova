package documentrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
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

	assert.Equal(t, []platformplanports.QuotaRequest{
		{TenantInfo: tenant, Meter: platformcatalog.MeterDocumentFileBytes, Quantity: 4096},
		{TenantInfo: tenant, Meter: platformcatalog.MeterDocumentUploads, Quantity: 1},
		{TenantInfo: tenant, Meter: platformcatalog.MeterDocumentStorageBytes, Quantity: 4096},
	}, documentQuotaRequests(entity))

	negative := documentQuotaRequests(quotaDocument(-1))
	assert.Zero(t, negative[0].Quantity)
	assert.Zero(t, negative[2].Quantity)
}
