package tenant

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// People in the organization: everyone who is a member of it, by name, job title and account
// status. A person who has left the organization is not listed, and work they did shows without
// a name.
type Person struct {
	keel.Table `keel:"organization_people,report=Desk,label=Person,plural=People"`

	ID             pulid.ID `json:"id"             keel:"id,pk"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	// The person's full name
	Name string `json:"name"     keel:"name"`
	// The job title on the person's profile
	JobTitle string `json:"jobTitle" keel:"job_title,nullzero"`
	// Whether the person's account is active
	Status domaintypes.Status `json:"status"   keel:"status"`
	// When the person joined the organization
	JoinedAt int64 `json:"joinedAt" keel:"joined_at,label=Joined"`
}
