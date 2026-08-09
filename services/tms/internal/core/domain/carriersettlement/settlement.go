package carriersettlement

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*CarrierSettlement)(nil)
	_ domaintypes.PostgresSearchable     = (*CarrierSettlement)(nil)
	_ pagination.CursorEntity            = (*CarrierSettlement)(nil)
	_ validationframework.TenantedEntity = (*CarrierSettlement)(nil)
	_ bun.BeforeAppendModelHook          = (*CarrierSettlementLine)(nil)
)

type CarrierSettlement struct {
	bun.BaseModel             `bun:"table:carrier_settlements,alias:carstl" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                 json:"-"`

	ID                   pulid.ID  `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID  `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID  `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	CarrierID            pulid.ID  `json:"carrierId"            bun:"carrier_id,type:VARCHAR(100),notnull"`
	BatchID              *pulid.ID `json:"batchId"              bun:"batch_id,type:VARCHAR(100),nullzero"`
	SettlementNumber     string    `json:"settlementNumber"     bun:"settlement_number,type:VARCHAR(100),notnull"`
	Status               Status    `json:"status"               bun:"status,type:VARCHAR(50),notnull,default:'Draft'"`
	PeriodStart          int64     `json:"periodStart"          bun:"period_start,type:BIGINT,notnull"`
	PeriodEnd            int64     `json:"periodEnd"            bun:"period_end,type:BIGINT,notnull"`
	PayDate              int64     `json:"payDate"              bun:"pay_date,type:BIGINT,notnull"`
	GrossCostMinor       int64     `json:"grossCostMinor"       bun:"gross_cost_minor,type:BIGINT,notnull,default:0"`
	AdjustmentsMinor     int64     `json:"adjustmentsMinor"     bun:"adjustments_minor,type:BIGINT,notnull,default:0"`
	NetPayableMinor      int64     `json:"netPayableMinor"      bun:"net_payable_minor,type:BIGINT,notnull,default:0"`
	ShipmentCount        int       `json:"shipmentCount"        bun:"shipment_count,type:INTEGER,notnull,default:0"`
	CurrencyCode         string    `json:"currencyCode"         bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	Notes                string    `json:"notes"                bun:"notes,type:TEXT,nullzero"`
	SubmittedByID        pulid.ID  `json:"submittedById"        bun:"submitted_by_id,type:VARCHAR(100),nullzero"`
	SubmittedAt          *int64    `json:"submittedAt"          bun:"submitted_at,type:BIGINT,nullzero"`
	ApprovedByID         pulid.ID  `json:"approvedById"         bun:"approved_by_id,type:VARCHAR(100),nullzero"`
	ApprovedAt           *int64    `json:"approvedAt"           bun:"approved_at,type:BIGINT,nullzero"`
	PostedByID           pulid.ID  `json:"postedById"           bun:"posted_by_id,type:VARCHAR(100),nullzero"`
	PostedAt             *int64    `json:"postedAt"             bun:"posted_at,type:BIGINT,nullzero"`
	PostedJournalBatchID *pulid.ID `json:"postedJournalBatchId" bun:"posted_journal_batch_id,type:VARCHAR(100),nullzero"`
	PaidAt               *int64    `json:"paidAt"               bun:"paid_at,type:BIGINT,nullzero"`
	PaidByID             pulid.ID  `json:"paidById"             bun:"paid_by_id,type:VARCHAR(100),nullzero"`
	PaymentMethod        string    `json:"paymentMethod"        bun:"payment_method,type:VARCHAR(50),nullzero"`
	PaymentReference     string    `json:"paymentReference"     bun:"payment_reference,type:VARCHAR(100),nullzero"`
	PaidJournalBatchID   *pulid.ID `json:"paidJournalBatchId"   bun:"paid_journal_batch_id,type:VARCHAR(100),nullzero"`
	VoidedByID           pulid.ID  `json:"voidedById"           bun:"voided_by_id,type:VARCHAR(100),nullzero"`
	VoidedAt             *int64    `json:"voidedAt"             bun:"voided_at,type:BIGINT,nullzero"`
	VoidReason           string    `json:"voidReason"           bun:"void_reason,type:TEXT,nullzero"`
	VoidJournalBatchID   *pulid.ID `json:"voidJournalBatchId"   bun:"void_journal_batch_id,type:VARCHAR(100),nullzero"`
	Version              int64     `json:"version"              bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt            int64     `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64     `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit     `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization     `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Carrier      *carrier.Carrier         `json:"carrier,omitempty"      bun:"rel:belongs-to,join:carrier_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Batch        *CarrierSettlementBatch  `json:"batch,omitempty"        bun:"rel:belongs-to,join:batch_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Lines        []*CarrierSettlementLine `json:"lines,omitempty"        bun:"rel:has-many,join:id=settlement_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

type CarrierSettlementLine struct {
	bun.BaseModel `bun:"table:carrier_settlement_lines,alias:carstll" json:"-"`

	ID             pulid.ID      `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID      `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID      `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	SettlementID   pulid.ID      `json:"settlementId"   bun:"settlement_id,type:VARCHAR(100),notnull"`
	LineNumber     int           `json:"lineNumber"     bun:"line_number,type:INTEGER,notnull"`
	EventType      CostEventType `json:"eventType"      bun:"event_type,type:VARCHAR(50),notnull"`
	Description    string        `json:"description"    bun:"description,type:VARCHAR(255),notnull"`
	AmountMinor    int64         `json:"amountMinor"    bun:"amount_minor,type:BIGINT,notnull"`
	CostEventID    *pulid.ID     `json:"costEventId"    bun:"cost_event_id,type:VARCHAR(100),nullzero"`
	GLAccountID    *pulid.ID     `json:"glAccountId"    bun:"gl_account_id,type:VARCHAR(100),nullzero"`
	ShipmentID     *pulid.ID     `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),nullzero"`
	MoveID         *pulid.ID     `json:"moveId"         bun:"move_id,type:VARCHAR(100),nullzero"`
	ProNumber      string        `json:"proNumber"      bun:"pro_number,type:VARCHAR(100),nullzero"`
	CreatedAt      int64         `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64         `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (s *CarrierSettlement) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.CarrierID, validation.Required.Error("Carrier is required")),
		validation.Field(&s.PeriodStart, validation.Required.Error("Period start is required")),
		validation.Field(&s.PeriodEnd, validation.Required.Error("Period end is required")),
		validation.Field(&s.PayDate, validation.Required.Error("Pay date is required")),
		validation.Field(&s.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Length(3, 3).Error("Currency code must be 3 characters"),
		),
	))

	if !s.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Settlement status is invalid")
	}
	if s.PeriodEnd <= s.PeriodStart {
		multiErr.Add(
			"periodEnd",
			errortypes.ErrInvalid,
			"Period end must be after the period start",
		)
	}
	for idx, line := range s.Lines {
		if line == nil {
			multiErr.Add(
				"lines",
				errortypes.ErrInvalid,
				"Settlement lines must not contain null values",
			)
			continue
		}
		line.Validate(multiErr.WithIndex("lines", idx))
	}
}

