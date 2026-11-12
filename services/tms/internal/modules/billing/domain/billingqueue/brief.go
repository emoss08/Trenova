package billingqueue

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type QueueBrief struct {
	keel.Table `keel:"billing_queue_briefs"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:bqb_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	Text           string   `json:"text"           keel:"text"`
	InputHash      string   `json:"-"              keel:"input_hash"`
	LanesHash      string   `json:"-"              keel:"lanes_hash"`
	GeneratedAt    int64    `json:"generatedAt"    keel:"generated_at"`
	Version        int64    `json:"version"        keel:"version,version"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`
}
