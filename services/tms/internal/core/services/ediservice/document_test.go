package ediservice

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResolvePayloadInvoiceIDResolves210Payload(t *testing.T) {
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	invoiceID := pulid.MustNew("inv_")
	shipmentID := pulid.MustNew("sp_")
	invoiceRepo := mocks.NewMockInvoiceRepository(t)
	invoiceRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceByIDRequest{
			ID:         invoiceID,
			TenantInfo: tenantInfo,
		}).
		Return(&invoice.Invoice{
			ID:           invoiceID,
			ShipmentID:   shipmentID,
			Number:       "INV-210",
			CurrencyCode: "USD",
			TotalAmount:  decimal.NewFromInt(100),
			Lines: []*invoice.InoviceLine{
				{LineNumber: 1, Type: invoice.InvoiceLineTypeFreight, Amount: decimal.NewFromInt(100)},
			},
		}, nil).
		Once()

	svc := &Service{invoiceRepo: invoiceRepo}
	payload, err := svc.resolvePayload(t.Context(), &PreviewEDIDocumentRequest{
		TenantInfo: tenantInfo,
		InvoiceID:  invoiceID,
	}, &edi.EDIPartnerDocumentProfile{TransactionSet: edi.TransactionSet210})

	require.NoError(t, err)
	require.NotNil(t, payload.FreightInvoice)
	require.Equal(t, edi.TransactionSet210, payload.TransactionSet)
	require.Equal(t, invoiceID, payload.FreightInvoice.InvoiceID)
	require.Equal(t, shipmentID, payload.FreightInvoice.ShipmentID)
}

func TestResolvePayloadShipmentEventIDResolves214Payload(t *testing.T) {
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	eventID := pulid.MustNew("se_")
	shipmentID := pulid.MustNew("sp_")
	eventRepo := mocks.NewMockShipmentEventRepository(t)
	shipmentSvc := mocks.NewMockShipmentService(t)
	eventRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetShipmentEventByIDRequest{
			ID:         eventID,
			TenantInfo: tenantInfo,
		}).
		Return(&shipmentevent.Event{
			ID:         eventID,
			ShipmentID: shipmentID,
			Type:       shipmentevent.TypeMoveDeparted,
			OccurredAt: 1715817600,
		}, nil).
		Once()
	shipmentSvc.EXPECT().
		Get(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == shipmentID && req.ExpandShipmentDetails
		})).
		Return(&shipment.Shipment{ID: shipmentID, BOL: "BOL-214"}, nil).
		Once()

	svc := &Service{shipmentEventRepo: eventRepo, shipmentSvc: shipmentSvc}
	payload, err := svc.resolvePayload(t.Context(), &PreviewEDIDocumentRequest{
		TenantInfo:      tenantInfo,
		ShipmentEventID: eventID,
	}, &edi.EDIPartnerDocumentProfile{TransactionSet: edi.TransactionSet214})

	require.NoError(t, err)
	require.NotNil(t, payload.ShipmentStatus)
	require.Equal(t, edi.TransactionSet214, payload.TransactionSet)
	require.Equal(t, shipmentID, payload.ShipmentStatus.ShipmentID)
	require.Equal(t, "AF", payload.ShipmentStatus.StatusCode)
}

func TestResolvePayloadRejectsSourceTransactionMismatch(t *testing.T) {
	svc := &Service{}
	_, err := svc.resolvePayload(t.Context(), &PreviewEDIDocumentRequest{
		InvoiceID: pulid.MustNew("inv_"),
	}, &edi.EDIPartnerDocumentProfile{TransactionSet: edi.TransactionSet204})

	require.Error(t, err)
	require.Contains(t, err.Error(), "invoice")
	require.Contains(t, err.Error(), "210")
}

func TestResolvePayloadRejectsShipmentEventMismatch(t *testing.T) {
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	eventID := pulid.MustNew("se_")
	eventShipmentID := pulid.MustNew("sp_")
	requestShipmentID := pulid.MustNew("sp_")
	eventRepo := mocks.NewMockShipmentEventRepository(t)
	eventRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetShipmentEventByIDRequest{
			ID:         eventID,
			TenantInfo: tenantInfo,
		}).
		Return(&shipmentevent.Event{
			ID:         eventID,
			ShipmentID: eventShipmentID,
			Type:       shipmentevent.TypeMoveDeparted,
		}, nil).
		Once()

	svc := &Service{shipmentEventRepo: eventRepo}
	_, err := svc.resolvePayload(t.Context(), &PreviewEDIDocumentRequest{
		TenantInfo:      tenantInfo,
		ShipmentID:      requestShipmentID,
		ShipmentEventID: eventID,
	}, &edi.EDIPartnerDocumentProfile{TransactionSet: edi.TransactionSet214})

	require.Error(t, err)
	require.Contains(t, err.Error(), "Shipment ID must match")
}

