package factoring

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	SubmissionDocumentResourceType = "factoring_submission"
	MaxSubmissionInvoices          = 200
	MaxSubmissionErrorLength       = 2000
	MaxSubmissionCancelReason      = 500
	MaxTransportTargetLength       = 1000
	MaxFundingReasonLength         = 500
)

var (
	_ bun.BeforeAppendModelHook = (*Submission)(nil)
	_ bun.BeforeAppendModelHook = (*SubmissionItem)(nil)
)

// Schedules of accounts sent to each factor, with their invoices' total, expected advance, fee and reserve, and how delivery went
type Submission struct {
	bun.BaseModel             `bun:"table:factoring_submissions,alias:facsub" json:"-"`
	keel.Table                `                                                        keel:"factoring_submissions,report=Billing,label=Factoring Schedule"`
	pagination.CursorValueSet `bun:",embed"                                   json:"-"`

	ID                    pulid.ID         `json:"id"                    bun:"id,pk,type:VARCHAR(100),notnull"                     keel:"id,pk,autoid:facsub_"`
	OrganizationID        pulid.ID         `json:"organizationId"        bun:"organization_id,pk,type:VARCHAR(100),notnull"        keel:"organization_id,pk,tenant"`
	BusinessUnitID        pulid.ID         `json:"businessUnitId"        bun:"business_unit_id,pk,type:VARCHAR(100),notnull"       keel:"business_unit_id,pk,tenant"`
	Number                string           `json:"number"                bun:"number,type:VARCHAR(100),notnull"                    keel:"number,label=Schedule Number"`
	FactoringCompanyID    pulid.ID         `json:"factoringCompanyId"    bun:"factoring_company_id,type:VARCHAR(100),notnull"      keel:"factoring_company_id"`
	AgreementID           pulid.ID         `json:"agreementId"           bun:"agreement_id,type:VARCHAR(100),notnull"              keel:"agreement_id"`
	Status                SubmissionStatus `json:"status"                bun:"status,type:VARCHAR(20),notnull,default:'Sending'"   keel:"status"`
	CurrencyCode          string           `json:"currencyCode"          bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'" keel:"currency_code,label=Currency"`
	InvoiceCount          int              `json:"invoiceCount"          bun:"invoice_count,type:INTEGER,notnull"                  keel:"invoice_count,label=Invoices,format=count"`
	FaceMinor             int64            `json:"faceMinor"             bun:"face_minor,type:BIGINT,notnull"                      keel:"face_minor,label=Face Amount (Minor)"`
	ExpectedAdvanceMinor  int64            `json:"expectedAdvanceMinor"  bun:"expected_advance_minor,type:BIGINT,notnull"          keel:"expected_advance_minor,label=Expected Advance (Minor)"`
	ExpectedFeeMinor      int64            `json:"expectedFeeMinor"      bun:"expected_fee_minor,type:BIGINT,notnull"              keel:"expected_fee_minor,label=Expected Fee (Minor)"`
	ExpectedReserveMinor  int64            `json:"expectedReserveMinor"  bun:"expected_reserve_minor,type:BIGINT,notnull"          keel:"expected_reserve_minor,label=Expected Reserve (Minor)"`
	SchedulePDFDocumentID pulid.ID         `json:"schedulePdfDocumentId" bun:"schedule_pdf_document_id,type:VARCHAR(100),nullzero" keel:"schedule_pdf_document_id,nullzero"`
	ScheduleCSVDocumentID pulid.ID         `json:"scheduleCsvDocumentId" bun:"schedule_csv_document_id,type:VARCHAR(100),nullzero" keel:"schedule_csv_document_id,nullzero"`
	TransportMethod       TransportMethod  `json:"transportMethod"       bun:"transport_method,type:VARCHAR(20),notnull"           keel:"transport_method,label=Sent By"`
	TransportTarget       string           `json:"transportTarget"       bun:"transport_target,type:VARCHAR(1000),notnull"         keel:"transport_target,label=Sent To"`
	TransportReference    string           `json:"transportReference"    bun:"transport_reference,type:TEXT,nullzero"              keel:"transport_reference,nullzero,label=Delivery Reference"`
	SendAttemptGroup      int              `json:"sendAttemptGroup"      bun:"send_attempt_group,type:INTEGER,notnull,default:1"   keel:"send_attempt_group,label=Send Number,dimension"`
	Attempts              int              `json:"attempts"              bun:"attempts,type:INTEGER,notnull,default:0"             keel:"attempts,dimension"`
	LastError             string           `json:"lastError"             bun:"last_error,type:TEXT,nullzero"                       keel:"last_error,nullzero"`
	IdempotencyKey        string           `json:"-"                     bun:"idempotency_key,type:VARCHAR(200),notnull"           keel:"idempotency_key,noreport"`
	SentAt                *int64           `json:"sentAt"                bun:"sent_at,type:BIGINT,nullzero"                        keel:"sent_at"`
	FailedAt              *int64           `json:"failedAt"              bun:"failed_at,type:BIGINT,nullzero"                      keel:"failed_at"`
	CancelledAt           *int64           `json:"cancelledAt"           bun:"cancelled_at,type:BIGINT,nullzero"                   keel:"cancelled_at"`
	CancelReason          string           `json:"cancelReason"          bun:"cancel_reason,type:VARCHAR(500),nullzero"            keel:"cancel_reason,nullzero"`
	FundedAt              *int64           `json:"fundedAt"              bun:"funded_at,type:BIGINT,nullzero"                      keel:"funded_at"`
	SubmittedByID         pulid.ID         `json:"submittedById"         bun:"submitted_by_id,type:VARCHAR(100),nullzero"          keel:"submitted_by_id,nullzero"`
	AgentRunID            pulid.ID         `json:"agentRunId"            bun:"agent_run_id,type:VARCHAR(100),nullzero"             keel:"agent_run_id,nullzero"`
	Version               int64            `json:"version"               bun:"version,type:BIGINT,notnull,default:0"               keel:"version,version"`
	CreatedAt             int64            `json:"createdAt"             bun:"created_at,type:BIGINT,notnull"                      keel:"created_at,autocreate"`
	UpdatedAt             int64            `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull"                      keel:"updated_at,autoupdate"`

	FactoringCompany *factoringcompany.FactoringCompany `json:"factoringCompany,omitempty" bun:"rel:belongs-to,join:factoring_company_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:factoring_company_id"`
	Agreement        *Agreement                         `json:"agreement,omitempty"        bun:"rel:belongs-to,join:agreement_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"         keel:"belongs:agreement_id"`
	Items            []*SubmissionItem                  `json:"items,omitempty"            bun:"rel:has-many,join:id=submission_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"          keel:"many:submission_id"`
}

