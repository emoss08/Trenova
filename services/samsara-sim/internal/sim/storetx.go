package sim

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

type storeTx struct {
	store *Store
}

func (s *Store) Transact(apply func(tx *storeTx) error) error {
	return s.mutate(func() error {
		backup := s.state.shallowCopy()
		counters := maps.Clone(s.counters)
		if err := apply(&storeTx{store: s}); err != nil {
			s.state = backup
			s.counters = counters
			return err
		}
		return nil
	})
}

func (f *Fixture) shallowCopy() Fixture {
	out := Fixture{AssetLocation: f.AssetLocation}
	for _, resource := range allResources {
		if resource == ResourceAssetLocation {
			continue
		}
		source, sourceOK := f.collection(resource)
		target, targetOK := out.collection(resource)
		if sourceOK && targetOK {
			*target = slices.Clone(*source)
		}
	}
	return out
}

func (s *Store) View(resource Resource, read func(records []Record)) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list, err := s.resourceSliceLocked(resource)
	if err != nil {
		return err
	}
	read(list)
	return nil
}

func (tx *storeTx) records(resource Resource) []Record {
	list, err := tx.store.resourceSliceLocked(resource)
	if err != nil {
		return nil
	}
	return list
}

func (tx *storeTx) replace(resource Resource, records []Record) error {
	return tx.store.setResourceSliceLocked(resource, records)
}

func (tx *storeTx) find(resource Resource, id string) (record Record, index int) {
	clean := strings.TrimSpace(id)
	if clean == "" {
		return nil, -1
	}
	for idx, candidate := range tx.records(resource) {
		if recordID(candidate) == clean {
			return candidate, idx
		}
	}
	return nil, -1
}

func (tx *storeTx) insert(resource Resource, payload Record, at time.Time) (Record, error) {
	list, err := tx.store.resourceSliceLocked(resource)
	if err != nil {
		return nil, err
	}
	fields := resourceTimestampFields(resource)
	record := cloneRecord(payload)
	stripServerManagedFields(record, fields)
	stamp := formatStoreTime(at)
	if fields.has(timestampCreated) {
		record[fieldCreatedAtTime] = stamp
	}
	if fields.has(timestampUpdated) {
		record[fieldUpdatedAtTime] = stamp
	}
	if generatedID := tx.store.nextIDLocked(resource, list); generatedID != "" {
		record["id"] = generatedID
	}
	if err = tx.store.setResourceSliceLocked(resource, append(list, record)); err != nil {
		return nil, err
	}
	return record, nil
}

func (tx *storeTx) update(
	resource Resource,
	id string,
	mutate func(record Record) error,
	at time.Time,
) (Record, error) {
	current, idx := tx.find(resource, id)
	if idx < 0 {
		return nil, ErrRecordNotFound
	}
	next := cloneRecord(current)
	if err := mutate(next); err != nil {
		return nil, err
	}
	next["id"] = recordID(current)
	if resourceTimestampFields(resource).has(timestampUpdated) {
		next[fieldUpdatedAtTime] = formatStoreTime(at)
	}
	if created, ok := current[fieldCreatedAtTime]; ok {
		next[fieldCreatedAtTime] = created
	}
	tx.records(resource)[idx] = next
	return next, nil
}

func (tx *storeTx) remove(resource Resource, id string) error {
	list := tx.records(resource)
	_, idx := tx.find(resource, id)
	if idx < 0 {
		return ErrRecordNotFound
	}
	updated := make([]Record, 0, len(list)-1)
	updated = append(updated, list[:idx]...)
	updated = append(updated, list[idx+1:]...)
	if err := tx.store.setResourceSliceLocked(resource, updated); err != nil {
		return fmt.Errorf("remove %s %s: %w", resource, id, err)
	}
	return nil
}

type collectionsSnapshot struct {
	Records         map[Resource][]Record
	Version         uint64
	LocationVersion uint64
	Waypoints       map[string][]routePoint
}

func (s *Store) Collections(
	resources []Resource,
	knownLocationVersion uint64,
	reuseWaypoints bool,
) collectionsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := collectionsSnapshot{
		Records:         make(map[Resource][]Record, len(resources)),
		Version:         s.version.Load(),
		LocationVersion: s.locationVersion.Load(),
	}
	for _, resource := range resources {
		if slot, ok := s.state.collection(resource); ok {
			out.Records[resource] = cloneRecords(*slot)
		}
	}
	if !reuseWaypoints || knownLocationVersion != out.LocationVersion {
		out.Waypoints = parseWaypoints(s.state.AssetLocation)
	}
	return out
}

func (s *Store) TransactID(apply func(tx *storeTx) (string, error)) (string, error) {
	var id string
	err := s.Transact(func(tx *storeTx) error {
		var applyErr error
		id, applyErr = apply(tx)
		return applyErr
	})
	return id, err
}
