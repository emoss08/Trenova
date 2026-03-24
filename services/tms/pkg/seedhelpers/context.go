package seedhelpers

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

var (
	ErrKeyNotFound    = errors.New("key not found in shared state")
	ErrInvalidType    = errors.New("value is not of expected type")
	ErrEmptyKey       = errors.New("key cannot be empty")
	ErrNilValue       = errors.New("value cannot be nil")
	ErrEntityNotFound = errors.New("entity not found")
)

type SeedContext struct {
	DB bun.IDB

	defaultBU  *tenant.BusinessUnit
	defaultOrg *tenant.Organization
	states     map[string]*usstate.UsState
	cacheMutex sync.RWMutex

	sharedState map[string]any
	stateMutex  sync.RWMutex

	tracker           *EntityTracker
	persistentTracker *PersistentEntityTracker
	logger            SeedLogger
	cfg               *config.Config
}

func NewSeedContext(db bun.IDB, logger SeedLogger, cfg *config.Config) *SeedContext {
	if logger == nil {
		logger = NewNoOpLogger()
	}

	var persistentTracker *PersistentEntityTracker
	if bunDB, ok := db.(*bun.DB); ok {
		persistentTracker = NewPersistentEntityTracker(bunDB)
	}

	return &SeedContext{
		DB:                db,
		states:            make(map[string]*usstate.UsState),
		sharedState:       make(map[string]any),
		tracker:           NewEntityTracker(),
		persistentTracker: persistentTracker,
		logger:            logger,
		cfg:               cfg,
	}
}

func (sc *SeedContext) Config() *config.Config {
	return sc.cfg
}

func (sc *SeedContext) GetDefaultBusinessUnit(ctx context.Context) (*tenant.BusinessUnit, error) {
	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()

	if sc.defaultBU != nil {
		sc.logger.CacheHit("default_business_unit")
		return sc.defaultBU, nil
	}

	sc.logger.CacheMiss("default_business_unit")

	var bu tenant.BusinessUnit
	err := sc.DB.NewSelect().
		Model(&bu).
		Where("code = ?", "DEFAULT").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get default business unit: %w", err)
	}

	sc.logger.EntityQueried("business_units", bu.ID)
	sc.defaultBU = &bu
	return &bu, nil
}

func (sc *SeedContext) GetState(
	ctx context.Context,
	abbreviation string,
) (*usstate.UsState, error) {
	sc.cacheMutex.RLock()
	if state, exists := sc.states[abbreviation]; exists {
		sc.cacheMutex.RUnlock()
		sc.logger.CacheHit(fmt.Sprintf("state_%s", abbreviation))
		return state, nil
	}
	sc.cacheMutex.RUnlock()

	sc.logger.CacheMiss(fmt.Sprintf("state_%s", abbreviation))

	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()

	if state, exists := sc.states[abbreviation]; exists {
		return state, nil
	}

	var state usstate.UsState
	err := sc.DB.NewSelect().
		Model(&state).
		Where("abbreviation = ?", abbreviation).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get state %s: %w", abbreviation, err)
	}

	sc.logger.EntityQueried("us_states", state.ID)
	sc.states[abbreviation] = &state
	return &state, nil
}

func (sc *SeedContext) GetDefaultOrganization(ctx context.Context) (*tenant.Organization, error) {
	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()

	if sc.defaultOrg != nil {
		sc.logger.CacheHit("default_organization")
		return sc.defaultOrg, nil
	}

	sc.logger.CacheMiss("default_organization")

	var org tenant.Organization
	err := sc.DB.NewSelect().
		Model(&org).
		Where("scac_code IN (?)", bun.In([]string{"DFLT", "TRNV"})).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get default organization: %w", err)
	}

	sc.logger.EntityQueried("organizations", org.ID)
	sc.defaultOrg = &org
	return &org, nil
}

