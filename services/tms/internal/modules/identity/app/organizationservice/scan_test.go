package organizationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks/contentscanportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	storageutil "github.com/emoss08/trenova/shared/testutil/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUploadLogo_RefusesAnInfectedFileBeforeItIsStored(t *testing.T) {
	t.Parallel()

	gate := contentscanportsmock.NewInfectedGate("Png.Exploit.CVE_2026-1")
	svc := &service{
		l:           zap.NewNop(),
		repo:        tenantportsmock.NewMockOrganizationRepository(t),
		storage:     kernelportsmock.NewMockClient(t),
		storageCfg:  &storageconfig.Config{MaxFileSize: 1 << 20},
		contentGate: gate,
	}
	content := []byte("\x89PNG\r\n\x1a\ninfected")
	orgID := pulid.MustNew("org_")

	org, err := svc.UploadLogo(t.Context(), &tenantports.UploadLogoRequest{
		TenantInfo:     pagination.TenantInfo{OrgID: orgID, BuID: pulid.MustNew("bu_")},
		OrganizationID: orgID,
		File:           storageutil.NewMockFileHeader("logo.png", content, "image/png"),
	}, pulid.MustNew("usr_"))

	require.Error(t, err)
	assert.Nil(t, org)
	assert.True(t, errortypes.IsError(err))
	require.Len(t, gate.Requests(), 1)
	assert.Equal(t, content, gate.Requests()[0].Content)
}