// Each invoice on a factoring schedule, with what the factor was expected to advance, charge and hold back on it
type SubmissionItem struct {
	bun.BaseModel `bun:"table:factoring_submission_items,alias:facsit" json:"-"`
	keel.Table    `                                                             keel:"factoring_submission_items,report=Billing,label=Factoring Schedule Invoice"`

	ID                   pulid.ID             `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"                   keel:"id,pk,autoid:facsit_"`
	OrganizationID       pulid.ID             `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"      keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID             `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"     keel:"business_unit_id,pk,tenant"`
	SubmissionID         pulid.ID             `json:"submissionId"         bun:"submission_id,type:VARCHAR(100),notnull"           keel:"submission_id,label=Schedule ID"`
	InvoiceID            pulid.ID             `json:"invoiceId"            bun:"invoice_id,type:VARCHAR(100),notnull"              keel:"invoice_id"`
	InvoiceNumber        string               `json:"invoiceNumber"        bun:"invoice_number,type:VARCHAR(100),notnull"          keel:"invoice_number"`
	CustomerID           pulid.ID             `json:"customerId"           bun:"customer_id,type:VARCHAR(100),notnull"             keel:"customer_id"`
	Position             int                  `json:"position"             bun:"position,type:INTEGER,notnull"                     keel:"position,label=Line,dimension"`
	FaceMinor            int64                `json:"faceMinor"            bun:"face_minor,type:BIGINT,notnull"                    keel:"face_minor,label=Face Amount (Minor)"`
	ExpectedAdvanceMinor int64                `json:"expectedAdvanceMinor" bun:"expected_advance_minor,type:BIGINT,notnull"        keel:"expected_advance_minor,label=Expected Advance (Minor)"`
	ExpectedFeeMinor     int64                `json:"expectedFeeMinor"     bun:"expected_fee_minor,type:BIGINT,notnull"            keel:"expected_fee_minor,label=Expected Fee (Minor)"`
	ExpectedReserveMinor int64                `json:"expectedReserveMinor" bun:"expected_reserve_minor,type:BIGINT,notnull"        keel:"expected_reserve_minor,label=Expected Reserve (Minor)"`
	ExpectedDays         int                  `json:"expectedDays"         bun:"expected_days,type:INTEGER,notnull"                keel:"expected_days,label=Expected Days to Pay,dimension"`
	PacketDocumentID     pulid.ID             `json:"packetDocumentId"     bun:"packet_document_id,type:VARCHAR(100),nullzero"     keel:"packet_document_id,nullzero"`
	Status               SubmissionItemStatus `json:"status"               bun:"status,type:VARCHAR(20),notnull,default:'Pending'" keel:"status"`
	ReleasedAt           *int64               `json:"releasedAt"           bun:"released_at,type:BIGINT,nullzero"                  keel:"released_at"`
	FundedOn             *int64               `json:"fundedOn"             bun:"funded_on,type:BIGINT,nullzero"                    keel:"funded_on,type=epoch"`
	AdvanceMinor         *int64               `json:"advanceMinor"         bun:"advance_minor,type:BIGINT,nullzero"                keel:"advance_minor,label=Advance (Minor)"`
	FeeMinor             *int64               `json:"feeMinor"             bun:"fee_minor,type:BIGINT,nullzero"                    keel:"fee_minor,label=Fee (Minor)"`
	ReserveMinor         *int64               `json:"reserveMinor"         bun:"reserve_minor,type:BIGINT,nullzero"                keel:"reserve_minor,label=Reserve (Minor)"`
	RejectionReason      string               `json:"rejectionReason"      bun:"rejection_reason,type:VARCHAR(500),nullzero"       keel:"rejection_reason,nullzero"`
	SettledOn            *int64               `json:"settledOn"            bun:"settled_on,type:BIGINT,nullzero"                   keel:"settled_on,type=epoch"`
	ChargedBackOn        *int64               `json:"chargedBackOn"        bun:"charged_back_on,type:BIGINT,nullzero"              keel:"charged_back_on,type=epoch"`
	ChargebackReason     string               `json:"chargebackReason"     bun:"chargeback_reason,type:VARCHAR(500),nullzero"      keel:"chargeback_reason,nullzero"`
	ChargebackWarnedAt   *int64               `json:"chargebackWarnedAt"   bun:"chargeback_warned_at,type:BIGINT,nullzero"         keel:"chargeback_warned_at"`
	CreatedAt            int64                `json:"createdAt"            bun:"created_at,type:BIGINT,notnull"                    keel:"created_at,autocreate"`
	UpdatedAt            int64                `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull"                    keel:"updated_at,autoupdate"`

	Invoice  *invoice.Invoice   `json:"invoice,omitempty"  bun:"rel:belongs-to,join:invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"  keel:"belongs:invoice_id"`
	Customer *customer.Customer `json:"customer,omitempty" bun:"rel:belongs-to,join:customer_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:customer_id"`
}

