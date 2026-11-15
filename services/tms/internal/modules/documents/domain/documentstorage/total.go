package documentstorage

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type Total struct {
	keel.Table `keel:"document_storage_totals"`

	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	Bytes          int64    `json:"bytes"          keel:"bytes"`
}
