package tenant

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	SystemActorName        = "System Account"
	SystemActorTimezone    = "UTC"
	systemActorPrefix      = "system-"
	systemActorEmailDomain = "system.trenova.invalid"
	systemActorSuffixRunes = MaxUsernameLength - len(systemActorPrefix)
)

type SystemActor struct {
	keel.Table `keel:"system_actors"`

	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	UserID         pulid.ID `json:"userId"         keel:"user_id"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
}

func NewSystemActorUser(businessUnitID, organizationID pulid.ID) *User {
	username := SystemActorUsername(businessUnitID)
	return &User{
		BusinessUnitID:        businessUnitID,
		CurrentOrganizationID: organizationID,
		Name:                  SystemActorName,
		Username:              username,
		EmailAddress:          username + "@" + systemActorEmailDomain,
		Timezone:              SystemActorTimezone,
	}
}

func SystemActorUsername(businessUnitID pulid.ID) string {
	id := strings.ToLower(businessUnitID.String())
	if len(id) > systemActorSuffixRunes {
		id = id[len(id)-systemActorSuffixRunes:]
	}
	return systemActorPrefix + id
}
