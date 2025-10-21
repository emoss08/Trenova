package seedhelpers

import (
	"context"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/uptrace/bun"
)

type SeedContext struct {
	DB  *bun.DB
	ctx context.Context

	defaultOrg *tenant.Organization
	defaultBU  *tenant.BusinessUnit
	states     map[string]*usstate.UsState // keyed by abbreviation
	roles      map[string]*permission.Role // keyed by name
	cacheMutex sync.RWMutex
}

func NewSeedContext(ctx context.Context, db *bun.DB) *SeedContext {
	return &SeedContext{
		DB:     db,
		ctx:    ctx,
		states: make(map[string]*usstate.UsState),
		roles:  make(map[string]*permission.Role),
	}
}

func (sc *SeedContext) GetDefaultOrganization() (*tenant.Organization, error) {
	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()

	if sc.defaultOrg != nil {
		return sc.defaultOrg, nil
	}

	var org tenant.Organization
	err := sc.DB.NewSelect().
		Model(&org).
		Where("scac_code = ?", "DFLT").
		Scan(sc.ctx)
	if err != nil {
		return nil, fmt.Errorf("get default organization: %w", err)
	}

	sc.defaultOrg = &org
	return &org, nil
}

func (sc *SeedContext) GetDefaultBusinessUnit() (*tenant.BusinessUnit, error) {
	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()

	if sc.defaultBU != nil {
		return sc.defaultBU, nil
	}

	var bu tenant.BusinessUnit
	err := sc.DB.NewSelect().
		Model(&bu).
		Where("code = ?", "DEFAULT").
		Scan(sc.ctx)
	if err != nil {
		return nil, fmt.Errorf("get default business unit: %w", err)
	}

	sc.defaultBU = &bu
	return &bu, nil
}

func (sc *SeedContext) GetState(abbreviation string) (*usstate.UsState, error) {
	sc.cacheMutex.RLock()
	if state, exists := sc.states[abbreviation]; exists {
		sc.cacheMutex.RUnlock()
		return state, nil
	}
	sc.cacheMutex.RUnlock()

	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()

	if state, exists := sc.states[abbreviation]; exists {
		return state, nil
	}

	var state usstate.UsState
	err := sc.DB.NewSelect().
		Model(&state).
		Where("abbreviation = ?", abbreviation).
		Scan(sc.ctx)
	if err != nil {
		return nil, fmt.Errorf("get state %s: %w", abbreviation, err)
	}

	sc.states[abbreviation] = &state
	return &state, nil
}

func (sc *SeedContext) GetRole(name string) (*permission.Role, error) {
	sc.cacheMutex.RLock()
	if role, exists := sc.roles[name]; exists {
		sc.cacheMutex.RUnlock()
		return role, nil
	}
	sc.cacheMutex.RUnlock()

	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()

	if role, exists := sc.roles[name]; exists {
		return role, nil
	}

	var role permission.Role
	err := sc.DB.NewSelect().
		Model(&role).
		Where("name = ?", name).
		Scan(sc.ctx)
	if err != nil {
		return nil, fmt.Errorf("get role %s: %w", name, err)
	}

	sc.roles[name] = &role
	return &role, nil
}

func (sc *SeedContext) GetAdminUser() (*tenant.User, error) {
	var user tenant.User
	err := sc.DB.NewSelect().
		Model(&user).
		Where("email_address = ?", "admin@trenova.app").
		Scan(sc.ctx)
	if err != nil {
		return nil, fmt.Errorf("get admin user: %w", err)
	}
	return &user, nil
}

func (sc *SeedContext) Context() context.Context {
	return sc.ctx
}
