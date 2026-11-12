package workqueue

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

// Item is one subject waiting for background work of a kind: a shipment
// whose arrival times need projecting again, say. The row is the request, so
// a request made in a transaction is never lost between its commit and the
// signal that wakes the tenant's drain; the drain deletes it once the work is
// done, unless it was asked for again meanwhile.
type Item struct {
	keel.Table `keel:"work_queue_items"`

	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	Kind           string   `json:"kind"           keel:"kind,pk"`
	SubjectID      pulid.ID `json:"subjectId"      keel:"subject_id,pk"`
	DueAt          int64    `json:"dueAt"          keel:"due_at"`
	Detail         string   `json:"detail"         keel:"detail,nullzero"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`
}
