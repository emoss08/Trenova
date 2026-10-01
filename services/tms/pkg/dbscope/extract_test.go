package dbscope

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

type tenantInfoLike struct {
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type requestWithTenantInfo struct {
	ShipmentID pulid.ID
	TenantInfo tenantInfoLike
}

type workItem struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Cursor         string
}

type nested struct {
	Request *requestWithTenantInfo
}

type twoTenants struct {
	Left  workItem
	Right workItem
}

type explicit struct {
	tenant Tenant
}

func (e explicit) DBTenant() Tenant { return e.tenant }

func TestTenantOf(t *testing.T) {
	t.Parallel()

	org, bu, user := pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")
	full := Tenant{OrganizationID: org, BusinessUnitID: bu, UserID: user}
	noUser := Tenant{OrganizationID: org, BusinessUnitID: bu}

	cases := []struct {
		name   string
		values []any
		want   Tenant
		ok     bool
	}{
		{name: "tenant info value", values: []any{tenantInfoLike{OrgID: org, BuID: bu, UserID: user}}, want: full, ok: true},
		{name: "nested tenant info", values: []any{&requestWithTenantInfo{TenantInfo: tenantInfoLike{OrgID: org, BuID: bu}}}, want: noUser, ok: true},
		{name: "work item", values: []any{workItem{OrganizationID: org, BusinessUnitID: bu}}, want: noUser, ok: true},
		{name: "deeply nested pointer", values: []any{nested{Request: &requestWithTenantInfo{TenantInfo: tenantInfoLike{OrgID: org, BuID: bu, UserID: user}}}}, want: full, ok: true},
		{name: "explicit interface", values: []any{explicit{tenant: full}}, want: full, ok: true},
		{name: "agreeing args fill in the user", values: []any{workItem{OrganizationID: org, BusinessUnitID: bu}, tenantInfoLike{OrgID: org, BuID: bu, UserID: user}}, want: full, ok: true},
		{name: "disagreeing args", values: []any{workItem{OrganizationID: org, BusinessUnitID: bu}, workItem{OrganizationID: pulid.MustNew("org_"), BusinessUnitID: bu}}},
		{name: "disagreeing fields", values: []any{twoTenants{Left: workItem{OrganizationID: org, BusinessUnitID: bu}, Right: workItem{OrganizationID: pulid.MustNew("org_"), BusinessUnitID: bu}}}},
		{name: "org without bu", values: []any{workItem{OrganizationID: org}}},
		{name: "no tenant", values: []any{"x", 3, nil, (*workItem)(nil)}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := TenantOf(tc.values...)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}