func (s *Submission) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		s,
		validation.Field(&s.OrganizationID, validation.Required),
		validation.Field(&s.BusinessUnitID, validation.Required),
		validation.Field(&s.Number, validation.Required.Error("Schedule number is required")),
		validation.Field(
			&s.FactoringCompanyID,
			validation.Required.Error("Factoring company is required"),
		),
		validation.Field(&s.AgreementID, validation.Required.Error("Agreement is required")),
		validation.Field(
			&s.Status,
			validation.Required,
			domainvalidation.ValidEnum[SubmissionStatus]("Invalid submission status"),
		),
		validation.Field(
			&s.TransportMethod,
			validation.Required,
			domainvalidation.ValidEnum[TransportMethod]("Invalid transport"),
		),
		validation.Field(
			&s.TransportTarget,
			validation.Required.Error("The schedule needs somewhere to go"),
			validation.Length(0, MaxTransportTargetLength),
		),
		validation.Field(&s.IdempotencyKey, validation.Required),
		validation.Field(
			&s.CancelReason,
			validation.Length(0, MaxSubmissionCancelReason).
				Error("Reason cannot be longer than 500 characters"),
		),
	))
	if s.InvoiceCount < 1 {
		multiErr.Add("invoiceIds", errortypes.ErrRequired, "Choose at least one invoice")
	}
	if s.InvoiceCount > MaxSubmissionInvoices {
		multiErr.Add(
			"invoiceIds",
			errortypes.ErrInvalid,
			"A schedule can hold at most 200 invoices",
		)
	}
}

