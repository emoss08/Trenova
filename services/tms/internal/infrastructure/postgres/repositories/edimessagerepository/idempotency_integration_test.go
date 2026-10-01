//go:build integration

package edimessagerepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/edimessagerepository"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEDIMessageIdempotencyKey_Integration(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(
		db,
		registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	var org seededSummaryOrg
	require.NoError(t, db.NewSelect().
		Table("organizations").
		Column("id", "business_unit_id").
		Limit(1).
		Scan(ctx, &org))
	var documentType seededSummaryDocumentType
	require.NoError(t, db.NewSelect().
		Table("edi_document_types").
		Column("id").
		Where("transaction_set = ?", edi.TransactionSet204).
		Where("direction = ?", edi.DocumentDirectionOutbound).
		Limit(1).
		Scan(ctx, &documentType))
	tenantInfo := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}

	partner := &edi.EDIPartner{
		BusinessUnitID: org.BusinessUnitID,
		OrganizationID: org.ID,
		Kind:           edi.PartnerKindExternal,
		Code:           "IDEMPOTENCY-TEST",
		Name:           "Idempotency Test Partner",
	}
	_, err = db.NewInsert().Model(partner).Exec(ctx)
	require.NoError(t, err)

	repo := edimessagerepository.New(edimessagerepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	newMessage := func(key, control string) *edi.EDIMessage {
		return &edi.EDIMessage{
			BusinessUnitID:           org.BusinessUnitID,
			OrganizationID:           org.ID,
			EDIPartnerID:             partner.ID,
			DocumentTypeID:           documentType.ID,
			Direction:                edi.DocumentDirectionOutbound,
			Standard:                 edi.EDIStandardX12,
			TransactionSet:           edi.TransactionSet204,
			X12Version:               edi.DefaultX12204Version,
			Status:                   edi.MessageStatusGenerated,
			ValidationMode:           edi.ValidationModeDisabled,
			RawX12:                   "ISA*TEST~",
			InterchangeControlNumber: "00000" + control,
			GroupControlNumber:       control,
			TransactionControlNumber: control,
			AckStatus:                edi.MessageAcknowledgmentStatusNotExpected,
			GeneratedAt:              timeutils.NowUnix(),
			IdempotencyKey:           key,
		}
	}

	first, err := repo.CreateMessageWithDiagnostics(ctx, repositories.CreateEDIMessageWithDiagnosticsRequest{
		Message: newMessage("tender-offer-204-tof_1", "1001"),
	})
	require.NoError(t, err)

	found, err := repo.GetMessageByIdempotencyKey(ctx, repositories.GetEDIMessageByIdempotencyKeyRequest{
		TenantInfo:     tenantInfo,
		IdempotencyKey: "tender-offer-204-tof_1",
	})
	require.NoError(t, err)
	assert.Equal(t, first.ID, found.ID)
	assert.Equal(t, "tender-offer-204-tof_1", found.IdempotencyKey)

	_, err = repo.CreateMessageWithDiagnostics(ctx, repositories.CreateEDIMessageWithDiagnosticsRequest{
		Message: newMessage("tender-offer-204-tof_1", "1002"),
	})
	require.Error(t, err)
	assert.True(t, dberror.IsUniqueConstraintViolation(err))

	for _, control := range []string{"1003", "1004"} {
		_, err = repo.CreateMessageWithDiagnostics(ctx, repositories.CreateEDIMessageWithDiagnosticsRequest{
			Message: newMessage("", control),
		})
		require.NoError(t, err, "messages without a key never collide")
	}

	_, err = repo.GetMessageByIdempotencyKey(ctx, repositories.GetEDIMessageByIdempotencyKeyRequest{
		TenantInfo:     tenantInfo,
		IdempotencyKey: "tender-offer-204-unknown",
	})
	assert.True(t, errortypes.IsNotFoundError(err))
}
