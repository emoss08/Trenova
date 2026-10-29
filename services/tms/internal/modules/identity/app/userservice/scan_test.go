package userservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil/mocks/contentscanportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	storageutil "github.com/emoss08/trenova/shared/testutil/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUploadProfilePicture_RefusesAFileTheScannerCannotCheck(t *testing.T) {
	t.Parallel()

	users := tenantportsmock.NewMockUserRepository(t)
	users.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&tenant.User{}, nil).Maybe()
	gate := contentscanportsmock.NewRefusingGate(contentscanports.ErrScannerUnavailable)
	svc := &Service{
		l:           zap.NewNop(),
		repo:        users,
		storage:     kernelportsmock.NewMockClient(t),
		storageCfg:  &storageconfig.Config{MaxFileSize: 1 << 20},
		contentGate: gate,
	}
	content := []byte("\x89PNG\r\n\x1a\nportrait")

	user, err := svc.UploadProfilePicture(
		t.Context(),
		pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		storageutil.NewMockFileHeader("me.png", content, "image/png"),
	)

	require.ErrorIs(t, err, contentscanports.ErrScannerUnavailable)
	assert.Nil(t, user)
	require.Len(t, gate.Requests(), 1)
	assert.Equal(t, content, gate.Requests()[0].Content)
}
