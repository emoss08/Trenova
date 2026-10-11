package ediservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/ediportsmock"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGenerateDocument_ReturnsTheDocumentAlreadyGeneratedForItsKey(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	key := TenderOfferDocumentKey(pulid.MustNew("tof_"))
	existing := &edi.EDIMessage{
		ID:             pulid.MustNew("emsg_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		TransactionSet: edi.TransactionSet204,
		EDIPartnerID:   pulid.MustNew("edip_"),
		Direction:      edi.DocumentDirectionOutbound,
		Status:         edi.MessageStatusGenerated,
		IdempotencyKey: key,
	}

	messages := ediportsmock.NewMockEDIMessageRepository(t)
	messages.EXPECT().GetMessageByIdempotencyKey(t.Context(), ediports.GetEDIMessageByIdempotencyKeyRequest{
		TenantInfo:     tenant,
		IdempotencyKey: key,
	}).Return(existing, nil).Once()

	partners := ediportsmock.NewMockEDIPartnerRepository(t)
	partners.EXPECT().GetByID(t.Context(), ediports.GetEDIPartnerByIDRequest{
		ID:         existing.EDIPartnerID,
		TenantInfo: tenant,
	}).Return(&edi.EDIPartner{Kind: edi.PartnerKindInternal}, nil).Once()

	svc := &Service{l: zap.NewNop(), messageRepo: messages, partnerRepo: partners}

	message, err := svc.GenerateDocument(t.Context(), &GenerateEDIDocumentRequest{
		TenantInfo:     tenant,
		TransactionSet: edi.TransactionSet204,
		Direction:      edi.DocumentDirectionOutbound,
		IdempotencyKey: key,
	})

	require.NoError(t, err)
	assert.Equal(t, existing.ID, message.ID)
}

func TestGenerateDocument_RefusesAKeyThatNamesAnotherTransactionSet(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	messages := ediportsmock.NewMockEDIMessageRepository(t)
	messages.EXPECT().GetMessageByIdempotencyKey(t.Context(), ediports.GetEDIMessageByIdempotencyKeyRequest{
		TenantInfo:     tenant,
		IdempotencyKey: "shared-key",
	}).Return(&edi.EDIMessage{TransactionSet: edi.TransactionSet214}, nil).Once()

	svc := &Service{l: zap.NewNop(), messageRepo: messages}

	_, err := svc.GenerateDocument(t.Context(), &GenerateEDIDocumentRequest{
		TenantInfo:     tenant,
		TransactionSet: edi.TransactionSet204,
		IdempotencyKey: "shared-key",
	})

	assert.True(t, errortypes.IsBusinessError(err))
}
