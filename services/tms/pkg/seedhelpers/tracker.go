package seedhelpers

import (
	"fmt"
	"sync"

	"github.com/emoss08/trenova/shared/pulid"
)

type TrackedEntity struct {
	Table    string
	ID       pulid.ID
	SeedName string
}

type EntityTracker struct {
	entities       []TrackedEntity
	entitiesBySeed map[string][]TrackedEntity
	mutex          sync.RWMutex
}

func NewEntityTracker() *EntityTracker {
	return &EntityTracker{
		entities:       make([]TrackedEntity, 0, 100),
		entitiesBySeed: make(map[string][]TrackedEntity),
	}
}

func (et *EntityTracker) Track(table string, id pulid.ID, seedName string) error {
	if table == "" {
		return fmt.Errorf("table: %w", ErrEmptyKey)
	}
	if id == "" {
		return fmt.Errorf("id: %w", ErrEmptyKey)
	}
	if seedName == "" {
		return fmt.Errorf("seed name: %w", ErrEmptyKey)
	}

	et.mutex.Lock()
	defer et.mutex.Unlock()

	entity := TrackedEntity{
		Table:    table,
		ID:       id,
		SeedName: seedName,
	}

	et.entities = append(et.entities, entity)
	et.entitiesBySeed[seedName] = append(et.entitiesBySeed[seedName], entity)

	return nil
}

func (et *EntityTracker) GetBySeed(seedName string) []TrackedEntity {
	et.mutex.RLock()
	defer et.mutex.RUnlock()

	entities, exists := et.entitiesBySeed[seedName]
	if !exists {
		return []TrackedEntity{}
	}

	result := make([]TrackedEntity, len(entities))
	copy(result, entities)
	return result
}

func (et *EntityTracker) GetAll() []TrackedEntity {
	et.mutex.RLock()
	defer et.mutex.RUnlock()

	result := make([]TrackedEntity, len(et.entities))
	copy(result, et.entities)
	return result
}

func (et *EntityTracker) Clear() {
	et.mutex.Lock()
	defer et.mutex.Unlock()

	et.entities = make([]TrackedEntity, 0, 100)
	et.entitiesBySeed = make(map[string][]TrackedEntity)
}

func (et *EntityTracker) Count() int {
	et.mutex.RLock()
	defer et.mutex.RUnlock()
	return len(et.entities)
}

func (et *EntityTracker) CountBySeed(seedName string) int {
	et.mutex.RLock()
	defer et.mutex.RUnlock()
	return len(et.entitiesBySeed[seedName])
}
