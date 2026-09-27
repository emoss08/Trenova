package invoiceadjustmentservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/core/services/invoiceledger"
	"github.com/emoss08/trenova/internal/core/services/journalposting"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func (s *Service) createReplacementDraftInvoice(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	sourceInvoice *invoice.Invoice,
	lines []*invoice.InvoiceLine,
	preview *servicesports.InvoiceAdjustmentPreview,
) (*invoice.Invoice, error) {
	entity := &invoice.Invoice{
		OrganizationID:     sourceInvoice.OrganizationID,
		BusinessUnitID:     sourceInvoice.BusinessUnitID,
		BillingQueueItemID: item.ID,
		Scope:              invoice.ScopeAdjustment,
		ShipmentID:         sourceInvoice.ShipmentID,
		CustomerID:         sourceInvoice.CustomerID,
		Number:             item.Number,
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusDraft,
		PaymentTerm:        sourceInvoice.PaymentTerm,
		CurrencyCode:       sourceInvoice.CurrencyCode,
		InvoiceDate:        preview.AccountingDate,
		DueDate: invoice.DueDateFromPaymentTerm(
			preview.AccountingDate,
			sourceInvoice.PaymentTerm,
		),
		ShipmentProNumber:         sourceInvoice.ShipmentProNumber,
		ShipmentBOL:               sourceInvoice.ShipmentBOL,
		ServiceDate:               sourceInvoice.ServiceDate,
		BillToName:                sourceInvoice.BillToName,
		BillToCode:                sourceInvoice.BillToCode,
		BillToAddressLine1:        sourceInvoice.BillToAddressLine1,
		BillToAddressLine2:        sourceInvoice.BillToAddressLine2,
		BillToCity:                sourceInvoice.BillToCity,
		BillToState:               sourceInvoice.BillToState,
		BillToPostalCode:          sourceInvoice.BillToPostalCode,
		BillToCountry:             sourceInvoice.BillToCountry,
		ShipperCustomerID:         sourceInvoice.ShipperCustomerID,
		IsSplitBill:               sourceInvoice.IsSplitBill,
		SubtotalAmount:            sumInvoiceLines(lines, invoice.InvoiceLineTypeFreight),
		OtherAmount:               sumInvoiceLines(lines, invoice.InvoiceLineTypeAccessorial),
		TotalAmount:               preview.RebillTotalAmount,
		AppliedAmount:             decimal.Zero,
		SettlementStatus:          invoice.SettlementStatusUnpaid,
		DisputeStatus:             invoice.DisputeStatusNone,
		CorrectionGroupID:         adjustment.CorrectionGroupID,
		SupersedesInvoiceID:       sourceInvoice.ID,
		SourceInvoiceAdjustmentID: adjustment.ID,
		IsAdjustmentArtifact:      true,
		Lines:                     lines,
	}

	created, err := s.invoiceRepo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	sourceInvoice.SupersededByInvoiceID = created.ID
	sourceInvoice.CorrectionGroupID = adjustment.CorrectionGroupID
	if _, err = s.invoiceRepo.Update(ctx, sourceInvoice); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) createWriteOffJournalEntry(
	ctx context.Context,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	sourceInvoice *invoice.Invoice,
	preview *servicesports.InvoiceAdjustmentPreview,
	actor *servicesports.RequestActor,
) (pulid.ID, error) {
	accountingControl, err := s.accountingRepo.GetByOrgID(ctx, adjustment.OrganizationID)
	if err != nil {
		return pulid.ID(""), err
	}
	if accountingControl.DefaultWriteOffAccountID.IsNil() {
		return pulid.ID(
				"",
			), errortypes.NewValidationError(
				"defaultWriteOffAccountId",
				errortypes.ErrRequired,
				"Default write-off account is required for write-offs",
			)
	}
	if accountingControl.DefaultARAccountID.IsNil() {
		return pulid.ID(
				"",
			), errortypes.NewValidationError(
				"defaultArAccountId",
				errortypes.ErrRequired,
				"Default AR account is required for write-offs",
			)
	}
	if accountingControl.JournalPostingMode == tenant.JournalPostingModeManual &&
		accountingControl.ManualJournalEntryPolicy == tenant.ManualJournalEntryPolicyDisallow {
		return pulid.Nil, errortypes.NewValidationError(
			"manualJournalEntryPolicy",
			errortypes.ErrInvalidOperation,
			"Manual journal entries are disallowed by accounting policy",
		)
	}

	amount := money.MinorUnits(preview.CreditTotalAmount.Abs())
	description := fmt.Sprintf("Invoice write-off for %s", sourceInvoice.Number)
	result, err := journalposting.Writer{
		Periods:  s.fiscalPeriodRepo,
		Numbers:  s.sequenceGenerator,
		Journals: s.journalRepo,
	}.Write(ctx, &journalposting.WriteRequest{
		OrganizationID: adjustment.OrganizationID,
		BusinessUnitID: adjustment.BusinessUnitID,
		Control:        accountingControl,
		ActorID:        actor.UserID,
		AccountingDate: preview.AccountingDate,
		Now:            timeutils.NowUnix(),
		Subject:        "invoice write-off",
		Description:    description,
		EntryType:      "Adjusting",
		AutoGenerated:  true,
		Source: journalposting.Source{
			ObjectType:     "InvoiceAdjustment",
			ObjectID:       adjustment.ID,
			DocumentNumber: sourceInvoice.Number,
			Event:          tenant.JournalSourceEventType("InvoiceWriteOffCreated"),
			IdempotencyKey: "invoice-writeoff:" + adjustment.ID.String(),
		},
		Reference: journalposting.Reference{Type: "InvoiceAdjustmentWriteOff"},
		Lines: []journalposting.Line{
			{
				AccountID:  accountingControl.DefaultWriteOffAccountID,
				Debit:      amount,
				CustomerID: sourceInvoice.CustomerID,
			},
			{
				AccountID:  accountingControl.DefaultARAccountID,
				Credit:     amount,
				CustomerID: sourceInvoice.CustomerID,
			},
		},
	})
	if err != nil {
		return pulid.Nil, fmt.Errorf("create write-off journal posting: %w", err)
	}

	return result.EntryID, nil
}

