package factoring

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const MaxReserveMemoLength = 500

// Each funding, reserve release, chargeback and reversal booked with a factor, with the cash, fee, reserve and receivables it moved
type ReserveEntry struct {
	keel.Table `                                                            keel:"factoring_reserve_entries,report=Billing,label=Factoring Reserve Entry"`

	ID                 pulid.ID            `json:"id" keel:"id,pk,autoid:facres_"`
	OrganizationID     pulid.ID            `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID     pulid.ID            `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	FactoringCompanyID pulid.ID            `json:"factoringCompanyId" keel:"factoring_company_id"`
	AgreementID        pulid.ID            `json:"agreementId" keel:"agreement_id"`
	SubmissionID       pulid.ID            `json:"submissionId" keel:"submission_id"`
	SubmissionItemID   pulid.ID            `json:"submissionItemId" keel:"submission_item_id,label=Schedule Invoice ID"`
	InvoiceID          pulid.ID            `json:"invoiceId" keel:"invoice_id"`
	CustomerID         pulid.ID            `json:"customerId" keel:"customer_id"`
	Kind               ReserveEntryKind    `json:"kind" keel:"kind"`
	Treatment          AccountingTreatment `json:"treatment" keel:"treatment,label=Accounting Treatment"`
	CurrencyCode       string              `json:"currencyCode" keel:"currency_code,label=Currency"`
	CashMinor          int64               `json:"cashMinor" keel:"cash_minor,label=Cash (Minor)"`
	FeeMinor           int64               `json:"feeMinor" keel:"fee_minor,label=Fee (Minor)"`
	ReserveMinor       int64               `json:"reserveMinor" keel:"reserve_minor,label=Reserve (Minor)"`
	LiabilityMinor     int64               `json:"liabilityMinor" keel:"liability_minor,label=Owed to Factor (Minor)"`
	ARMinor            int64               `json:"arMinor" keel:"ar_minor,label=Receivables (Minor)"`
	JournalEntryID     pulid.ID            `json:"journalEntryId" keel:"journal_entry_id"`
	JournalBatchID     pulid.ID            `json:"journalBatchId" keel:"journal_batch_id"`
	AccountingDate     int64               `json:"accountingDate" keel:"accounting_date"`
	OccurredOn         int64               `json:"occurredOn" keel:"occurred_on,type=epoch"`
	ReversesEntryID    pulid.ID            `json:"reversesEntryId" keel:"reverses_entry_id,nullzero"`
	RemittanceLineID   pulid.ID            `json:"remittanceLineId" keel:"remittance_line_id,nullzero"`
	Memo               string              `json:"memo" keel:"memo,nullzero"`
	CreatedByID        pulid.ID            `json:"createdById" keel:"created_by_id,nullzero,label=Booked By ID"`
	CreatedAt          int64               `json:"createdAt" keel:"created_at,autocreate"`

	FactoringCompany *factoringcompany.FactoringCompany `json:"factoringCompany,omitempty" keel:"belongs:factoring_company_id"`
	Invoice          *invoice.Invoice                   `json:"invoice,omitempty" keel:"belongs:invoice_id"`
	Customer         *customer.Customer                 `json:"customer,omitempty" keel:"belongs:customer_id"`
	Submission       *Submission                        `json:"submission,omitempty" keel:"belongs:submission_id,label=Schedule"`
}

func NewReserveEntry(
	item *SubmissionItem,
	submission *Submission,
	movement *Movement,
	occurredOn int64,
) *ReserveEntry {
	return &ReserveEntry{
		OrganizationID:     item.OrganizationID,
		BusinessUnitID:     item.BusinessUnitID,
		FactoringCompanyID: submission.FactoringCompanyID,
		AgreementID:        submission.AgreementID,
		SubmissionID:       submission.ID,
		SubmissionItemID:   item.ID,
		InvoiceID:          item.InvoiceID,
		CustomerID:         item.CustomerID,
		Kind:               movement.Kind,
		Treatment:          movement.Treatment,
		CurrencyCode:       submission.CurrencyCode,
		CashMinor:          movement.CashMinor,
		FeeMinor:           movement.FeeMinor,
		ReserveMinor:       movement.ReserveMinor,
		LiabilityMinor:     movement.LiabilityMinor,
		ARMinor:            movement.ARMinor,
		OccurredOn:         occurredOn,
	}
}

func (e *ReserveEntry) Movement() Movement {
	movement := Movement{
		Kind:           e.Kind,
		Treatment:      e.Treatment,
		CashMinor:      e.CashMinor,
		FeeMinor:       e.FeeMinor,
		ReserveMinor:   e.ReserveMinor,
		LiabilityMinor: e.LiabilityMinor,
		ARMinor:        e.ARMinor,
	}
	movement.Lines = make([]MovementLine, 0, 5)
	movement.Lines = appendSigned(movement.Lines, AccountRoleCash, e.CashMinor)
	movement.Lines = appendSigned(movement.Lines, AccountRoleFeeExpense, e.FeeMinor)
	movement.Lines = appendSigned(movement.Lines, AccountRoleDueFromFactor, e.ReserveMinor)
	movement.Lines = appendSigned(movement.Lines, AccountRoleLiability, -e.LiabilityMinor)
	movement.Lines = appendSigned(movement.Lines, AccountRoleAR, e.ARMinor)
	switch {
	case e.ARMinor < 0:
		movement.InvoiceEffect = InvoiceEffectSettle
		movement.InvoiceMinor = -e.ARMinor
	case e.ARMinor > 0:
		movement.InvoiceEffect = InvoiceEffectReopen
		movement.InvoiceMinor = e.ARMinor
	}
	return movement
}

func (e *ReserveEntry) BeforeCreate(ctx context.Context) error {
	if e.CurrencyCode == "" {
		e.CurrencyCode = "USD"
	}

	return nil
}

func (e *ReserveEntry) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "facres",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "kind", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightA},
			{Name: "memo", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
		},
	}
}

func (e *ReserveEntry) GetID() pulid.ID             { return e.ID }
func (e *ReserveEntry) GetCreatedAt() int64         { return e.CreatedAt }
func (e *ReserveEntry) GetOrganizationID() pulid.ID { return e.OrganizationID }
func (e *ReserveEntry) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }
func (e *ReserveEntry) GetTableName() string        { return "factoring_reserve_entries" }
