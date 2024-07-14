// Credit https://github.com/JunNishimura/casbin-bun-adapter

package casbin

import "github.com/uptrace/bun"

// CasbinPolicy represents a Casbin policy CasbinPolicy in the database.
type CasbinPolicy struct {
	bun.BaseModel `bun:"casbin_policies,alias:cp"`
	ID            int64  `bun:"id,pk,autoincrement"`
	PType         string `bun:"ptype,type:varchar(100),notnull"`
	V0            string `bun:"v0,type:varchar(100)"`
	V1            string `bun:"v1,type:varchar(100)"`
	V2            string `bun:"v2,type:varchar(100)"`
	V3            string `bun:"v3,type:varchar(100)"`
	V4            string `bun:"v4,type:varchar(100)"`
	V5            string `bun:"v5,type:varchar(100)"`
}

func (c CasbinPolicy) toSlice() []string {
	policies := make([]string, 0)
	if c.PType != "" {
		policies = append(policies, c.PType)
	}
	if c.V0 != "" {
		policies = append(policies, c.V0)
	}
	if c.V1 != "" {
		policies = append(policies, c.V1)
	}
	if c.V2 != "" {
		policies = append(policies, c.V2)
	}
	if c.V3 != "" {
		policies = append(policies, c.V3)
	}
	if c.V4 != "" {
		policies = append(policies, c.V4)
	}
	if c.V5 != "" {
		policies = append(policies, c.V5)
	}
	return policies
}

func (c CasbinPolicy) FilterValues() []string {
	values := make([]string, 0)
	if c.V0 != "" {
		values = append(values, c.V0)
	}
	if c.V1 != "" {
		values = append(values, c.V1)
	}
	if c.V2 != "" {
		values = append(values, c.V2)
	}
	if c.V3 != "" {
		values = append(values, c.V3)
	}
	if c.V4 != "" {
		values = append(values, c.V4)
	}
	if c.V5 != "" {
		values = append(values, c.V5)
	}

	return values
}

func (c CasbinPolicy) filterValuesWithKey() map[string]string {
	values := make(map[string]string)
	if c.V0 != "" {
		values["v0"] = c.V0
	}
	if c.V1 != "" {
		values["v1"] = c.V1
	}
	if c.V2 != "" {
		values["v2"] = c.V2
	}
	if c.V3 != "" {
		values["v3"] = c.V3
	}
	if c.V4 != "" {
		values["v4"] = c.V4
	}
	if c.V5 != "" {
		values["v5"] = c.V5
	}

	return values
}