func replacementQueueStatus(reviewRequired bool) billingqueue.Status {
	if reviewRequired {
		return billingqueue.StatusReadyForReview
	}
	return billingqueue.StatusApproved
}

func (s *Service) creditMemoLedger() *invoiceledger.Poster {
	policy := s.accountingPolicy
	if policy == nil {
		policy = accountingcontrolpolicyservice.New(
			accountingcontrolpolicyservice.Params{Logger: zap.NewNop()},
		)
	}
	poster := &invoiceledger.Poster{
		Controls: s.accountingRepo,
		Policy:   policy,
		Writer: journalposting.Writer{
			Periods:  s.fiscalPeriodRepo,
			Numbers:  s.sequenceGenerator,
			Journals: s.journalRepo,
		},
	}
	if s.customerLedgerRepo != nil {
		poster.Ledger = s.customerLedgerRepo
	}
	return poster
}

func (s *Service) postCreditMemoLedger(
	ctx context.Context,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	creditMemo *invoice.Invoice,
	sourceInvoice *invoice.Invoice,
	actor *servicesports.RequestActor,
) error {
	result, err := s.creditMemoLedger().Post(
		ctx,
		invoiceledger.CreditMemoRequest(adjustment.Kind, creditMemo, sourceInvoice.ID, actor.UserID),
	)
	if errors.Is(err, invoiceledger.ErrNoLedgerEntry) {
		return nil
	}
	if err != nil {
		return err
	}
	if result.Journal == nil {
		return nil
	}
	if adjustment.Metadata == nil {
		adjustment.Metadata = make(map[string]any, 1)
	}
	adjustment.Metadata["creditMemoJournalEntryId"] = result.Journal.EntryID.String()
	return nil
}
