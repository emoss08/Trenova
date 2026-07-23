package driversettlement

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
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
	_ bun.BeforeAppendModelHook          = (*Dispute)(nil)
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
	bun.BaseModel             `bun:"table:settlement_disputes,alias:dsd" json:"-"`
	pagination.CursorValueSet `bun:",embed"                              json:"-"`

	ID                pulid.ID        `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID        `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID        `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	SettlementID      pulid.ID        `json:"settlementId"      bun:"settlement_id,type:VARCHAR(100),notnull"`
	SettlementLineID  *pulid.ID       `json:"settlementLineId"  bun:"settlement_line_id,type:VARCHAR(100),nullzero"`
	WorkerID          pulid.ID        `json:"workerId"          bun:"worker_id,type:VARCHAR(100),notnull"`
	Status            DisputeStatus   `json:"status"            bun:"status,type:VARCHAR(20),notnull,default:'Open'"`
	Category          DisputeCategory `json:"category"          bun:"category,type:VARCHAR(30),notnull"`
	Description       string          `json:"description"       bun:"description,type:TEXT,notnull"`
	SubmittedByUserID pulid.ID        `json:"submittedByUserId" bun:"submitted_by_user_id,type:VARCHAR(100),notnull"`
	ResolutionNote    string          `json:"resolutionNote"    bun:"resolution_note,type:TEXT,nullzero"`
	ResolutionLineID  *pulid.ID       `json:"resolutionLineId"  bun:"resolution_line_id,type:VARCHAR(100),nullzero"`
	ResolvedByID      *pulid.ID       `json:"resolvedById"      bun:"resolved_by_id,type:VARCHAR(100),nullzero"`
	ResolvedAt        *int64          `json:"resolvedAt"        bun:"resolved_at,type:BIGINT,nullzero"`
	Version           int64           `json:"version"           bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt         int64           `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64           `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Settlement     *Settlement     `json:"settlement,omitempty"     bun:"rel:belongs-to,join:settlement_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	SettlementLine *SettlementLine `json:"settlementLine,omitempty" bun:"rel:belongs-to,join:settlement_line_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Worker         *worker.Worker  `json:"worker,omitempty"         bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	ResolvedBy     *tenant.User    `json:"resolvedBy,omitempty"     bun:"rel:belongs-to,join:resolved_by_id=id"`
}

func (d *Dispute) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(d,
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
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

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

func (d *Dispute) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("dsd_")
		}
		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}
	return nil
}
