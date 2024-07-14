package casbin

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/casbin/casbin/v2/model"
	"github.com/uptrace/bun"
)

// BunAdapter is a Casbin adapter for the Bun ORM.
type BunAdapter struct {
	db *bun.DB
}

// NewBunAdapter creates a new BunAdapter.
func NewBunAdapter(db *bun.DB) (*BunAdapter, error) {
	return &BunAdapter{db: db}, nil
}

func newCasbinPolicy(ptype string, rule []string) CasbinPolicy {
	c := CasbinPolicy{
		PType: ptype,
	}

	for i, v := range rule {
		switch i {
		case 0:
			c.V0 = v
		case 1:
			c.V1 = v
		case 2:
			c.V2 = v
		case 3:
			c.V3 = v
		case 4:
			c.V4 = v
		case 5:
			c.V5 = v
		}
	}

	return c
}

func (a *BunAdapter) LoadPolicy(model model.Model) error {
	var policies []CasbinPolicy
	err := a.db.NewSelect().
		Model(&policies).
		Scan(context.Background())
	if err != nil {
		return err
	}

	for _, policy := range policies {
		if err := loadPolicyRecord(policy, model); err != nil {
			return err
		}
	}

	return nil
}

func loadPolicyRecord(policy CasbinPolicy, model model.Model) error {
	pType := policy.PType
	sec := pType[:1]
	ok, err := model.HasPolicyEx(sec, pType, policy.FilterValues())
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	model.AddPolicy(sec, pType, policy.FilterValues())
	return nil
}

func (a *BunAdapter) SavePolicy(model model.Model) error {
	policies := make([]CasbinPolicy, 0)

	for ptype, ast := range model["p"] {
		for _, rule := range ast.Policy {
			policies = append(policies, newCasbinPolicy(ptype, rule))
		}
	}

	for ptype, ast := range model["g"] {
		for _, rule := range ast.Policy {
			policies = append(policies, newCasbinPolicy(ptype, rule))
		}
	}

	return a.savePolicyRecords(policies)
}

func (a *BunAdapter) savePolicyRecords(policies []CasbinPolicy) error {
	// Delete existing policies
	if err := a.refreshTable(); err != nil {
		return err
	}

	if _, err := a.db.NewInsert().
		Model(&policies).
		Exec(context.Background()); err != nil {
		return err
	}

	return nil
}

func (a *BunAdapter) refreshTable() error {
	if _, err := a.db.NewTruncateTable().
		Model((*CasbinPolicy)(nil)).
		Exec(context.Background()); err != nil {
		return err
	}

	return nil
}

func (a *BunAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	newPolicy := newCasbinPolicy(ptype, rule)
	if _, err := a.db.NewInsert().Model(&newPolicy).Exec(context.Background()); err != nil {
		return err
	}

	return nil
}

func (a *BunAdapter) AddPolicies(sec string, ptype string, rules [][]string) error {
	policies := make([]CasbinPolicy, 0)
	for _, rule := range rules {
		policies = append(policies, newCasbinPolicy(ptype, rule))
	}

	if _, err := a.db.NewInsert().Model(&policies).Exec(context.Background()); err != nil {
		return err
	}

	return nil
}

// RemovePolicy removes a policy rule from the storage.
// This is part of the Auto-Save feature.
func (a *BunAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	exisingPolicy := newCasbinPolicy(ptype, rule)
	if err := a.deleteRecord(exisingPolicy); err != nil {
		return err
	}
	return nil
}

// RemovePolicies removes policy rules from the storage.
// This is part of the Auto-Save feature.
func (a *BunAdapter) RemovePolicies(sec string, ptype string, rules [][]string) error {
	return a.db.RunInTx(context.Background(), &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		for _, rule := range rules {
			exisingPolicy := newCasbinPolicy(ptype, rule)
			if err := a.deleteRecordInTx(tx, exisingPolicy); err != nil {
				return err
			}
		}
		return nil
	})
}

func (a *BunAdapter) deleteRecord(existingPolicy CasbinPolicy) error {
	query := a.db.NewDelete().
		Model((*CasbinPolicy)(nil)).
		Where("ptype = ?", existingPolicy.PType)

	values := existingPolicy.filterValuesWithKey()

	return a.delete(query, values)
}

func (a *BunAdapter) deleteRecordInTx(tx bun.Tx, existingPolicy CasbinPolicy) error {
	query := tx.NewDelete().
		Model((*CasbinPolicy)(nil)).
		Where("ptype = ?", existingPolicy.PType)

	values := existingPolicy.filterValuesWithKey()

	return a.delete(query, values)
}

