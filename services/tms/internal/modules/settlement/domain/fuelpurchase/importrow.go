package fuelpurchase

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*ImportRow)(nil)
)

type ImportRow struct {
	keel.Table `keel:"fuel_purchase_import_rows"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:fpir_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	ImportBatchID  pulid.ID `json:"importBatchId"  keel:"import_batch_id"`

	RowNumber int           `json:"rowNumber" keel:"row_number"`
	Cells     []string      `json:"cells"     keel:"cells,jsonb,nullzero"`
	Parsed    *FuelPurchase `json:"parsed"    keel:"parsed,jsonb"`

	TransactionReference string          `json:"transactionReference" keel:"transaction_reference,nullzero"`
	Status               ImportRowStatus `json:"status"               keel:"status"`
	Error                string          `json:"error"                keel:"error,nullzero"`

	ResolvedTractorID      *pulid.ID `json:"resolvedTractorId"      keel:"resolved_tractor_id"`
	ResolvedFuelCardID     *pulid.ID `json:"resolvedFuelCardId"     keel:"resolved_fuel_card_id"`
	ResolvedJurisdictionID *pulid.ID `json:"resolvedJurisdictionId" keel:"resolved_jurisdiction_id"`
	ResolutionNotes        []string  `json:"resolutionNotes"        keel:"resolution_notes,jsonb,nullzero"`

	FuelPurchaseID *pulid.ID `json:"fuelPurchaseId" keel:"fuel_purchase_id"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`

	Batch        *ImportBatch  `json:"-"                      keel:"belongs:import_batch_id"`
	FuelPurchase *FuelPurchase `json:"fuelPurchase,omitempty" keel:"belongs:fuel_purchase_id"`
}

func (r *ImportRow) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.ImportBatchID,
			validation.Required.Error("Import batch is required"),
		),
		validation.Field(&r.RowNumber,
			validation.Min(1).Error("Row number must be at least 1"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ImportRowStatus]("Row status is not valid"),
		),
	))

	if r.Status == ImportRowStatusError && r.Error == "" {
		multiErr.Add("error", errortypes.ErrRequired, "An errored row must say what went wrong")
	}
	if r.Status == ImportRowStatusCommitted &&
		(r.FuelPurchaseID == nil || r.FuelPurchaseID.IsNil()) {
		multiErr.Add(
			"fuelPurchaseId",
			errortypes.ErrRequired,
			"A committed row must reference the purchase it created",
		)
	}
}

func (r *ImportRow) Failed() bool { return r.Status == ImportRowStatusError || r.Error != "" }

func (r *ImportRow) WillCommit() bool {
	return r.Status.WillCommit() && r.Parsed != nil && r.Error == ""
}

func (r *ImportRow) GetID() pulid.ID { return r.ID }

func (r *ImportRow) GetCreatedAt() int64 { return r.CreatedAt }

func (r *ImportRow) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *ImportRow) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *ImportRow) GetTableName() string { return "fuel_purchase_import_rows" }

func (r *ImportRow) BeforeCreate(ctx context.Context) error {
	if r.Status == "" {
		r.Status = ImportRowStatusNew
	}

	return nil
}
