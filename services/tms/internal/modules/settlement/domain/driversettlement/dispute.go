package driversettlement

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ pagination.CursorEntity            = (*Dispute)(nil)
	_ validationframework.TenantedEntity = (*Dispute)(nil)
	_ domaintypes.PostgresSearchable     = (*Dispute)(nil)
)

type DisputeStatus string

const (
	DisputeStatusOpen      = DisputeStatus("Open")
	DisputeStatusInReview  = DisputeStatus("InReview")
	DisputeStatusResolved  = DisputeStatus("Resolved")
	DisputeStatusDenied    = DisputeStatus("Denied")
	DisputeStatusWithdrawn = DisputeStatus("Withdrawn")
)

func (s DisputeStatus) String() string { return string(s) }

func (s DisputeStatus) IsValid() bool {
	switch s {
	case DisputeStatusOpen, DisputeStatusInReview, DisputeStatusResolved,
		DisputeStatusDenied, DisputeStatusWithdrawn:
		return true
	default:
		return false
	}
}

func (s DisputeStatus) IsTerminal() bool {
	return s == DisputeStatusResolved || s == DisputeStatusDenied || s == DisputeStatusWithdrawn
}

type DisputeCategory string

const (
	DisputeCategoryMissingPay           = DisputeCategory("MissingPay")
	DisputeCategoryIncorrectRate        = DisputeCategory("IncorrectRate")
	DisputeCategoryIncorrectDeduction   = DisputeCategory("IncorrectDeduction")
	DisputeCategoryMissingReimbursement = DisputeCategory("MissingReimbursement")
	DisputeCategoryOther                = DisputeCategory("Other")
)

func (c DisputeCategory) String() string { return string(c) }

func (c DisputeCategory) IsValid() bool {
	switch c {
	case DisputeCategoryMissingPay, DisputeCategoryIncorrectRate,
		DisputeCategoryIncorrectDeduction, DisputeCategoryMissingReimbursement,
		DisputeCategoryOther:
		return true
	default:
		return false
	}
}

type Dispute struct {
	keel.Table `keel:"settlement_disputes"`

	ID                pulid.ID        `json:"id" keel:"id,pk,autoid:dsd_"`
	BusinessUnitID    pulid.ID        `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID        `json:"organizationId" keel:"organization_id,pk,tenant"`
	SettlementID      pulid.ID        `json:"settlementId" keel:"settlement_id"`
	SettlementLineID  *pulid.ID       `json:"settlementLineId" keel:"settlement_line_id"`
	WorkerID          pulid.ID        `json:"workerId" keel:"worker_id"`
	Status            DisputeStatus   `json:"status" keel:"status"`
	Category          DisputeCategory `json:"category" keel:"category"`
	Description       string          `json:"description" keel:"description"`
	SubmittedByUserID pulid.ID        `json:"submittedByUserId" keel:"submitted_by_user_id"`
	ResolutionNote    string          `json:"resolutionNote" keel:"resolution_note,nullzero"`
	ResolutionLineID  *pulid.ID       `json:"resolutionLineId" keel:"resolution_line_id"`
	ResolvedByID      *pulid.ID       `json:"resolvedById" keel:"resolved_by_id"`
	ResolvedAt        *int64          `json:"resolvedAt" keel:"resolved_at"`
	Version           int64           `json:"version" keel:"version,version"`
	CreatedAt         int64           `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt         int64           `json:"updatedAt" keel:"updated_at,autoupdate"`

	Settlement     *Settlement     `json:"settlement,omitempty" keel:"belongs:settlement_id"`
	SettlementLine *SettlementLine `json:"settlementLine,omitempty" keel:"belongs:settlement_line_id"`
	Worker         *worker.Worker  `json:"worker,omitempty" keel:"belongs:worker_id"`
	ResolvedBy     *tenant.User    `json:"resolvedBy,omitempty" keel:"belongs:resolved_by_id"`
}

func (d *Dispute) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(d,
		validation.Field(&d.SettlementID,
			validation.Required.Error("Settlement is required"),
		),
		validation.Field(&d.WorkerID,
			validation.Required.Error("Worker is required"),
		),
		validation.Field(&d.Description,
			validation.Required.Error("Description is required"),
			validation.Length(1, 4000).Error("Description must be between 1 and 4000 characters"),
		),
		validation.Field(&d.SubmittedByUserID,
			validation.Required.Error("Submitting user is required"),
		),
	))

	if !d.Status.IsValid() {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			"Status must be Open, InReview, Resolved, Denied, or Withdrawn",
		)
	}
	if !d.Category.IsValid() {
		multiErr.Add(
			"category",
			errortypes.ErrInvalid,
			"Category must be MissingPay, IncorrectRate, IncorrectDeduction, MissingReimbursement, or Other",
		)
	}
}

func (d *Dispute) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dsd",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "resolution_note",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (d *Dispute) GetID() pulid.ID { return d.ID }

func (d *Dispute) GetCreatedAt() int64 { return d.CreatedAt }

func (d *Dispute) GetOrganizationID() pulid.ID { return d.OrganizationID }

func (d *Dispute) GetBusinessUnitID() pulid.ID { return d.BusinessUnitID }

func (d *Dispute) GetTableName() string { return "settlement_disputes" }

func (d *Dispute) BeforeCreate(ctx context.Context) error {
	if d.Status == "" {
		d.Status = DisputeStatusOpen
	}

	return nil
}
