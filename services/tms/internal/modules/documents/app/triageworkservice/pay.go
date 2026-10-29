package triageworkservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageworkports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var (
	invoiceNumberKeys = []string{"invoice_number", "invoicenumber", "invoice", "invoice_no"}
	invoiceTotalKeys  = []string{"total", "total_amount", "totalamount", "amount_due", "amount"}
)

func (s *Service) queueForPay(
	ctx context.Context,
	f *matchedFiling,
	attachment *inboundmessage.InboundAttachment,
) {
	if s.carrierPay == nil || f.message.MatchedCarrierID.IsNil() {
		s.l.Info("filed a carrier invoice without queueing it for pay",
			zap.String("messageId", f.message.ID.String()),
			zap.Bool("carrierMatched", f.message.MatchedCarrierID.IsNotNil()))

		return
	}

	extractionID, err := s.work.LatestExtractionID(ctx, f.tenantInfo, attachment.DocumentID)
	if err != nil {
		s.l.Warn("could not read the invoice extraction to queue for pay",
			zap.String("attachmentId", attachment.ID.String()), zap.Error(err))

		return
	}

	number, totalMinor := s.invoiceFields(ctx, f, attachment.DocumentID)
	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeSystem,
		OrganizationID: f.tenantInfo.OrgID,
		BusinessUnitID: f.tenantInfo.BuID,
	}
	if f.userID.IsNotNil() {
		tenantInfo := f.tenantInfo
		tenantInfo.UserID = f.userID
		actor = permissionports.UserActor(tenantInfo)
	}

	if err = s.carrierPay.QueueCarrierInvoice(ctx, &triageworkports.QueueCarrierInvoiceRequest{
		TenantInfo:             f.tenantInfo,
		Actor:                  actor,
		ShipmentID:             f.message.MatchedShipmentID,
		CarrierID:              f.message.MatchedCarrierID,
		DocumentAIExtractionID: extractionID,
		InvoiceNumber:          number,
		TotalMinor:             totalMinor,
	}); err != nil {
		s.l.Warn("filed a carrier invoice but could not queue it for pay",
			zap.String("messageId", f.message.ID.String()), zap.Error(err))
	}
}

func (s *Service) invoiceFields(
	ctx context.Context,
	f *matchedFiling,
	documentID pulid.ID,
) (number string, totalMinor int64) {
	contents, err := s.work.ContentsByDocumentIDs(ctx, f.tenantInfo, []pulid.ID{documentID})
	if err != nil {
		s.l.Warn("could not read what the invoice says",
			zap.String("documentId", documentID.String()), zap.Error(err))

		return "", 0
	}

	content := contents[documentID]
	if content == nil {
		return "", 0
	}
	findings := content.Findings()
	if findings == nil {
		return "", 0
	}

	number = fieldValue(findings, invoiceNumberKeys)
	total, parseErr := decimalutils.ParseMoneyText(fieldValue(findings, invoiceTotalKeys))
	if parseErr != nil || !total.Valid {
		return number, 0
	}

	return number, money.MinorUnits(total.Decimal)
}

func fieldValue(findings *documentcontent.Findings, keys []string) string {
	for _, key := range keys {
		for _, field := range findings.Fields {
			if strings.EqualFold(normalizedKey(field.Key), key) {
				return strings.TrimSpace(field.Value)
			}
		}
	}

	return ""
}

func normalizedKey(key string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(key)), " ", "_")
}
