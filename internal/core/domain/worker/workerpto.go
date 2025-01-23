package worker

import (
	"context"
	"fmt"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/trenova-app/transport/internal/core/domain/businessunit"
	"github.com/trenova-app/transport/internal/core/domain/organization"
	"github.com/trenova-app/transport/internal/core/domain/user"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Worker)(nil)

type WorkerPTO struct {
	bun.BaseModel `bun:"table:worker_pto,alias:wpto" json:"-"`

	// Primary identifiers
	ID             pulid.ID `json:"id" bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),notnull,pk" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),notnull,pk" json:"organizationId"`
	WorkerID       pulid.ID `bun:"worker_id,type:VARCHAR(100),notnull,pk" json:"workerId"`

	// Relationship identifiers (Non-Primary-Keys)
	ApproverID *pulid.ID `bun:"approver_id,type:VARCHAR(100),nullzero" json:"approverId"`

	// Core Fields
	Status    PTOStatus `json:"status" bun:"status,type:worker_pto_status_enum,notnull,default:'Requested'"`
	Type      PTOType   `json:"type" bun:"type,type:worker_pto_type_enum,notnull,default:'Vacation'"`
	StartDate int64     `json:"startDate" bun:"start_date,type:BIGINT,notnull"`
	EndDate   int64     `json:"endDate" bun:"end_date,type:BIGINT,notnull"`
	Reason    string    `json:"reason" bun:"reason,type:VARCHAR(255),notnull"`

	// Metadata
	Version   int64 `json:"version" bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Worker       *Worker                    `json:"worker,omitempty" bun:"rel:belongs-to,join:worker_id=id"`
	Approver     *user.User                 `json:"approver,omitempty" bun:"rel:belongs-to,join:approver_id=id"`
}

// Validation
func (w *WorkerPTO) Validate(ctx context.Context, multiErr *errors.MultiError, index int) {
	err := validation.ValidateStructWithContext(ctx, w,
		// Status is required and must be a valid PTO status
		validation.Field(&w.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				PTOStatusRequested,
				PTOStatusApproved,
				PTOStatusRejected,
				PTOStatusCancelled,
			).Error("Status must be a valid PTO status"),
		),

		// Approver ID is required when the status is approved.
		validation.Field(&w.ApproverID,
			validation.When(w.Status == PTOStatusApproved,
				validation.Required.Error("Approver ID is required when the status is approved")),
		),

		// Type is required and must be a valid PTO type
		validation.Field(&w.Type,
			validation.Required.Error("Type is required"),
			validation.In(
				PTOTypeVacation,
				PTOTypeSick,
				PTOTypeHoliday,
				PTOTypeBereavement,
				PTOTypeMaternity,
				PTOTypePaternity,
			).Error("Type must be a valid PTO type"),
		),

		// Start date is required and must be before the end date
		validation.Field(&w.StartDate,
			validation.Required.Error("Start date is required"),
			validation.Max(w.EndDate).Error("Start date cannot be after end date"),
		),

		// End date is required and must be after the start date
		validation.Field(&w.EndDate,
			validation.Required.Error("End date is required"),
			validation.Min(w.StartDate).Error("End date must be after start date"),
		),

		// Reason is required when the status is cancelled or rejected and Cannot input reason if the status is not cancelled or rejected
		validation.Field(&w.Reason,
			validation.When(w.Status == PTOStatusCancelled || w.Status == PTOStatusRejected,
				validation.Required.Error("Reason is required when the status is cancelled or rejected")),
			validation.When(w.Status != PTOStatusCancelled && w.Status != PTOStatusRejected,
				validation.Empty.Error("Reason cannot be input when the status is not cancelled or rejected")),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromValidationErrors(validationErrs, multiErr, fmt.Sprintf("pto[%d]", index))
		}
	}
}

// IsInvalid returns true if the PTO is cancelled, rejected, or both
// This indicates the PTO should not be considered for operations like overlap validation
func (w *WorkerPTO) IsInvalid() bool {
	return w.IsCancelled() || w.IsRejected()
}

func (w *WorkerPTO) IsCancelled() bool {
	return w.Status == PTOStatusCancelled
}

func (w *WorkerPTO) IsRejected() bool {
	return w.Status == PTOStatusRejected
}

func (w *WorkerPTO) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := time.Now().Unix()

	switch query.(type) {
	case *bun.InsertQuery:
		if w.ID == "" {
			w.ID = pulid.MustNew("pto_")
		}

		w.CreatedAt = now
	case *bun.UpdateQuery:
		w.UpdatedAt = now
	}

	return nil
}
