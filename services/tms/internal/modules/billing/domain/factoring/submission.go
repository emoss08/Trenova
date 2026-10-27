package factoring

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/shared/money"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	SubmissionDocumentResourceType = "factoring_submission"
	MaxSubmissionInvoices          = 200
	MaxSubmissionErrorLength       = 2000
	MaxSubmissionCancelReason      = 500
	MaxTransportTargetLength       = 1000
	MaxFundingReasonLength         = 500
)

// Schedules of accounts sent to each factor, with their invoices' total, expected advance, fee and reserve, and how delivery went
type Submission struct {
	keel.Table `keel:"factoring_submissions,report=Billing,label=Factoring Schedule"`

	ID                    pulid.ID         `json:"id"                    keel:"id,pk,autoid:facsub_"`
	OrganizationID        pulid.ID         `json:"organizationId"        keel:"organization_id,pk,tenant"`
	BusinessUnitID        pulid.ID         `json:"businessUnitId"        keel:"business_unit_id,pk,tenant"`
	Number                string           `json:"number"                keel:"number,label=Schedule Number"`
	FactoringCompanyID    pulid.ID         `json:"factoringCompanyId"    keel:"factoring_company_id"`
	AgreementID           pulid.ID         `json:"agreementId"           keel:"agreement_id"`
	Status                SubmissionStatus `json:"status"                keel:"status"`
	CurrencyCode          string           `json:"currencyCode"          keel:"currency_code,label=Currency"`
	InvoiceCount          int              `json:"invoiceCount"          keel:"invoice_count,label=Invoices,format=count"`
	FaceMinor             int64            `json:"faceMinor"             keel:"face_minor,label=Face Amount (Minor)"`
	ExpectedAdvanceMinor  int64            `json:"expectedAdvanceMinor"  keel:"expected_advance_minor,label=Expected Advance (Minor)"`
	ExpectedFeeMinor      int64            `json:"expectedFeeMinor"      keel:"expected_fee_minor,label=Expected Fee (Minor)"`
	ExpectedReserveMinor  int64            `json:"expectedReserveMinor"  keel:"expected_reserve_minor,label=Expected Reserve (Minor)"`
	SchedulePDFDocumentID pulid.ID         `json:"schedulePdfDocumentId" keel:"schedule_pdf_document_id,nullzero"`
	ScheduleCSVDocumentID pulid.ID         `json:"scheduleCsvDocumentId" keel:"schedule_csv_document_id,nullzero"`
	TransportMethod       TransportMethod  `json:"transportMethod"       keel:"transport_method,label=Sent By"`
	TransportTarget       string           `json:"transportTarget"       keel:"transport_target,label=Sent To"`
	TransportReference    string           `json:"transportReference"    keel:"transport_reference,nullzero,label=Delivery Reference"`
	SendAttemptGroup      int              `json:"sendAttemptGroup"      keel:"send_attempt_group,label=Send Number,dimension"`
	Attempts              int              `json:"attempts"              keel:"attempts,dimension"`
	LastError             string           `json:"lastError"             keel:"last_error,nullzero"`
	IdempotencyKey        string           `json:"-"                     keel:"idempotency_key,noreport"`
	SentAt                *int64           `json:"sentAt"                keel:"sent_at"`
	FailedAt              *int64           `json:"failedAt"              keel:"failed_at"`
	CancelledAt           *int64           `json:"cancelledAt"           keel:"cancelled_at"`
	CancelReason          string           `json:"cancelReason"          keel:"cancel_reason,nullzero"`
	FundedAt              *int64           `json:"fundedAt"              keel:"funded_at"`
	SubmittedByID         pulid.ID         `json:"submittedById"         keel:"submitted_by_id,nullzero"`
	AgentRunID            pulid.ID         `json:"agentRunId"            keel:"agent_run_id,nullzero"`
	Version               int64            `json:"version"               keel:"version,version"`
	CreatedAt             int64            `json:"createdAt"             keel:"created_at,autocreate"`
	UpdatedAt             int64            `json:"updatedAt"             keel:"updated_at,autoupdate"`

	FactoringCompany *factoringcompany.FactoringCompany `json:"factoringCompany,omitempty" keel:"belongs:factoring_company_id"`
	Agreement        *Agreement                         `json:"agreement,omitempty"        keel:"belongs:agreement_id"`
	Items            []*SubmissionItem                  `json:"items,omitempty"            keel:"many:submission_id"`
}

