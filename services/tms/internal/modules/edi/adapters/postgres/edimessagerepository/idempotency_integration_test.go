//go:build integration

package edimessagerepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimessagerepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEDIMessageIdempotencyKey_Integration(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	var err error

	scope := tenantWithoutPartners(t, ctx, db)
	ctx = scope.Ctx
	org := scope.Info
	documentType := findDocumentType(
		t,
		ctx,
		db,
		edi.TransactionSet204,
		edi.DocumentDirectionOutbound,
	)
	tenantInfo := org

	partner := &edi.EDIPartner{
		BusinessUnitID: org.BuID,
		OrganizationID: org.OrgID,
		Kind:           edi.PartnerKindExternal,
		Code:           "IDEMPOTENCY-TEST",
		Name:           "Idempotency Test Partner",
	}
	require.NoError(t, create(ctx, db, partner))

	repo := edimessagerepository.New(edimessagerepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	newMessage := func(key, control string) *edi.EDIMessage {
		return &edi.EDIMessage{
			BusinessUnitID:           org.BuID,
			OrganizationID:           org.OrgID,
			EDIPartnerID:             partner.ID,
			DocumentTypeID:           documentType.ID,
			Direction:                edi.DocumentDirectionOutbound,
			Standard:                 edi.EDIStandardX12,
			TransactionSet:           edi.TransactionSet204,
			X12Version:               edi.DefaultX12204Version,
			Status:                   edi.MessageStatusGenerated,
			RawX12:                   "ISA*TEST~",
			InterchangeControlNumber: "00000" + control,
			GroupControlNumber:       control,
			TransactionControlNumber: control,
			AckStatus:                edi.MessageAcknowledgmentStatusNotExpected,
			GeneratedAt:              timeutils.NowUnix(),
			IdempotencyKey:           key,
		}
	}

	first, err := repo.CreateMessageWithDiagnostics(
		ctx,
		edimessageports.CreateEDIMessageWithDiagnosticsRequest{
			Message: newMessage("tender-offer-204-tof_1", "1001"),
		},
	)
	require.NoError(t, err)

	found, err := repo.GetMessageByIdempotencyKey(
		ctx,
		edimessageports.GetEDIMessageByIdempotencyKeyRequest{
			TenantInfo:     tenantInfo,
			IdempotencyKey: "tender-offer-204-tof_1",
		},
	)
	require.NoError(t, err)
	assert.Equal(t, first.ID, found.ID)
	assert.Equal(t, "tender-offer-204-tof_1", found.IdempotencyKey)

	_, err = repo.CreateMessageWithDiagnostics(
		ctx,
		edimessageports.CreateEDIMessageWithDiagnosticsRequest{
			Message: newMessage("tender-offer-204-tof_1", "1002"),
		},
	)
	require.Error(t, err)
	assert.True(t, dberror.IsUniqueConstraintViolation(err))

	for _, control := range []string{"1003", "1004"} {
		_, err = repo.CreateMessageWithDiagnostics(
			ctx,
			edimessageports.CreateEDIMessageWithDiagnosticsRequest{
				Message: newMessage("", control),
			},
		)
		require.NoError(t, err, "messages without a key never collide")
	}

	_, err = repo.GetMessageByIdempotencyKey(
		ctx,
		edimessageports.GetEDIMessageByIdempotencyKeyRequest{
			TenantInfo:     tenantInfo,
			IdempotencyKey: "tender-offer-204-unknown",
		},
	)
	assert.True(t, errortypes.IsNotFoundError(err))
}
