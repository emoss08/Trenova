package ediservice

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/testutil/mocks/ediportsmock"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestValidatePartnerSettings_ASetWithNoSchemaHasNothingToCheck(t *testing.T) {
	t.Parallel()

	settings := ediportsmock.NewMockEDIPartnerSettingRepository(t)
	settings.EXPECT().
		GetActivePartnerSettingSchema(mock.Anything, mock.Anything).
		Return(nil, dberror.HandleNotFoundError(sql.ErrNoRows, "EDIPartnerSettingSchema")).
		Once()
	service := New(Params{Logger: zap.NewNop(), PartnerSettingRepo: settings})

	diagnostics, err := service.ValidatePartnerSettings(t.Context(), &ValidatePartnerSettingsRequest{
		TenantInfo:     pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		DocumentTypeID: pulid.MustNew("edidt_"),
		Standard:       edi.EDIStandardX12,
		TransactionSet: edi.TransactionSet990,
		Direction:      edi.DocumentDirectionOutbound,
		Settings:       map[string]any{"carrier": map[string]any{"scac": "TRNV"}},
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)
}
