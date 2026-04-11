package invoiceadjustmentservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func (s *Service) createReplacementDraftInvoice(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	adjustment *invoiceadjustment.Adjustment,
	sourceInvoice *invoice.Invoice,
	lines []*invoice.Line,
	preview *servicesports.InvoiceAdjustmentPreview,
) (*invoice.Invoice, error) {
	entity := &invoice.Invoice{
		OrganizationID:            sourceInvoice.OrganizationID,
		BusinessUnitID:            sourceInvoice.BusinessUnitID,
		BillingQueueItemID:        item.ID,
		ShipmentID:                sourceInvoice.ShipmentID,
		CustomerID:                sourceInvoice.CustomerID,
		Number:                    item.Number,
		BillType:                  billingqueue.BillTypeInvoice,
		Status:                    invoice.StatusDraft,
		PaymentTerm:               sourceInvoice.PaymentTerm,
		CurrencyCode:              sourceInvoice.CurrencyCode,
		InvoiceDate:               preview.AccountingDate,
		DueDate:                   invoice.DueDateFromPaymentTerm(preview.AccountingDate, sourceInvoice.PaymentTerm),
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
		SubtotalAmount:            sumInvoiceLines(lines, invoice.LineTypeFreight),
		OtherAmount:               sumInvoiceLines(lines, invoice.LineTypeAccessorial),
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
	adjustment *invoiceadjustment.Adjustment,
	sourceInvoice *invoice.Invoice,
	preview *servicesports.InvoiceAdjustmentPreview,
	actor *servicesports.RequestActor,
) (pulid.ID, error) {
	accountingControl, err := s.accountingRepo.GetByOrgID(ctx, adjustment.OrganizationID)
	if err != nil {
		return pulid.ID(""), err
	}
	if accountingControl.DefaultWriteOffAccountID.IsNil() {
		return pulid.ID(""), errortypes.NewValidationError("defaultWriteOffAccountId", errortypes.ErrRequired, "Default write-off account is required for write-offs")
	}
	if accountingControl.DefaultARAccountID.IsNil() {
		return pulid.ID(""), errortypes.NewValidationError("defaultArAccountId", errortypes.ErrRequired, "Default AR account is required for write-offs")
	}
	period, err := s.fiscalPeriodRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{
		OrgID: adjustment.OrganizationID,
		BuID:  adjustment.BusinessUnitID,
		Date:  preview.AccountingDate,
	})
	if err != nil {
		return pulid.ID(""), err
	}

	now := adjustment.AccountingDate
	amount := money.MinorUnits(preview.CreditTotalAmount.Abs())
	entryID := pulid.MustNew("je_")
	batchID := pulid.MustNew("jb_")
	batchNumber, err := s.sequenceGenerator.GenerateJournalBatchNumber(ctx, adjustment.OrganizationID, adjustment.BusinessUnitID, "", "")
	if err != nil {
		return pulid.ID(""), err
	}
	entryNumber, err := s.sequenceGenerator.GenerateJournalEntryNumber(ctx, adjustment.OrganizationID, adjustment.BusinessUnitID, "", "")
	if err != nil {
		return pulid.ID(""), err
	}

	entryStatus := "Posted"
	batchStatus := "Posted"
	postedAt := &now
	postedByID := actor.UserID
	requiresApproval := false
	isApproved := true
	approvedAt := &now
	approvedByID := actor.UserID

	switch accountingControl.JournalPostingMode {
	case tenant.JournalPostingModeManual:
		if accountingControl.ManualJournalEntryPolicy == tenant.ManualJournalEntryPolicyDisallow {
			return pulid.ID(""), errortypes.NewValidationError("manualJournalEntryPolicy", errortypes.ErrInvalidOperation, "Manual journal entries are disallowed by accounting policy")
		}
		entryStatus = "Pending"
		batchStatus = "Pending"
		postedAt = nil
		postedByID = pulid.Nil
		requiresApproval = accountingControl.RequireManualJEApproval
		isApproved = !accountingControl.RequireManualJEApproval
		if !accountingControl.RequireManualJEApproval {
			entryStatus = "Approved"
			batchStatus = "Approved"
		} else {
			approvedAt = nil
			approvedByID = pulid.Nil
		}
	}

	lines := []repositories.JournalPostingLine{
		{
			ID:           pulid.MustNew("jel_"),
			GLAccountID:  accountingControl.DefaultWriteOffAccountID,
			LineNumber:   1,
			Description:  fmt.Sprintf("Invoice write-off for %s", sourceInvoice.Number),
			DebitAmount:  amount,
			CreditAmount: 0,
			NetAmount:    amount,
			CustomerID:   sourceInvoice.CustomerID,
		},
		{
			ID:           pulid.MustNew("jel_"),
			GLAccountID:  accountingControl.DefaultARAccountID,
			LineNumber:   2,
			Description:  fmt.Sprintf("Invoice write-off for %s", sourceInvoice.Number),
			DebitAmount:  0,
			CreditAmount: amount,
			NetAmount:    -amount,
			CustomerID:   sourceInvoice.CustomerID,
		},
	}

	err = s.journalRepo.CreatePosting(ctx, repositories.CreateJournalPostingParams{
		BatchID:              batchID,
		OrganizationID:       adjustment.OrganizationID,
		BusinessUnitID:       adjustment.BusinessUnitID,
		BatchNumber:          batchNumber,
		BatchType:            "System",
		BatchStatus:          batchStatus,
		BatchDescription:     fmt.Sprintf("Invoice write-off for %s", sourceInvoice.Number),
		FiscalYearID:         period.FiscalYearID,
		FiscalPeriodID:       period.ID,
		AccountingDate:       preview.AccountingDate,
		PostedAt:             postedAt,
		PostedByID:           postedByID,
		CreatedByID:          actor.UserID,
		UpdatedByID:          actor.UserID,
		EntryID:              entryID,
		EntryNumber:          entryNumber,
		EntryType:            "Adjusting",
		EntryStatus:          entryStatus,
		ReferenceNumber:      sourceInvoice.Number,
		ReferenceType:        "InvoiceAdjustmentWriteOff",
		ReferenceID:          adjustment.ID.String(),
		EntryDescription:     fmt.Sprintf("Invoice write-off for %s", sourceInvoice.Number),
		TotalDebit:           amount,
		TotalCredit:          amount,
		IsPosted:             postedAt != nil,
		IsAutoGenerated:      true,
		RequiresApproval:     requiresApproval,
		IsApproved:           isApproved,
		ApprovedByID:         approvedByID,
		ApprovedAt:           approvedAt,
		SourceID:             pulid.MustNew("jsrc_"),
		SourceObjectType:     "InvoiceAdjustment",
		SourceObjectID:       adjustment.ID.String(),
		SourceEventType:      "InvoiceWriteOffCreated",
		SourceStatus:         entryStatus,
		SourceDocumentNumber: sourceInvoice.Number,
		SourceIdempotencyKey: "invoice-writeoff:" + adjustment.ID.String(),
		Lines:                lines,
	})
	if err != nil {
		return pulid.ID(""), fmt.Errorf("create write-off journal posting: %w", err)
	}

	return entryID, nil
}

func replacementQueueStatus(reviewRequired bool) billingqueue.Status {
	if reviewRequired {
		return billingqueue.StatusReadyForReview
	}
	return billingqueue.StatusApproved
}
