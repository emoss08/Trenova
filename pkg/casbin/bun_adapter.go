package casbin

import (
	"context"
	"fmt"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/uptrace/bun"
)

// BunAdapter is a Casbin adapter for the Bun ORM.
type BunAdapter struct {
	db *bun.DB
}

// Rule represents a Casbin policy rule in the database.
type Rule struct {
	bun.BaseModel `bun:"table:casbin_rule,alias:cr"`

	ID    int64  `bun:",pk,autoincrement"`
	Ptype string `bun:",notnull"`
	V0    string `bun:",notnull"`
	V1    string `bun:",notnull"`
	V2    string `bun:",notnull"`
	V3    string `bun:",notnull"`
	V4    string `bun:",notnull"`
	V5    string `bun:",notnull"`
}

// String converts the Rule to a string representation.
func (r *Rule) String() string {
	return fmt.Sprintf("%s, %s, %s, %s, %s, %s, %s", r.Ptype, r.V0, r.V1, r.V2, r.V3, r.V4, r.V5)
}

// NewBunAdapter creates a new BunAdapter.
func NewBunAdapter(db *bun.DB) (*BunAdapter, error) {
	adapter := &BunAdapter{db: db}

	// Ensure the database schema is up to date
	err := db.ResetModel(context.Background(), (*Rule)(nil))
	if err != nil {
		return nil, fmt.Errorf("failed to reset database model: %w", err)
	}

	return adapter, nil
}

// LoadPolicy loads policy rules from the database.
func (a *BunAdapter) LoadPolicy(model model.Model) error {
	ctx := context.Background()
	var rules []Rule

	err := a.db.NewSelect().Model(&rules).Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to load policy rules: %w", err)
	}

	for _, rule := range rules {
		if err = persist.LoadPolicyLine(rule.String(), model); err != nil {
			return fmt.Errorf("failed to load policy line: %w", err)
		}
	}

	return nil
}

// SavePolicy saves policy rules to the database.
func (a *BunAdapter) SavePolicy(model model.Model) error {
	ctx := context.Background()
	var rules []Rule

	for ptype, ast := range model["p"] {
		for _, rule := range ast.Policy {
			rules = append(rules, a.savePolicyLine(ptype, rule))
		}
	}

	for ptype, ast := range model["g"] {
		for _, rule := range ast.Policy {
			rules = append(rules, a.savePolicyLine(ptype, rule))
		}
	}

	_, err := a.db.NewInsert().Model(&rules).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save policy rules: %w", err)
	}

	return nil
}

// AddPolicy adds a policy rule to the database.
func (a *BunAdapter) AddPolicy(_ string, ptype string, rule []string) error {
	ctx := context.Background()
	_, err := a.db.NewInsert().Model(a.savePolicyLine(ptype, rule)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add policy rule: %w", err)
	}
	return nil
}

// RemovePolicy removes a policy rule from the database.
func (a *BunAdapter) RemovePolicy(_ string, ptype string, rule []string) error {
	ctx := context.Background()
	_, err := a.db.NewDelete().Model((*Rule)(nil)).
		Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ? AND v3 = ? AND v4 = ? AND v5 = ?",
			ptype, rule[0], rule[1], rule[2], rule[3], rule[4], rule[5]).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove policy rule: %w", err)
	}
	return nil
}

// RemoveFilteredPolicy removes policy rules that match the filter from the database.
func (a *BunAdapter) RemoveFilteredPolicy(_ string, ptype string, fieldIndex int, fieldValues ...string) error {
	ctx := context.Background()
	query := a.db.NewDelete().Model((*Rule)(nil)).Where("ptype = ?", ptype)

	for i, v := range fieldValues {
		if v != "" {
			query = query.Where(fmt.Sprintf("v%d = ?", fieldIndex+i), v)
		}
	}

	_, err := query.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove filtered policy rules: %w", err)
	}
	return nil
}

// savePolicyLine converts a policy rule to a Rule struct.
func (a *BunAdapter) savePolicyLine(ptype string, rule []string) Rule {
	line := Rule{
		Ptype: ptype,
	}

	for i, v := range rule {
		if i >= 6 {
			break
		}
		switch i {
		case 0:
			line.V0 = v
		case 1:
			line.V1 = v
		case 2:
			line.V2 = v
		case 3:
			line.V3 = v
		case 4:
			line.V4 = v
		case 5:
			line.V5 = v
		}
	}

	return line
}