func (sc *SeedContext) Set(key string, value any) error {
	if key == "" {
		return ErrEmptyKey
	}
	if value == nil {
		return ErrNilValue
	}

	sc.stateMutex.Lock()
	defer sc.stateMutex.Unlock()
	sc.sharedState[key] = value
	return nil
}

func (sc *SeedContext) Get(key string) (any, bool) {
	if key == "" {
		return nil, false
	}

	sc.stateMutex.RLock()
	defer sc.stateMutex.RUnlock()
	val, exists := sc.sharedState[key]
	return val, exists
}

func (sc *SeedContext) GetOrganization(key string) (*tenant.Organization, error) {
	val, exists := sc.Get(key)
	if !exists {
		return nil, fmt.Errorf("organization not found in shared state: %s", key)
	}

	org, ok := val.(*tenant.Organization)
	if !ok {
		return nil, fmt.Errorf("value for key %s is not an Organization", key)
	}

	return org, nil
}

func (sc *SeedContext) GetUser(key string) (*tenant.User, error) {
	val, exists := sc.Get(key)
	if !exists {
		return nil, fmt.Errorf("user not found in shared state: %s", key)
	}

	user, ok := val.(*tenant.User)
	if !ok {
		return nil, fmt.Errorf("value for key %s is not a User", key)
	}

	return user, nil
}

func (sc *SeedContext) GetBusinessUnit(key string) (*tenant.BusinessUnit, error) {
	val, exists := sc.Get(key)
	if !exists {
		return nil, fmt.Errorf("business unit not found in shared state: %s", key)
	}

	bu, ok := val.(*tenant.BusinessUnit)
	if !ok {
		return nil, fmt.Errorf("value for key %s is not a BusinessUnit", key)
	}

	return bu, nil
}

func (sc *SeedContext) GetOrganizationByScac(
	ctx context.Context,
	scacCode string,
) (*tenant.Organization, error) {
	var org tenant.Organization
	err := sc.DB.NewSelect().
		Model(&org).
		Where("scac_code = ?", scacCode).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get organization by scac %s: %w", scacCode, err)
	}

	sc.logger.EntityQueried("organizations", org.ID)
	return &org, nil
}

func (sc *SeedContext) GetUserByUsername(
	ctx context.Context,
	username string,
) (*tenant.User, error) {
	var user tenant.User
	err := sc.DB.NewSelect().
		Model(&user).
		Where("username = ?", username).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user by username %s: %w", username, err)
	}

	sc.logger.EntityQueried("users", user.ID)
	return &user, nil
}

func (sc *SeedContext) TrackCreated(
	ctx context.Context,
	table string,
	id pulid.ID,
	seedName string,
) error {
	if err := sc.tracker.Track(table, id, seedName); err != nil {
		return err
	}

	if sc.persistentTracker != nil {
		if err := sc.persistentTracker.Track(ctx, table, id, seedName); err != nil {
			return fmt.Errorf("track created entity in database: %w", err)
		}
	}

	return nil
}

func (sc *SeedContext) GetCreatedEntities(
	ctx context.Context,
	seedName string,
) ([]TrackedEntity, error) {
	if sc.persistentTracker != nil {
		return sc.persistentTracker.GetBySeed(ctx, seedName)
	}

	return sc.tracker.GetBySeed(seedName), nil
}

func (sc *SeedContext) GetAllCreatedEntities(ctx context.Context) ([]TrackedEntity, error) {
	if sc.persistentTracker != nil {
		return sc.persistentTracker.GetAll(ctx)
	}

	return sc.tracker.GetAll(), nil
}

func (sc *SeedContext) DeleteTrackedEntities(ctx context.Context, seedName string) error {
	if sc.persistentTracker != nil {
		return sc.persistentTracker.DeleteBySeed(ctx, seedName)
	}

	sc.tracker.Clear()
	return nil
}

func (sc *SeedContext) Logger() SeedLogger {
	return sc.logger
}
