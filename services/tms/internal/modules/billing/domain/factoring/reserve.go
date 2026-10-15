package factoring

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const MaxReserveMemoLength = 500

var _ bun.BeforeAppendModelHook = (*ReserveEntry)(nil)

type ReserveEntry struct {
	bun.BaseModel             `bun:"table:factoring_reserve_entries,alias:facres" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                       json:"-"`

	ID                 pulid.ID            `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID            `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID     pulid.ID            `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	FactoringCompanyID pulid.ID            `json:"factoringCompanyId" bun:"factoring_company_id,type:VARCHAR(100),notnull"`
	AgreementID        pulid.ID            `json:"agreementId"        bun:"agreement_id,type:VARCHAR(100),notnull"`
	SubmissionID       pulid.ID            `json:"submissionId"       bun:"submission_id,type:VARCHAR(100),notnull"`
	SubmissionItemID   pulid.ID            `json:"submissionItemId"   bun:"submission_item_id,type:VARCHAR(100),notnull"`
	InvoiceID          pulid.ID            `json:"invoiceId"          bun:"invoice_id,type:VARCHAR(100),notnull"`
	CustomerID         pulid.ID            `json:"customerId"         bun:"customer_id,type:VARCHAR(100),notnull"`
	Kind               ReserveEntryKind    `json:"kind"               bun:"kind,type:VARCHAR(30),notnull"`
	Treatment          AccountingTreatment `json:"treatment"          bun:"treatment,type:VARCHAR(30),notnull"`
	CurrencyCode       string              `json:"currencyCode"       bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	CashMinor          int64               `json:"cashMinor"          bun:"cash_minor,type:BIGINT,notnull"`
	FeeMinor           int64               `json:"feeMinor"           bun:"fee_minor,type:BIGINT,notnull"`
	ReserveMinor       int64               `json:"reserveMinor"       bun:"reserve_minor,type:BIGINT,notnull"`
	LiabilityMinor     int64               `json:"liabilityMinor"     bun:"liability_minor,type:BIGINT,notnull"`
	ARMinor            int64               `json:"arMinor"            bun:"ar_minor,type:BIGINT,notnull"`
	JournalEntryID     pulid.ID            `json:"journalEntryId"     bun:"journal_entry_id,type:VARCHAR(100),notnull"`
	JournalBatchID     pulid.ID            `json:"journalBatchId"     bun:"journal_batch_id,type:VARCHAR(100),notnull"`
	AccountingDate     int64               `json:"accountingDate"     bun:"accounting_date,type:BIGINT,notnull"`
	OccurredOn         int64               `json:"occurredOn"         bun:"occurred_on,type:BIGINT,notnull"`
	ReversesEntryID    pulid.ID            `json:"reversesEntryId"    bun:"reverses_entry_id,type:VARCHAR(100),nullzero"`
	Memo               string              `json:"memo"               bun:"memo,type:VARCHAR(500),nullzero"`
	CreatedByID        pulid.ID            `json:"createdById"        bun:"created_by_id,type:VARCHAR(100),nullzero"`
	CreatedAt          int64               `json:"createdAt"          bun:"created_at,type:BIGINT,notnull"`

	FactoringCompany *factoringcompany.FactoringCompany `json:"factoringCompany,omitempty" bun:"rel:belongs-to,join:factoring_company_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Invoice          *invoice.Invoice                   `json:"invoice,omitempty"          bun:"rel:belongs-to,join:invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Customer         *customer.Customer                 `json:"customer,omitempty"         bun:"rel:belongs-to,join:customer_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Submission       *Submission                        `json:"submission,omitempty"       bun:"rel:belongs-to,join:submission_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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

func (e *ReserveEntry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("facres_")
		}
		e.CreatedAt = timeutils.NowUnix()
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