func (a *BunAdapter) delete(query *bun.DeleteQuery, values map[string]string) error {
	for key, value := range values {
		query = query.Where(fmt.Sprintf("%s = ?", key), value)
	}

	if _, err := query.Exec(context.Background()); err != nil {
		return err
	}

	return nil
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
// This is part of the Auto-Save feature.
// This API is explained in the link below:
// https://casbin.org/docs/management-api/#removefilteredpolicy
func (a *BunAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	if err := a.deleteFilteredPolicy(ptype, fieldIndex, fieldValues...); err != nil {
		return err
	}
	return nil
}

func (a *BunAdapter) deleteFilteredPolicy(ptype string, fieldIndex int, fieldValues ...string) error {
	query := a.db.NewDelete().
		Model((*CasbinPolicy)(nil)).
		Where("ptype = ?", ptype)

	// Note that empty string in fieldValues could be any word.
	if fieldIndex <= 0 && 0 < fieldIndex+len(fieldValues) {
		value := fieldValues[0-fieldIndex]
		if value == "" {
			query = query.Where("v0 LIKE '%'")
		} else {
			query = query.Where("v0 = ?", value)
		}
	}
	if fieldIndex <= 1 && 1 < fieldIndex+len(fieldValues) {
		value := fieldValues[1-fieldIndex]
		if value == "" {
			query = query.Where("v1 LIKE '%'")
		} else {
			query = query.Where("v1 = ?", value)
		}
	}
	if fieldIndex <= 2 && 2 < fieldIndex+len(fieldValues) {
		value := fieldValues[2-fieldIndex]
		if value == "" {
			query = query.Where("v2 LIKE '%'")
		} else {
			query = query.Where("v2 = ?", value)
		}
	}
	if fieldIndex <= 3 && 3 < fieldIndex+len(fieldValues) {
		value := fieldValues[3-fieldIndex]
		if value == "" {
			query = query.Where("v3 LIKE '%'")
		} else {
			query = query.Where("v3 = ?", value)
		}
	}
	if fieldIndex <= 4 && 4 < fieldIndex+len(fieldValues) {
		value := fieldValues[4-fieldIndex]
		if value == "" {
			query = query.Where("v4 LIKE '%'")
		} else {
			query = query.Where("v4 = ?", value)
		}
	}
	if fieldIndex <= 5 && 5 < fieldIndex+len(fieldValues) {
		value := fieldValues[5-fieldIndex]
		if value == "" {
			query = query.Where("v5 LIKE '%'")
		} else {
			query = query.Where("v5 = ?", value)
		}
	}

	if _, err := query.Exec(context.Background()); err != nil {
		return err
	}

	return nil
}

// UpdatePolicy updates a policy rule from storage.
// This is part of the Auto-Save feature.
func (a *BunAdapter) UpdatePolicy(sec string, ptype string, oldRule, newRule []string) error {
	oldPolicy := newCasbinPolicy(ptype, oldRule)
	newPolicy := newCasbinPolicy(ptype, newRule)
	return a.updateRecord(oldPolicy, newPolicy)
}

func (a *BunAdapter) updateRecord(oldPolicy, newPolicy CasbinPolicy) error {
	query := a.db.NewUpdate().
		Model(&newPolicy).
		Where("ptype = ?", oldPolicy.PType)

	values := oldPolicy.filterValuesWithKey()

	return a.update(query, values)
}

func (a *BunAdapter) updateRecordInTx(tx bun.Tx, oldPolicy, newPolicy CasbinPolicy) error {
	query := tx.NewUpdate().
		Model(&newPolicy).
		Where("ptype = ?", oldPolicy.PType)

	values := oldPolicy.filterValuesWithKey()

	return a.update(query, values)
}

func (a *BunAdapter) update(query *bun.UpdateQuery, values map[string]string) error {
	for key, value := range values {
		query = query.Where(fmt.Sprintf("%s = ?", key), value)
	}

	if _, err := query.Exec(context.Background()); err != nil {
		return err
	}

	return nil
}

// UpdatePolicies updates some policy rules to storage, like db, redis.
func (a *BunAdapter) UpdatePolicies(sec string, ptype string, oldRules, newRules [][]string) error {
	oldPolicies := make([]CasbinPolicy, 0, len(oldRules))
	newPolicies := make([]CasbinPolicy, 0, len(newRules))
	for _, rule := range oldRules {
		oldPolicies = append(oldPolicies, newCasbinPolicy(ptype, rule))
	}
	for _, rule := range newRules {
		newPolicies = append(newPolicies, newCasbinPolicy(ptype, rule))
	}

	return a.db.RunInTx(context.Background(), &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		for i := range oldPolicies {
			if err := a.updateRecordInTx(tx, oldPolicies[i], newPolicies[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateFilteredPolicies deletes old rules and adds new rules.
func (a *BunAdapter) UpdateFilteredPolicies(sec string, ptype string, newRules [][]string, fieldIndex int, fieldValues ...string) ([][]string, error) {
	newPolicies := make([]CasbinPolicy, 0, len(newRules))
	for _, rule := range newRules {
		newPolicies = append(newPolicies, newCasbinPolicy(ptype, rule))
	}

	tx, err := a.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return nil, err
	}

	oldPolicies := make([]CasbinPolicy, 0)
	selectQuery := tx.NewSelect().
		Model(&oldPolicies).
		Where("ptype = ?", ptype)
	deleteQuery := tx.NewDelete().
		Model((*CasbinPolicy)(nil)).
		Where("ptype = ?", ptype)

	// Note that empty string in fieldValues could be any word.
	if fieldIndex <= 0 && 0 < fieldIndex+len(fieldValues) {
		value := fieldValues[0-fieldIndex]
		if value == "" {
			selectQuery = selectQuery.Where("v0 LIKE '%'")
			deleteQuery = deleteQuery.Where("v0 LIKE '%'")
		} else {
			selectQuery = selectQuery.Where("v0 = ?", value)
			deleteQuery = deleteQuery.Where("v0 = ?", value)
		}
	}
	if fieldIndex <= 1 && 1 < fieldIndex+len(fieldValues) {
		value := fieldValues[1-fieldIndex]
		if value == "" {
			selectQuery = selectQuery.Where("v1 LIKE '%'")
			deleteQuery = deleteQuery.Where("v1 LIKE '%'")
		} else {
			selectQuery = selectQuery.Where("v1 = ?", value)
			deleteQuery = deleteQuery.Where("v1 = ?", value)
		}
	}
	if fieldIndex <= 2 && 2 < fieldIndex+len(fieldValues) {
		value := fieldValues[2-fieldIndex]
		if value == "" {
			selectQuery = selectQuery.Where("v2 LIKE '%'")
			deleteQuery = deleteQuery.Where("v2 LIKE '%'")
		} else {
			selectQuery = selectQuery.Where("v2 = ?", value)
			deleteQuery = deleteQuery.Where("v2 = ?", value)
		}
	}
	if fieldIndex <= 3 && 3 < fieldIndex+len(fieldValues) {
		value := fieldValues[3-fieldIndex]
		if value == "" {
			selectQuery = selectQuery.Where("v3 LIKE '%'")
			deleteQuery = deleteQuery.Where("v3 LIKE '%'")
		} else {
			selectQuery = selectQuery.Where("v3 = ?", value)
			deleteQuery = deleteQuery.Where("v3 = ?", value)
		}
	}
	if fieldIndex <= 4 && 4 < fieldIndex+len(fieldValues) {
		value := fieldValues[4-fieldIndex]
		if value == "" {
			selectQuery = selectQuery.Where("v4 LIKE '%'")
			deleteQuery = deleteQuery.Where("v4 LIKE '%'")
		} else {
			selectQuery = selectQuery.Where("v4 = ?", value)
			deleteQuery = deleteQuery.Where("v4 = ?", value)
		}
	}
	if fieldIndex <= 5 && 5 < fieldIndex+len(fieldValues) {
		value := fieldValues[5-fieldIndex]
		if value == "" {
			selectQuery = selectQuery.Where("v5 LIKE '%'")
			deleteQuery = deleteQuery.Where("v5 LIKE '%'")
		} else {
			selectQuery = selectQuery.Where("v5 = ?", value)
			deleteQuery = deleteQuery.Where("v5 = ?", value)
		}
	}

	// store old policies
	if err := selectQuery.Scan(context.Background()); err != nil {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, err
	}

	// delete old policies
	if _, err := deleteQuery.Exec(context.Background()); err != nil {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, err
	}

	// create new policies
	if _, err := tx.NewInsert().
		Model(&newPolicies).
		Exec(context.Background()); err != nil {
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		return nil, err
	}

	out := make([][]string, 0, len(oldPolicies))
	for _, policy := range oldPolicies {
		out = append(out, policy.toSlice())
	}

	return out, tx.Commit()
}