func TestUpsertPartnerDocumentProfile_AllowsInactiveProfileWithoutActiveTemplateVersion(
	t *testing.T,
) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	templateID := pulid.MustNew("editpl_")
	versionID := pulid.MustNew("editplv_")
	documentTypeID := pulid.MustNew("edidt_")
	partnerID := pulid.MustNew("edip_")
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		GetActiveTemplateVersion(mock.Anything, repositories.GetActiveEDITemplateVersionRequest{
			TemplateID: templateID,
			TenantInfo: tenantInfo,
		}).
		Return(nil, errortypes.NewNotFoundError("template version not found")).
		Once()
	repo.EXPECT().
		ListTemplateVersions(mock.Anything, repositories.ListEDITemplateVersionsRequest{
			TemplateID: templateID,
			TenantInfo: tenantInfo,
		}).
		Return([]*edi.EDITemplateVersion{{
			ID:                versionID,
			TemplateID:        templateID,
			Status:            edi.TemplateStatusDraft,
			FunctionalGroupID: "SM",
		}}, nil).
		Once()
	repo.EXPECT().
		GetTemplateByID(mock.Anything, repositories.GetEDITemplateByIDRequest{
			ID:         templateID,
			TenantInfo: tenantInfo,
		}).
		Return(&edi.EDITemplate{
			ID:             templateID,
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
			DocumentType: &edi.EDIDocumentType{
				ID:             documentTypeID,
				Standard:       edi.EDIStandardX12,
				TransactionSet: edi.TransactionSet204,
				Direction:      edi.DocumentDirectionOutbound,
			},
		}, nil).
		Once()
	repo.EXPECT().
		GetActivePartnerSettingSchema(mock.Anything, mock.Anything).
		Return(nil, sql.ErrNoRows).
		Once()
	repo.EXPECT().
		CreatePartnerDocumentProfile(
			mock.Anything,
			mock.MatchedBy(func(profile *edi.EDIPartnerDocumentProfile) bool {
				return profile.EDIPartnerID == partnerID &&
					profile.TemplateID == templateID &&
					profile.TemplateVersionID == versionID &&
					profile.Status == edi.DocumentStatusInactive
			}),
		).
		Return(&edi.EDIPartnerDocumentProfile{
			ID:                pulid.MustNew("edipdp_"),
			EDIPartnerID:      partnerID,
			TemplateID:        templateID,
			TemplateVersionID: versionID,
			Status:            edi.DocumentStatusInactive,
		}, nil).
		Once()

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo, validator: NewValidator()}
	profile, err := service.UpsertPartnerDocumentProfile(
		t.Context(),
		&UpsertEDIPartnerDocumentProfileRequest{
			TenantInfo:        tenantInfo,
			EDIPartnerID:      partnerID,
			TemplateID:        templateID,
			Status:            edi.DocumentStatusInactive,
			Name:              "New Profile",
			FunctionalGroupID: "SM",
			Envelope:          edi.DefaultX12EnvelopeSettings(),
			Acknowledgment:    edi.AcknowledgmentConfig{Type: edi.AcknowledgmentTypeNone},
			ValidationMode:    edi.ValidationModeStrict,
			PartnerSettings:   map[string]any{},
		},
		nil,
	)

	require.NoError(t, err)
	require.Equal(t, versionID, profile.TemplateVersionID)
	require.Equal(t, edi.DocumentStatusInactive, profile.Status)
}

func TestUpsertPartnerDocumentProfile_ActiveProfileRequiresActiveTemplateVersion(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	templateID := pulid.MustNew("editpl_")
	repo := mocks.NewMockEDIDocumentRepository(t)
	repo.EXPECT().
		GetActiveTemplateVersion(mock.Anything, repositories.GetActiveEDITemplateVersionRequest{
			TemplateID: templateID,
			TenantInfo: tenantInfo,
		}).
		Return(nil, errortypes.NewNotFoundError("template version not found")).
		Once()

	service := &Service{documentTypeRepo: repo, sourceContextRepo: repo, partnerSettingRepo: repo, templateRepo: repo, documentProfileRepo: repo, controlNumberRepo: repo, messageRepo: repo, testCaseRepo: repo, validator: NewValidator()}
	_, err := service.UpsertPartnerDocumentProfile(
		t.Context(),
		&UpsertEDIPartnerDocumentProfileRequest{
			TenantInfo:        tenantInfo,
			EDIPartnerID:      pulid.MustNew("edip_"),
			TemplateID:        templateID,
			Status:            edi.DocumentStatusActive,
			Name:              "New Profile",
			FunctionalGroupID: "SM",
			Envelope:          edi.DefaultX12EnvelopeSettings(),
			Acknowledgment:    edi.AcknowledgmentConfig{Type: edi.AcknowledgmentTypeNone},
			ValidationMode:    edi.ValidationModeStrict,
			PartnerSettings:   map[string]any{},
		},
		nil,
	)

	requireValidationError(t, err, "templateVersionId", errortypes.ErrInvalidOperation)
}
