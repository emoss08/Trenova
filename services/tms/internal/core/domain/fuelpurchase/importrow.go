package fuelpurchase

import (
	"context"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*ImportRow)(nil)
	_ validationframework.TenantedEntity = (*ImportRow)(nil)
)

type ImportRow struct {
	bun.BaseModel `bun:"table:fuel_purchase_import_rows,alias:fpir" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ImportBatchID  pulid.ID `json:"importBatchId"  bun:"import_batch_id,type:VARCHAR(100),notnull"`

	RowNumber int           `json:"rowNumber" bun:"row_number,type:INTEGER,notnull"`
	Cells     []string      `json:"cells"     bun:"cells,type:JSONB,nullzero"`
	Parsed    *FuelPurchase `json:"parsed"    bun:"parsed,type:JSONB,nullzero"`

	TransactionReference string          `json:"transactionReference" bun:"transaction_reference,type:VARCHAR(150),nullzero"`
	Status               ImportRowStatus `json:"status"               bun:"status,type:fuel_import_row_status_enum,notnull,default:'New'"`
	Error                string          `json:"error"                bun:"error,type:TEXT,nullzero"`

	ResolvedTractorID      *pulid.ID `json:"resolvedTractorId"      bun:"resolved_tractor_id,type:VARCHAR(100),nullzero"`
	ResolvedFuelCardID     *pulid.ID `json:"resolvedFuelCardId"     bun:"resolved_fuel_card_id,type:VARCHAR(100),nullzero"`
	ResolvedJurisdictionID *pulid.ID `json:"resolvedJurisdictionId" bun:"resolved_jurisdiction_id,type:VARCHAR(100),nullzero"`
	ResolutionNotes        []string  `json:"resolutionNotes"        bun:"resolution_notes,type:JSONB,nullzero"`

	FuelPurchaseID *pulid.ID `json:"fuelPurchaseId" bun:"fuel_purchase_id,type:VARCHAR(100),nullzero"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Batch        *ImportBatch  `json:"-"                      bun:"rel:belongs-to,join:import_batch_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	FuelPurchase *FuelPurchase `json:"fuelPurchase,omitempty" bun:"rel:belongs-to,join:fuel_purchase_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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

func (r *ImportRow) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("fpir_")
		}
		if r.Status == "" {
			r.Status = ImportRowStatusNew
		}
		r.CreatedAt = timeutils.NowUnix()
	}

	return nil
}
