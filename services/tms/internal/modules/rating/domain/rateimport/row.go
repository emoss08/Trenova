package rateimport

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/shared/pulid"
)

// RateImportRow is one row of the uploaded sheet, as it was read.
//
// The raw cells are kept beside the parsed rule so an error can be shown next
// to what caused it. "Row 47 is invalid" sends somebody back to the spreadsheet
// to count lines; "row 47: 'per furlong' is not a rate type" does not.
type RateImportRow struct {
	keel.Table `keel:"rate_import_rows"`

	ID                pulid.ID `json:"id"                keel:"id,pk,autoid:rir_"`
	BusinessUnitID    pulid.ID `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID `json:"organizationId"    keel:"organization_id,pk,tenant"`
	RateImportBatchID pulid.ID `json:"rateImportBatchId" keel:"rate_import_batch_id"`

	// RowNumber is the line in the uploaded file, counting the header as one,
	// so it matches what somebody sees in their spreadsheet.
	RowNumber int `json:"rowNumber" keel:"row_number"`

	Cells []string `json:"cells" keel:"cells,jsonb,nullzero"`

	// Rule is what the row parsed to, absent when it could not be read.
	Rule *rateagreement.RateAgreementRule `json:"rule" keel:"rule,jsonb"`

	LaneKey string `json:"laneKey" keel:"lane_key,nullzero"`

	Error string `json:"error" keel:"error,nullzero"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`

	Batch *RateImportBatch `json:"-" keel:"belongs:rate_import_batch_id"`
}

// Failed reports whether this row could be read.
func (r *RateImportRow) Failed() bool { return r.Error != "" }

func (r *RateImportRow) GetID() pulid.ID { return r.ID }

func (r *RateImportRow) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *RateImportRow) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *RateImportRow) GetTableName() string { return "rate_import_rows" }