func (l *CarrierSettlementLine) Validate(multiErr *errortypes.MultiError) {
	if !l.EventType.IsValid() {
		multiErr.Add("eventType", errortypes.ErrInvalid, "Line event type is invalid")
	}
	if l.Description == "" {
		multiErr.Add("description", errortypes.ErrRequired, "Description is required")
	}
	if l.EventType != CostEventTypeAdjustment && l.AmountMinor < 0 {
		multiErr.Add(
			"amountMinor",
			errortypes.ErrInvalid,
			"Only adjustment lines may carry a negative amount",
		)
	}
}

func (l *CarrierSettlementLine) IsManualAdjustment() bool {
	return l.EventType == CostEventTypeAdjustment &&
		(l.CostEventID == nil || l.CostEventID.IsNil())
}

func (s *CarrierSettlement) SyncTotals() {
	var gross, adjustments int64
	for idx, line := range s.Lines {
		if line == nil {
			continue
		}
		line.LineNumber = idx + 1
		switch line.EventType {
		case CostEventTypeLinehaulCost, CostEventTypeFuelSurcharge,
			CostEventTypeAccessorial:
			gross += line.AmountMinor
		case CostEventTypeAdjustment:
			adjustments += line.AmountMinor
		}
	}
	s.GrossCostMinor = gross
	s.AdjustmentsMinor = adjustments
	s.NetPayableMinor = gross + adjustments
}

func (s *CarrierSettlement) IsEditable() bool {
	return s.Status == StatusDraft || s.Status == StatusPendingApproval
}

func (s *CarrierSettlement) GetID() pulid.ID { return s.ID }

func (s *CarrierSettlement) GetCreatedAt() int64 { return s.CreatedAt }

func (s *CarrierSettlement) GetOrganizationID() pulid.ID { return s.OrganizationID }

func (s *CarrierSettlement) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }

func (s *CarrierSettlement) GetTableName() string { return "carrier_settlements" }

func (s *CarrierSettlement) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "carstl",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "settlement_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
		},
	}
}

func (s *CarrierSettlement) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("carstl_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}
	return nil
}

func (l *CarrierSettlementLine) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("carstll_")
		}
		l.CreatedAt = now
	case *bun.UpdateQuery:
		l.UpdatedAt = now
	}
	return nil
}
