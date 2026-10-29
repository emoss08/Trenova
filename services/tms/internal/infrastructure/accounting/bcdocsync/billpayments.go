package bcdocsync

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcrefsync"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcsession"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/businesscentralpay"
	"github.com/emoss08/trenova/shared/businesscentralref"
)

func (c *Documents) CreateBillPayment(
	ctx context.Context,
	doc *accountingconnectorports.AccountingBillPaymentDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if !doc.Kind.IsBillPayment() {
		return nil, bcsession.ErrDocumentKind
	}
	if !doc.Amount.IsPositive() {
		return nil, businesscentralerrors.ErrAmountInvalid
	}
	if strings.TrimSpace(doc.BankAccountExternalID) == "" {
		return nil, unmappedAccount("The bank account")
	}
	client, err := c.Client(doc.Auth)
	if err != nil {
		return nil, err
	}
	if _, err = c.PostingFor(ctx, client, doc.TxnDate); err != nil {
		return nil, err
	}
	result := newResult(nil)
	number := RequestReference(doc.RequestID)
	if err = c.postPayment(ctx, &journalWrite{
		client:         client,
		party:          businesscentralref.PartyVendor,
		accountID:      doc.BankAccountExternalID,
		documentNumber: number,
		lines: []businesscentralpay.PaymentInput{{
			PartyID:        doc.VendorExternalID,
			PostingDate:    doc.TxnDate,
			DocumentNumber: number,
			ExternalDocumentNumber: bcrefsync.CleanText(
				doc.DocNumber,
				businesscentral.MaxExternalDocumentNumberLength,
			),
			Amount:             doc.Amount,
			AppliesToInvoiceID: strings.TrimSpace(doc.BillExternalID),
			Description:        bcrefsync.CleanText(doc.PrivateNote, businesscentral.MaxDescriptionLength),
		}},
		refs: result.Refs,
	}); err != nil {
		return result, err
	}
	finishPayment(result, number, DocTypeVendorPayment)
	return result, nil
}

func (c *Documents) UpdateBillPayment(
	_ context.Context,
	_ *accountingconnectorports.AccountingBillPaymentDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return nil, reversalRefused()
}
