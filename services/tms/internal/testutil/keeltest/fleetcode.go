//go:build integration

package keeltest

import "github.com/emoss08/keel"

type FleetCode struct {
	keel.Table `keel:"fleet_codes"`

	ID             string  `keel:"id,pk,autoid:fc_"`
	OrganizationID string  `keel:"organization_id,pk,tenant"`
	BusinessUnitID string  `keel:"business_unit_id,pk,tenant"`
	ManagerID      string  `keel:"manager_id"`
	Code           string  `keel:"code"`
	Status         string  `keel:"status"`
	Description    *string `keel:"description"`
	Version        int64   `keel:"version,version"`
	CreatedAt      int64   `keel:"created_at,autocreate"`
	UpdatedAt      int64   `keel:"updated_at,autoupdate"`
}
