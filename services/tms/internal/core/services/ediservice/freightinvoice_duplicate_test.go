package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func duplicateInvoiceFixture() (*edi.EDIPartner, *edi.EDIMessage, *edi.FreightInvoicePayload) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	partner := &edi.EDIPartner{
		ID:             pulid.MustNew("edip_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           "ACME",
	}
	message := &edi.EDIMessage{ID: pulid.MustNew("edim_"), OrganizationID: orgID, BusinessUnitID: buID}
	payload := &edi.FreightInvoicePayload{
		InvoiceNumber:    "inv-1 ",
		TotalAmount:      decimal.NullDecimal{Decimal: decimal.NewFromInt(1600), Valid: true},
		ReferenceNumbers: map[string]string{},
	}
	return partner, message, payload
}

func TestRecordInboundFreightInvoice_ReturnsTheInvoiceAlreadyRecorded(t *testing.T) {
	t.Parallel()

	partner, message, payload := duplicateInvoiceFixture()
	recorded := &edi.CarrierInvoice{
		ID:            pulid.MustNew("edici_"),
		InvoiceNumber: "INV-1",
		TotalAmount:   decimal.NullDecimal{Decimal: decimal.NewFromInt(1500), Valid: true},
	}
	invoiceRepo := mocks.NewMockEDICarrierInvoiceRepository(t)
	invoiceRepo.EXPECT().GetCarrierInvoiceByNumber(mock.Anything, &repositories.GetEDICarrierInvoiceByNumberRequest{
		TenantInfo:    pagination.TenantInfo{OrgID: message.OrganizationID, BuID: message.BusinessUnitID},
		PartnerID:     partner.ID,
		InvoiceNumber: "inv-1 ",
	}).Return(recorded, nil).Once()
	service := &Service{l: zap.NewNop(), carrierInvoiceRepo: invoiceRepo}

	result, err := service.RecordInboundFreightInvoice(t.Context(), &RecordInboundFreightInvoiceRequest{
		Partner: partner,
		Message: message,
		Payload: payload,
	})

	require.NoError(t, err)
	assert.True(t, result.AlreadyRecorded)
	assert.Equal(t, recorded.ID, result.Invoice.ID)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "already recorded")
	assert.Contains(t, result.Warnings[0], "recorded 1500.00, resent 1600.00")
	invoiceRepo.AssertNotCalled(t, "CreateCarrierInvoice", mock.Anything, mock.Anything)
}

func TestRecordInboundFreightInvoice_LosingTheInsertRaceReturnsTheWinner(t *testing.T) {
	t.Parallel()

	partner, message, payload := duplicateInvoiceFixture()
	winner := &edi.CarrierInvoice{
		ID:            pulid.MustNew("edici_"),
		InvoiceNumber: "INV-1",
		TotalAmount:   payload.TotalAmount,
	}
	invoiceRepo := mocks.NewMockEDICarrierInvoiceRepository(t)
	invoiceRepo.EXPECT().GetCarrierInvoiceByNumber(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("CarrierInvoice not found")).Once()
	invoiceRepo.EXPECT().CreateCarrierInvoice(mock.Anything, mock.Anything).
		Return(nil, &pgconn.PgError{Code: "23505"}).Once()
	invoiceRepo.EXPECT().GetCarrierInvoiceByNumber(mock.Anything, mock.Anything).
		Return(winner, nil).Once()
	service := &Service{l: zap.NewNop(), carrierInvoiceRepo: invoiceRepo}

	result, err := service.RecordInboundFreightInvoice(t.Context(), &RecordInboundFreightInvoiceRequest{
		Partner: partner,
		Message: message,
		Payload: payload,
	})

	require.NoError(t, err)
	assert.True(t, result.AlreadyRecorded)
	assert.Equal(t, winner.ID, result.Invoice.ID)
	assert.NotContains(t, result.Warnings[0], "differs")
}