func (s *Submission) Totalize() {
	s.InvoiceCount = 0
	s.FaceMinor = 0
	s.ExpectedAdvanceMinor = 0
	s.ExpectedFeeMinor = 0
	s.ExpectedReserveMinor = 0
	for _, item := range s.Items {
		if item == nil {
			continue
		}
		s.InvoiceCount++
		s.FaceMinor += item.FaceMinor
		s.ExpectedAdvanceMinor += item.ExpectedAdvanceMinor
		s.ExpectedFeeMinor += item.ExpectedFeeMinor
		s.ExpectedReserveMinor += item.ExpectedReserveMinor
	}
}

func (s *Submission) IsLive() bool {
	return s != nil && s.Status.IsLive()
}

func (s *Submission) CanRetry() bool {
	return s != nil && s.Status == SubmissionStatusFailed
}

func (s *Submission) CanCancel() bool {
	return s != nil && s.Status == SubmissionStatusFailed
}

func (s *Submission) MarkSent(at int64, reference string, attempts int) error {
	if s.Status != SubmissionStatusSending {
		return errortypes.NewBusinessError("Only a schedule being sent can be marked sent")
	}
	s.Status = SubmissionStatusSent
	s.SentAt = &at
	s.FailedAt = nil
	s.TransportReference = reference
	s.Attempts = attempts
	s.LastError = ""
	return nil
}

func (s *Submission) MarkFailed(at int64, cause string, attempts int) error {
	if s.Status != SubmissionStatusSending {
		return errortypes.NewBusinessError("Only a schedule being sent can be marked failed")
	}
	s.Status = SubmissionStatusFailed
	s.FailedAt = &at
	s.Attempts = attempts
	s.LastError = stringutils.TruncateBytes(strings.TrimSpace(cause), MaxSubmissionErrorLength)
	if s.LastError == "" {
		s.LastError = "The factor could not be reached"
	}
	return nil
}

func (s *Submission) Retry() error {
	if !s.CanRetry() {
		return errortypes.NewBusinessError("Only a schedule that failed to send can be retried")
	}
	s.Status = SubmissionStatusSending
	s.SendAttemptGroup++
	s.Attempts = 0
	s.FailedAt = nil
	return nil
}

func (s *Submission) Cancel(at int64, reason string) error {
	if !s.CanCancel() {
		return errortypes.NewBusinessError(
			"Only a schedule that failed to send can be cancelled; a sent schedule is with the factor",
		)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Say why the schedule is being cancelled",
		)
	}
	s.Status = SubmissionStatusCancelled
	s.CancelledAt = &at
	s.CancelReason = reason
	for _, item := range s.Items {
		if item != nil && item.Status == SubmissionItemStatusPending {
			item.Status = SubmissionItemStatusReleased
			item.ReleasedAt = &at
		}
	}
	return nil
}

func (s *Submission) CanFund() bool {
	return s != nil && s.Status.AwaitsFunding()
}

func (s *Submission) Item(id pulid.ID) *SubmissionItem {
	for _, item := range s.Items {
		if item != nil && item.ID == id {
			return item
		}
	}
	return nil
}

func (s *Submission) Resolve(at int64) {
	if s == nil || !s.Status.AwaitsFunding() {
		return
	}
	pending, funded := 0, 0
	for _, item := range s.Items {
		if item == nil {
			continue
		}
		switch item.Status {
		case SubmissionItemStatusPending:
			pending++
		case SubmissionItemStatusFunded, SubmissionItemStatusSettled,
			SubmissionItemStatusChargedBack:
			funded++
		case SubmissionItemStatusReleased, SubmissionItemStatusRejected:
		}
	}
	switch {
	case funded > 0 && pending > 0:
		s.Status = SubmissionStatusPartially
	case funded > 0:
		s.Status = SubmissionStatusFunded
	case pending == 0:
		s.Status = SubmissionStatusRejected
	default:
		return
	}
	if funded > 0 && s.FundedAt == nil {
		s.FundedAt = &at
	}
}

func (s *Submission) InvoiceIDs() []pulid.ID {
	ids := make([]pulid.ID, 0, len(s.Items))
	for _, item := range s.Items {
		if item != nil {
			ids = append(ids, item.InvoiceID)
		}
	}
	return ids
}

func (s *Submission) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("facsub_")
		}
		if s.Status == "" {
			s.Status = SubmissionStatusSending
		}
		if s.SendAttemptGroup < 1 {
			s.SendAttemptGroup = 1
		}
		s.CreatedAt = now
		s.UpdatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}
	return nil
}

func (s *Submission) BeforeCreate(ctx context.Context) error {
	return s.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}

