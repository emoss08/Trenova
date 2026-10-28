package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/edimessageportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/edipartnerportsmock"
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
	existing := &edimessage.EDIMessage{
		ID:             pulid.MustNew("emsg_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		TransactionSet: edidocument.TransactionSet204,
		EDIPartnerID:   pulid.MustNew("edip_"),
		Direction:      edidocument.DocumentDirectionOutbound,
		Status:         edimessage.MessageStatusGenerated,
		IdempotencyKey: key,
	}

	messages := edimessageportsmock.NewMockEDIMessageRepository(t)
	messages.EXPECT().
		GetMessageByIdempotencyKey(t.Context(), edimessageports.GetEDIMessageByIdempotencyKeyRequest{
			TenantInfo:     tenant,
			IdempotencyKey: key,
		}).
		Return(existing, nil).
		Once()

	partners := edipartnerportsmock.NewMockEDIPartnerRepository(t)
	partners.EXPECT().GetByID(t.Context(), edipartnerports.GetEDIPartnerByIDRequest{
		ID:         existing.EDIPartnerID,
		TenantInfo: tenant,
	}).Return(&edi.EDIPartner{Kind: edi.PartnerKindInternal}, nil).Once()

	svc := &Service{l: zap.NewNop(), messageRepo: messages, partnerRepo: partners}

	message, err := svc.GenerateDocument(t.Context(), &GenerateEDIDocumentRequest{
		TenantInfo:     tenant,
		TransactionSet: edidocument.TransactionSet204,
		Direction:      edidocument.DocumentDirectionOutbound,
		IdempotencyKey: key,
	})

	require.NoError(t, err)
	assert.Equal(t, existing.ID, message.ID)
}

func TestGenerateDocument_RefusesAKeyThatNamesAnotherTransactionSet(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	messages := edimessageportsmock.NewMockEDIMessageRepository(t)
	messages.EXPECT().
		GetMessageByIdempotencyKey(t.Context(), edimessageports.GetEDIMessageByIdempotencyKeyRequest{
			TenantInfo:     tenant,
			IdempotencyKey: "shared-key",
		}).
		Return(&edimessage.EDIMessage{TransactionSet: edidocument.TransactionSet214}, nil).
		Once()

	svc := &Service{l: zap.NewNop(), messageRepo: messages}

	_, err := svc.GenerateDocument(t.Context(), &GenerateEDIDocumentRequest{
		TenantInfo:     tenant,
		TransactionSet: edidocument.TransactionSet204,
		IdempotencyKey: "shared-key",
	})

	assert.True(t, errortypes.IsBusinessError(err))
}
