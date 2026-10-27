package driverpay

import (
	"context"

	"github.com/emoss08/trenova/shared/money"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ pagination.CursorEntity            = (*PayAdvance)(nil)
	_ validationframework.TenantedEntity = (*PayAdvance)(nil)
)

// Cash and money-code advances, what has been recovered, and what was written off
type PayAdvance struct {
	keel.Table `keel:"pay_advances,report=Payroll"`

	ID              pulid.ID      `json:"id"              keel:"id,pk,autoid:padv_"`
	BusinessUnitID  pulid.ID      `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID      `json:"organizationId"  keel:"organization_id,pk,tenant"`
	WorkerID        pulid.ID      `json:"workerId"        keel:"worker_id"`
	Status          AdvanceStatus `json:"status"          keel:"status"`
	Source          AdvanceSource `json:"source"          keel:"source"`
	Reference       string        `json:"reference"       keel:"reference,nullzero"`
	IssuedDate      int64         `json:"issuedDate"      keel:"issued_date,label=Issued"`
	AmountMinor     int64         `json:"amountMinor"     keel:"amount_minor,label=Amount (Minor)"`
	RecoveredMinor  int64         `json:"recoveredMinor"  keel:"recovered_minor,label=Recovered (Minor)"`
	WrittenOffMinor int64         `json:"writtenOffMinor" keel:"written_off_minor,label=Written Off (Minor)"`
	WriteOffReason  string        `json:"writeOffReason"  keel:"write_off_reason,nullzero,label=Write-Off Reason"`
	Notes           string        `json:"notes"           keel:"notes,nullzero"`
	CurrencyCode    string        `json:"currencyCode"    keel:"currency_code,label=Currency"`
	CreatedByID     pulid.ID      `json:"createdById"     keel:"created_by_id,nullzero"`
	WrittenOffByID  pulid.ID      `json:"writtenOffById"  keel:"written_off_by_id,nullzero"`
	WrittenOffAt    *int64        `json:"writtenOffAt"    keel:"written_off_at"`
	Version         int64         `json:"version"         keel:"version,version"`
	CreatedAt       int64         `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt       int64         `json:"updatedAt"       keel:"updated_at,autoupdate"`

	Worker *worker.Worker `json:"worker,omitempty" keel:"belongs:worker_id,label=Driver"`
}

func (a *PayAdvance) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&a.IssuedDate, validation.Required.Error("Issued date is required")),
		validation.Field(&a.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Length(3, 3).Error("Currency code must be 3 characters"),
		),
	))

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

func (a *PayAdvance) BeforeCreate(ctx context.Context) error {
	if a.Status == "" {
		a.Status = AdvanceStatusOutstanding
	}
	if a.CurrencyCode == "" {
		a.CurrencyCode = money.DefaultCurrencyCode
	}

	return nil
}