func (s *Submission) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "facsub",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "number", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
		},
	}
}

func (s *Submission) GetID() pulid.ID             { return s.ID }
func (s *Submission) GetCreatedAt() int64         { return s.CreatedAt }
func (s *Submission) GetOrganizationID() pulid.ID { return s.OrganizationID }
func (s *Submission) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }
func (s *Submission) GetTableName() string        { return "factoring_submissions" }

type ItemFunding struct {
	AdvanceMinor int64
	FeeMinor     int64
}

func (i *SubmissionItem) ExpectedFunding() ItemFunding {
	return ItemFunding{AdvanceMinor: i.ExpectedAdvanceMinor, FeeMinor: i.ExpectedFeeMinor}
}

func (i *SubmissionItem) Fund(fundedOn int64, funding ItemFunding) error {
	if i.Status != SubmissionItemStatusPending {
		return errortypes.NewBusinessError(
			"Invoice {0} is not waiting on the factor",
			i.InvoiceNumber,
		)
	}
	if funding.AdvanceMinor < 0 || funding.FeeMinor < 0 {
		return errortypes.NewValidationError(
			"items",
			errortypes.ErrInvalid,
			"The advance and fee for invoice {0} cannot be negative",
			i.InvoiceNumber,
		)
	}
	reserve := i.FaceMinor - funding.AdvanceMinor - funding.FeeMinor
	if reserve < 0 {
		return errortypes.NewValidationError(
			"items",
			errortypes.ErrInvalid,
			"The advance and fee for invoice {0} come to more than the invoice",
			i.InvoiceNumber,
		)
	}
	i.Status = SubmissionItemStatusFunded
	i.FundedOn = &fundedOn
	i.AdvanceMinor = &funding.AdvanceMinor
	i.FeeMinor = &funding.FeeMinor
	i.ReserveMinor = &reserve
	return nil
}

func (i *SubmissionItem) Reject(reason string) error {
	if i.Status != SubmissionItemStatusPending {
		return errortypes.NewBusinessError(
			"Invoice {0} is not waiting on the factor",
			i.InvoiceNumber,
		)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errortypes.NewValidationError(
			"items",
			errortypes.ErrRequired,
			"Say why the factor rejected invoice {0}",
			i.InvoiceNumber,
		)
	}
	i.Status = SubmissionItemStatusRejected
	i.RejectionReason = stringutils.TruncateRunes(reason, MaxFundingReasonLength)
	return nil
}

func (i *SubmissionItem) Settle(settledOn int64) error {
	if i.Status != SubmissionItemStatusFunded {
		return errortypes.NewBusinessError(
			"Invoice {0} is not funded, so it has no reserve to release",
			i.InvoiceNumber,
		)
	}
	i.Status = SubmissionItemStatusSettled
	i.SettledOn = &settledOn
	return nil
}

func (i *SubmissionItem) ChargeBack(chargedBackOn int64, reason string) error {
	if i.Status != SubmissionItemStatusFunded {
		return errortypes.NewBusinessError(
			"Invoice {0} is not funded and open with the factor, so it cannot be charged back",
			i.InvoiceNumber,
		)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Say why the factor charged the invoice back",
		)
	}
	i.Status = SubmissionItemStatusChargedBack
	i.ChargedBackOn = &chargedBackOn
	i.ChargebackReason = stringutils.TruncateRunes(reason, MaxFundingReasonLength)
	return nil
}

func (i *SubmissionItem) ReverseChargeback() error {
	if i.Status != SubmissionItemStatusChargedBack {
		return errortypes.NewBusinessError(
			"Invoice {0} has no chargeback to reverse",
			i.InvoiceNumber,
		)
	}
	i.Status = SubmissionItemStatusFunded
	i.ChargedBackOn = nil
	i.ChargebackReason = ""
	return nil
}

func (i *SubmissionItem) Actuals() (advance, fee, reserve int64) {
	if i.AdvanceMinor != nil {
		advance = *i.AdvanceMinor
	}
	if i.FeeMinor != nil {
		fee = *i.FeeMinor
	}
	if i.ReserveMinor != nil {
		reserve = *i.ReserveMinor
	}
	return advance, fee, reserve
}

func (i *SubmissionItem) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("facsit_")
		}
		if i.Status == "" {
			i.Status = SubmissionItemStatusPending
		}
		i.CreatedAt = now
		i.UpdatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}
	return nil
}

func (i *SubmissionItem) BeforeCreate(ctx context.Context) error {
	return i.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}
