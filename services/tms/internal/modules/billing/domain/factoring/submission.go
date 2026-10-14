package factoring

import (
	"context"
	"strings"

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
)

var (
	_ bun.BeforeAppendModelHook = (*Submission)(nil)
	_ bun.BeforeAppendModelHook = (*SubmissionItem)(nil)
)

type Submission struct {
	bun.BaseModel             `bun:"table:factoring_submissions,alias:facsub" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                   json:"-"`

	ID                    pulid.ID         `json:"id"                    bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID        pulid.ID         `json:"organizationId"        bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID        pulid.ID         `json:"businessUnitId"        bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	Number                string           `json:"number"                bun:"number,type:VARCHAR(100),notnull"`
	FactoringCompanyID    pulid.ID         `json:"factoringCompanyId"    bun:"factoring_company_id,type:VARCHAR(100),notnull"`
	AgreementID           pulid.ID         `json:"agreementId"           bun:"agreement_id,type:VARCHAR(100),notnull"`
	Status                SubmissionStatus `json:"status"                bun:"status,type:VARCHAR(20),notnull,default:'Sending'"`
	CurrencyCode          string           `json:"currencyCode"          bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	InvoiceCount          int              `json:"invoiceCount"          bun:"invoice_count,type:INTEGER,notnull"`
	FaceMinor             int64            `json:"faceMinor"             bun:"face_minor,type:BIGINT,notnull"`
	ExpectedAdvanceMinor  int64            `json:"expectedAdvanceMinor"  bun:"expected_advance_minor,type:BIGINT,notnull"`
	ExpectedFeeMinor      int64            `json:"expectedFeeMinor"      bun:"expected_fee_minor,type:BIGINT,notnull"`
	ExpectedReserveMinor  int64            `json:"expectedReserveMinor"  bun:"expected_reserve_minor,type:BIGINT,notnull"`
	SchedulePDFDocumentID pulid.ID         `json:"schedulePdfDocumentId" bun:"schedule_pdf_document_id,type:VARCHAR(100),nullzero"`
	ScheduleCSVDocumentID pulid.ID         `json:"scheduleCsvDocumentId" bun:"schedule_csv_document_id,type:VARCHAR(100),nullzero"`
	TransportMethod       TransportMethod  `json:"transportMethod"       bun:"transport_method,type:VARCHAR(20),notnull"`
	TransportTarget       string           `json:"transportTarget"       bun:"transport_target,type:VARCHAR(1000),notnull"`
	TransportReference    string           `json:"transportReference"    bun:"transport_reference,type:TEXT,nullzero"`
	SendAttemptGroup      int              `json:"sendAttemptGroup"      bun:"send_attempt_group,type:INTEGER,notnull,default:1"`
	Attempts              int              `json:"attempts"              bun:"attempts,type:INTEGER,notnull,default:0"`
	LastError             string           `json:"lastError"             bun:"last_error,type:TEXT,nullzero"`
	IdempotencyKey        string           `json:"-"                     bun:"idempotency_key,type:VARCHAR(200),notnull"`
	SentAt                *int64           `json:"sentAt"                bun:"sent_at,type:BIGINT,nullzero"`
	FailedAt              *int64           `json:"failedAt"              bun:"failed_at,type:BIGINT,nullzero"`
	CancelledAt           *int64           `json:"cancelledAt"           bun:"cancelled_at,type:BIGINT,nullzero"`
	CancelReason          string           `json:"cancelReason"          bun:"cancel_reason,type:VARCHAR(500),nullzero"`
	SubmittedByID         pulid.ID         `json:"submittedById"         bun:"submitted_by_id,type:VARCHAR(100),nullzero"`
	AgentRunID            pulid.ID         `json:"agentRunId"            bun:"agent_run_id,type:VARCHAR(100),nullzero"`
	Version               int64            `json:"version"               bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt             int64            `json:"createdAt"             bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt             int64            `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull"`

	FactoringCompany *factoringcompany.FactoringCompany `json:"factoringCompany,omitempty" bun:"rel:belongs-to,join:factoring_company_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Agreement        *Agreement                         `json:"agreement,omitempty"        bun:"rel:belongs-to,join:agreement_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Items            []*SubmissionItem                  `json:"items,omitempty"            bun:"rel:has-many,join:id=submission_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

type SubmissionItem struct {
	bun.BaseModel `bun:"table:factoring_submission_items,alias:facsit" json:"-"`

	ID                   pulid.ID             `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID             `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID             `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	SubmissionID         pulid.ID             `json:"submissionId"         bun:"submission_id,type:VARCHAR(100),notnull"`
	InvoiceID            pulid.ID             `json:"invoiceId"            bun:"invoice_id,type:VARCHAR(100),notnull"`
	InvoiceNumber        string               `json:"invoiceNumber"        bun:"invoice_number,type:VARCHAR(100),notnull"`
	CustomerID           pulid.ID             `json:"customerId"           bun:"customer_id,type:VARCHAR(100),notnull"`
	Position             int                  `json:"position"             bun:"position,type:INTEGER,notnull"`
	FaceMinor            int64                `json:"faceMinor"            bun:"face_minor,type:BIGINT,notnull"`
	ExpectedAdvanceMinor int64                `json:"expectedAdvanceMinor" bun:"expected_advance_minor,type:BIGINT,notnull"`
	ExpectedFeeMinor     int64                `json:"expectedFeeMinor"     bun:"expected_fee_minor,type:BIGINT,notnull"`
	ExpectedReserveMinor int64                `json:"expectedReserveMinor" bun:"expected_reserve_minor,type:BIGINT,notnull"`
	ExpectedDays         int                  `json:"expectedDays"         bun:"expected_days,type:INTEGER,notnull"`
	PacketDocumentID     pulid.ID             `json:"packetDocumentId"     bun:"packet_document_id,type:VARCHAR(100),nullzero"`
	Status               SubmissionItemStatus `json:"status"               bun:"status,type:VARCHAR(20),notnull,default:'Pending'"`
	ReleasedAt           *int64               `json:"releasedAt"           bun:"released_at,type:BIGINT,nullzero"`
	CreatedAt            int64                `json:"createdAt"            bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt            int64                `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull"`

	Invoice  *invoice.Invoice   `json:"invoice,omitempty"  bun:"rel:belongs-to,join:invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Customer *customer.Customer `json:"customer,omitempty" bun:"rel:belongs-to,join:customer_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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
