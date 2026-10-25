package journalsource

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type SourceJournalLink struct {
	keel.Table `keel:"source_journal_links"`

	ID              pulid.ID `json:"id"              keel:"id,pk,autoid:sjl_"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID `json:"organizationId"  keel:"organization_id,pk,tenant"`
	JournalSourceID pulid.ID `json:"journalSourceId" keel:"journal_source_id"`
	JournalBatchID  pulid.ID `json:"journalBatchId"  keel:"journal_batch_id"`
	JournalEntryID  pulid.ID `json:"journalEntryId"  keel:"journal_entry_id"`
	LinkRole        string   `json:"linkRole"        keel:"link_role"`
	CreatedAt       int64    `json:"createdAt"       keel:"created_at,autocreate"`
}
