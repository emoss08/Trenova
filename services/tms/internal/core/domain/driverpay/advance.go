package driverpay

import (
	"context"
	"errors"

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
	_ bun.BeforeAppendModelHook          = (*PayAdvance)(nil)
	_ pagination.CursorEntity            = (*PayAdvance)(nil)
	_ validationframework.TenantedEntity = (*PayAdvance)(nil)
)

type PayAdvance struct {
	bun.BaseModel             `bun:"table:pay_advances,alias:padv" json:"-"`
	pagination.CursorValueSet `bun:",embed"                        json:"-"`

	ID              pulid.ID      `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID      `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID  pulid.ID      `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID        pulid.ID      `json:"workerId"        bun:"worker_id,type:VARCHAR(100),notnull"`
	Status          AdvanceStatus `json:"status"          bun:"status,type:VARCHAR(50),notnull,default:'Outstanding'"`
	Source          AdvanceSource `json:"source"          bun:"source,type:VARCHAR(50),notnull"`
	Reference       string        `json:"reference"       bun:"reference,type:VARCHAR(100),nullzero"`
	IssuedDate      int64         `json:"issuedDate"      bun:"issued_date,type:BIGINT,notnull"`
	AmountMinor     int64         `json:"amountMinor"     bun:"amount_minor,type:BIGINT,notnull"`
	RecoveredMinor  int64         `json:"recoveredMinor"  bun:"recovered_minor,type:BIGINT,notnull,default:0"`
	WrittenOffMinor int64         `json:"writtenOffMinor" bun:"written_off_minor,type:BIGINT,notnull,default:0"`
	WriteOffReason  string        `json:"writeOffReason"  bun:"write_off_reason,type:TEXT,nullzero"`
	Notes           string        `json:"notes"           bun:"notes,type:TEXT,nullzero"`
	CurrencyCode    string        `json:"currencyCode"    bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	CreatedByID     pulid.ID      `json:"createdById"     bun:"created_by_id,type:VARCHAR(100),nullzero"`
	WrittenOffByID  pulid.ID      `json:"writtenOffById"  bun:"written_off_by_id,type:VARCHAR(100),nullzero"`
	WrittenOffAt    *int64        `json:"writtenOffAt"    bun:"written_off_at,type:BIGINT,nullzero"`
	Version         int64         `json:"version"         bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt       int64         `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt       int64         `json:"updatedAt"       bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker *worker.Worker `json:"worker,omitempty" bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (a *PayAdvance) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(a,
		validation.Field(&a.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&a.IssuedDate, validation.Required.Error("Issued date is required")),
		validation.Field(&a.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Length(3, 3).Error("Currency code must be 3 characters"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if !a.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Advance status is invalid")
	}
	if !a.Source.IsValid() {
		multiErr.Add("source", errortypes.ErrInvalid, "Advance source is invalid")
	}
	if a.AmountMinor <= 0 {
		multiErr.Add(
			"amountMinor",
			errortypes.ErrInvalid,
			"Advance amount must be greater than zero",
		)
	}
	if a.RecoveredMinor < 0 {
		multiErr.Add(
			"recoveredMinor",
			errortypes.ErrInvalid,
			"Recovered amount cannot be negative",
		)
	}
	if a.WrittenOffMinor < 0 {
		multiErr.Add(
			"writtenOffMinor",
			errortypes.ErrInvalid,
			"Written off amount cannot be negative",
		)
	}
	if a.RecoveredMinor+a.WrittenOffMinor > a.AmountMinor {
		multiErr.Add(
			"recoveredMinor",
			errortypes.ErrInvalid,
			"Recovered plus written off amounts cannot exceed the advance amount",
		)
	}
}

func (a *PayAdvance) OutstandingMinor() int64 {
	return max(a.AmountMinor-a.RecoveredMinor-a.WrittenOffMinor, 0)
}

func (a *PayAdvance) SyncStatus() {
	switch {
	case a.WrittenOffMinor > 0 && a.OutstandingMinor() == 0:
		a.Status = AdvanceStatusWrittenOff
	case a.OutstandingMinor() == 0:
		a.Status = AdvanceStatusRecovered
	case a.RecoveredMinor > 0:
		a.Status = AdvanceStatusPartiallyRecovered
	default:
		a.Status = AdvanceStatusOutstanding
	}
}

func (a *PayAdvance) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "padv",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "reference",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{Name: "notes", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
		},
	}
}

func (a *PayAdvance) GetID() pulid.ID { return a.ID }

func (a *PayAdvance) GetCreatedAt() int64 { return a.CreatedAt }

func (a *PayAdvance) GetOrganizationID() pulid.ID { return a.OrganizationID }

func (a *PayAdvance) GetBusinessUnitID() pulid.ID { return a.BusinessUnitID }

func (a *PayAdvance) GetTableName() string { return "pay_advances" }

func (a *PayAdvance) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("padv_")
		}
		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}
	return nil
}
