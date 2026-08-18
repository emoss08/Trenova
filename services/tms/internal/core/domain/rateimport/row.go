package rateimport

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RateImportRow)(nil)

// RateImportRow is one row of the uploaded sheet, as it was read.
//
// The raw cells are kept beside the parsed rule so an error can be shown next
// to what caused it. "Row 47 is invalid" sends somebody back to the spreadsheet
// to count lines; "row 47: 'per furlong' is not a rate type" does not.
type RateImportRow struct {
	bun.BaseModel `bun:"table:rate_import_rows,alias:rir" json:"-"`

	ID                pulid.ID `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RateImportBatchID pulid.ID `json:"rateImportBatchId" bun:"rate_import_batch_id,type:VARCHAR(100),notnull"`

	// RowNumber is the line in the uploaded file, counting the header as one,
	// so it matches what somebody sees in their spreadsheet.
	RowNumber int `json:"rowNumber" bun:"row_number,type:INTEGER,notnull"`

	Cells []string `json:"cells" bun:"cells,type:JSONB,nullzero"`

	// Rule is what the row parsed to, absent when it could not be read.
	Rule *rateagreement.RateAgreementRule `json:"rule" bun:"rule,type:JSONB,nullzero"`

	LaneKey string `json:"laneKey" bun:"lane_key,type:VARCHAR(255),nullzero"`

	Error string `json:"error" bun:"error,type:TEXT,nullzero"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Batch *RateImportBatch `json:"-" bun:"rel:belongs-to,join:rate_import_batch_id=id"`
}

// Failed reports whether this row could be read.
func (r *RateImportRow) Failed() bool { return r.Error != "" }

func (r *RateImportRow) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("rir_")
		}
		r.CreatedAt = timeutils.NowUnix()
	}

	return nil
}

func (r *RateImportRow) GetID() pulid.ID { return r.ID }

func (r *RateImportRow) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *RateImportRow) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *RateImportRow) GetTableName() string { return "rate_import_rows" }
