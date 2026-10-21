package rateimportservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/contentscanportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpload_RefusesAnInfectedSheetBeforeReadingIt(t *testing.T) {
	t.Parallel()

	gate := contentscanportsmock.NewInfectedGate("Xls.Dropper.Agent-1")
	svc := &Service{l: zap.NewNop(), contentGate: gate}
	content := []byte("PK\x03\x04 not a workbook")

	batch, err := svc.Upload(t.Context(), &UploadRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		RateAgreementID: pulid.MustNew("ra_"),
		FileName:        "tariff.xlsx",
		Content:         content,
	})

	require.Error(t, err)
	assert.Nil(t, batch)
	assert.True(t, errortypes.IsError(err))
	assert.Contains(t, err.Error(), "Xls.Dropper.Agent-1")
	require.Len(t, gate.Requests(), 1)
	assert.Equal(t, content, gate.Requests()[0].Content)
}
