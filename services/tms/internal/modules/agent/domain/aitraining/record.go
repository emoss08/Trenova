package aitraining

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*TrainingExportRecord)(nil)

type TrainingExportRecord struct {
	bun.BaseModel `bun:"table:ai_training_export_records,alias:aitr" json:"-"`
	keel.Table    `keel:"ai_training_export_records"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:aitr_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`

	ExportID         pulid.ID `json:"exportId"         bun:"export_id,type:VARCHAR(100),notnull" keel:"export_id"`
	CorrectionID     pulid.ID `json:"correctionId"     bun:"correction_id,type:VARCHAR(100),notnull" keel:"correction_id"`
	ExampleID        string   `json:"exampleId"        bun:"example_id,type:VARCHAR(64),notnull" keel:"example_id"`
	Split            Split    `json:"split"            bun:"split,type:VARCHAR(20),notnull" keel:"split"`
	ConsentGrantedAt int64    `json:"consentGrantedAt" bun:"consent_granted_at,type:BIGINT,notnull" keel:"consent_granted_at"`
	ExportedAt       int64    `json:"exportedAt"       bun:"exported_at,type:BIGINT,notnull" keel:"exported_at"`
	CreatedAt        int64    `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (r *TrainingExportRecord) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); !ok {
		return nil
	}
	if r.ID.IsNil() {
		r.ID = pulid.MustNew("aitr_")
	}
	if r.CreatedAt == 0 {
		r.CreatedAt = timeutils.NowUnix()
	}

	return nil
}

type ExportHistoryEntry struct {
	ExportID           pulid.ID     `json:"exportId"           bun:"export_id"`
	Status             ExportStatus `json:"status"             bun:"status"`
	Format             string       `json:"format"             bun:"format"`
	Examples           int          `json:"examples"           bun:"examples"`
	TrainExamples      int          `json:"trainExamples"      bun:"train_examples"`
	ValidationExamples int          `json:"validationExamples" bun:"validation_examples"`
	ConsentGrantedAt   int64        `json:"consentGrantedAt"   bun:"consent_granted_at"`
	ExportedAt         int64        `json:"exportedAt"         bun:"exported_at"`
}
