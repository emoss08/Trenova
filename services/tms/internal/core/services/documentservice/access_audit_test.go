package documentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type accessAuditCapture struct {
	services.AuditService

	params *services.LogActionParams
	entry  *audit.Entry
}

func (c *accessAuditCapture) LogAction(params *services.LogActionParams, opts ...services.LogOption) error {
	c.params = params
	c.entry = &audit.Entry{}
	for _, opt := range opts {
		if err := opt(c.entry); err != nil {
			return err
		}
	}
	return nil
}

func TestRecordAccessAuditsTheReaderAndTheRequest(t *testing.T) {
	t.Parallel()

	capture := &accessAuditCapture{}
	svc := &Service{auditService: capture, l: zap.NewNop()}
	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	doc := &document.Document{
		ID:             pulid.MustNew("doc_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		OriginalName:   "bol.pdf",
		ResourceType:   "shipment",
		ResourceID:     "shp_1",
	}
	ctx := requestmeta.With(t.Context(), requestmeta.New("req-3", "198.51.100.20", "browser"))

	svc.recordAccess(ctx, tenantInfo, doc, documentAccessDownload, documentChannelContent)

	require.NotNil(t, capture.params)
	assert.Equal(t, permission.ResourceDocument, capture.params.Resource)
	assert.Equal(t, permission.OpRead, capture.params.Operation)
	assert.Equal(t, doc.ID.String(), capture.params.ResourceID)
	assert.Equal(t, tenantInfo.UserID, capture.params.UserID)
	assert.Equal(t, services.PrincipalTypeUser, capture.params.PrincipalType)
	assert.False(t, capture.params.Critical)
	assert.Equal(t, "Document downloaded", capture.entry.Comment)
	assert.Equal(t, "198.51.100.20", capture.entry.IPAddress)
	assert.Equal(t, "browser", capture.entry.UserAgent)
	assert.Equal(t, "req-3", capture.entry.CorrelationID)
	assert.Equal(t, "download", capture.entry.Metadata["access"])
	assert.Equal(t, "content", capture.entry.Metadata["channel"])
	assert.Equal(t, "bol.pdf", capture.entry.Metadata["fileName"])
}

func TestRecordAccessWithoutAUserIsTheSystem(t *testing.T) {
	t.Parallel()

	capture := &accessAuditCapture{}
	svc := &Service{auditService: capture, l: zap.NewNop()}
	doc := &document.Document{
		ID:             pulid.MustNew("doc_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	svc.recordAccess(t.Context(), pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID},
		doc, documentAccessView, documentChannelPresignedURL)

	require.NotNil(t, capture.params)
	assert.Equal(t, services.PrincipalTypeSystem, capture.params.PrincipalType)
	assert.True(t, capture.params.UserID.IsNil())
	assert.Equal(t, "Document viewed", capture.entry.Comment)
}