// Each invoice on a factoring schedule, with what the factor was expected to advance, charge and hold back on it
type SubmissionItem struct {
	keel.Table `keel:"factoring_submission_items,report=Billing,label=Factoring Schedule Invoice"`

	ID                   pulid.ID             `json:"id"                   keel:"id,pk,autoid:facsit_"`
	OrganizationID       pulid.ID             `json:"organizationId"       keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID             `json:"businessUnitId"       keel:"business_unit_id,pk,tenant"`
	SubmissionID         pulid.ID             `json:"submissionId"         keel:"submission_id,label=Schedule ID"`
	InvoiceID            pulid.ID             `json:"invoiceId"            keel:"invoice_id"`
	InvoiceNumber        string               `json:"invoiceNumber"        keel:"invoice_number"`
	CustomerID           pulid.ID             `json:"customerId"           keel:"customer_id"`
	Position             int                  `json:"position"             keel:"position,label=Line,dimension"`
	FaceMinor            int64                `json:"faceMinor"            keel:"face_minor,label=Face Amount (Minor)"`
	ExpectedAdvanceMinor int64                `json:"expectedAdvanceMinor" keel:"expected_advance_minor,label=Expected Advance (Minor)"`
	ExpectedFeeMinor     int64                `json:"expectedFeeMinor"     keel:"expected_fee_minor,label=Expected Fee (Minor)"`
	ExpectedReserveMinor int64                `json:"expectedReserveMinor" keel:"expected_reserve_minor,label=Expected Reserve (Minor)"`
	ExpectedDays         int                  `json:"expectedDays"         keel:"expected_days,label=Expected Days to Pay,dimension"`
	PacketDocumentID     pulid.ID             `json:"packetDocumentId"     keel:"packet_document_id,nullzero"`
	Status               SubmissionItemStatus `json:"status"               keel:"status"`
	ReleasedAt           *int64               `json:"releasedAt"           keel:"released_at"`
	FundedOn             *int64               `json:"fundedOn"             keel:"funded_on,type=epoch"`
	AdvanceMinor         *int64               `json:"advanceMinor"         keel:"advance_minor,label=Advance (Minor)"`
	FeeMinor             *int64               `json:"feeMinor"             keel:"fee_minor,label=Fee (Minor)"`
	ReserveMinor         *int64               `json:"reserveMinor"         keel:"reserve_minor,label=Reserve (Minor)"`
	RejectionReason      string               `json:"rejectionReason"      keel:"rejection_reason,nullzero"`
	SettledOn            *int64               `json:"settledOn"            keel:"settled_on,type=epoch"`
	ChargedBackOn        *int64               `json:"chargedBackOn"        keel:"charged_back_on,type=epoch"`
	ChargebackReason     string               `json:"chargebackReason"     keel:"chargeback_reason,nullzero"`
	ChargebackWarnedAt   *int64               `json:"chargebackWarnedAt"   keel:"chargeback_warned_at"`
	CreatedAt            int64                `json:"createdAt"            keel:"created_at,autocreate"`
	UpdatedAt            int64                `json:"updatedAt"            keel:"updated_at,autoupdate"`

	Invoice    *invoice.Invoice   `json:"invoice,omitempty"  keel:"belongs:invoice_id"`
	Customer   *customer.Customer `json:"customer,omitempty" keel:"belongs:customer_id"`
	Submission *Submission        `json:"-"                  keel:"belongs:submission_id,noreport"`
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

func (s *Submission) BeforeCreate(ctx context.Context) error {
	if s.Status == "" {
		s.Status = SubmissionStatusSending
	}
	if s.CurrencyCode == "" {
		s.CurrencyCode = money.DefaultCurrencyCode
	}
	if s.SendAttemptGroup == 0 {
		s.SendAttemptGroup = 1
	}
	if s.SendAttemptGroup < 1 {
		s.SendAttemptGroup = 1
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

type ItemFunding struct {
	AdvanceMinor int64
	FeeMinor     int64
}

func (Submission) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_factoring_submissions_idempotency",
			Message: "These invoices were just submitted; reload to see the schedule",
		},
		{
			Name:    "uq_factoring_submissions_number",
			Message: "That schedule number is already in use; check the factoring schedule sequence",
		},
	}
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

func (i *SubmissionItem) BeforeCreate(ctx context.Context) error {
	if i.Status == "" {
		i.Status = SubmissionItemStatusPending
	}

	return nil
}

func (SubmissionItem) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_factoring_submission_items_live_invoice",
			Message: "One of these invoices is already on a schedule with the factor; reload the queue and try again",
		},
	}
}
