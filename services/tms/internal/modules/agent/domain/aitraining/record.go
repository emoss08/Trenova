package aitraining

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

type TrainingExportRecord struct {
	keel.Table `keel:"ai_training_export_records"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:aitr_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	ExportID         pulid.ID `json:"exportId"         keel:"export_id"`
	CorrectionID     pulid.ID `json:"correctionId"     keel:"correction_id"`
	ExampleID        string   `json:"exampleId"        keel:"example_id"`
	Split            Split    `json:"split"            keel:"split"`
	ConsentGrantedAt int64    `json:"consentGrantedAt" keel:"consent_granted_at"`
	ExportedAt       int64    `json:"exportedAt"       keel:"exported_at"`
	CreatedAt        int64    `json:"createdAt"        keel:"created_at,autocreate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

type ExportHistoryEntry struct {
	ExportID           pulid.ID     `json:"exportId"`
	Status             ExportStatus `json:"status"`
	Format             string       `json:"format"`
	Examples           int          `json:"examples"`
	TrainExamples      int          `json:"trainExamples"`
	ValidationExamples int          `json:"validationExamples"`
	ConsentGrantedAt   int64        `json:"consentGrantedAt"`
	ExportedAt         int64        `json:"exportedAt"`
}
