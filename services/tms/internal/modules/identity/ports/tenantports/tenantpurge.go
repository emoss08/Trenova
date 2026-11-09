package tenantports

import (
	"github.com/emoss08/trenova/shared/pulid"
)

type TenantMember struct {
	UserID           pulid.ID `json:"userId"`
	Name             string   `json:"name"`
	EmailAddress     string   `json:"emailAddress"`
	Username         string   `json:"username"`
	Locale           string   `json:"locale"`
	OtherMemberships int      `json:"otherMemberships"`
}

type PurgeTableOutcome struct {
	Table   string `json:"table"`
	Deleted int64  `json:"deleted"`
}
