package sim

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
)

type Summary struct {
	Resources                 map[string]int      `json:"resources"`
	ActiveEventsByType        map[string]int      `json:"activeEventsByType"`
	ViolationsActive          int                 `json:"violationsActive"`
	SpeedingActive            int                 `json:"speedingActive"`
	GeofenceEntriesDispatched int64               `json:"geofenceEntriesDispatched"`
	GeofenceExitsDispatched   int64               `json:"geofenceExitsDispatched"`
	Persistence               *PersistenceSummary `json:"persistence,omitempty"`
}

type PersistenceSummary struct {
	Path   string `json:"path"`
	Writes uint64 `json:"writes"`
}

type WebhookTarget struct {
	ID            string
	Name          string
	URL           string
	Secret        string
	EventTypes    []string
	CustomHeaders []webhookCustomHeader
	SimDelivery   map[string]any
}

type Store struct {
	mu              sync.RWMutex
	seed            Fixture
	state           Fixture
	counters        map[idSpace]int64
	revision        uint64
	version         atomic.Uint64
	locationVersion atomic.Uint64
	persister       *statePersister
}

type CreateOptions struct {
	At       time.Time
	Decorate func(record Record)
}

type timestampFields uint8

const (
	timestampCreated timestampFields = 1 << iota
	timestampUpdated
)

const (
	fieldCreatedAtTime = "createdAtTime"
	fieldUpdatedAtTime = "updatedAtTime"
)

func resourceTimestampFields(resource Resource) timestampFields {
	switch resource {
	case ResourceAddresses:
		return timestampCreated
	case ResourceAssets, ResourceDrivers, ResourceFormSubmissions, ResourceDocuments:
		return timestampCreated | timestampUpdated
	case ResourceDvirs:
		return timestampUpdated
	case ResourceAssetLocation,
		ResourceRoutes,
		ResourceFormTemplates,
		ResourceLiveShares,
		ResourceMessages,
		ResourceWebhooks,
		ResourceVehicleStats,
		ResourceHOSClocks,
		ResourceHOSLogs,
		ResourceDriverTachograph,
		ResourceVehicleTachograph,
		ResourceTags,
		ResourceDriverVehicleAssignments,
		ResourceDriverSignOuts,
		ResourceDriverWorkflows,
		ResourceContacts,
		ResourceDocumentTypes,
		ResourceDocumentPDFs,
		ResourceFormPDFExports,
		ResourceUsers,
		ResourceDvirResolutions:
		return 0
	default:
		return 0
	}
}

func (f timestampFields) has(field timestampFields) bool {
	return f&field != 0
}

func stripServerManagedFields(record Record, fields timestampFields) {
	delete(record, "id")
	if fields.has(timestampCreated) {
		delete(record, fieldCreatedAtTime)
	}
	if fields.has(timestampUpdated) {
		delete(record, fieldUpdatedAtTime)
	}
}

func formatStoreTime(at time.Time) string {
	if at.IsZero() {
		at = time.Now()
	}
	return at.UTC().Format(time.RFC3339)
}

func NewStoreFromFixtureFile(path string) (*Store, error) {
	fixturePath := strings.TrimSpace(path)
	if fixturePath == "" {
		return nil, ErrFixturePathRequired
	}

	bytes, err := os.ReadFile(fixturePath)
	if err != nil {
		return nil, fmt.Errorf("read fixture file: %w", err)
	}
	if len(bytes) == 0 {
		return nil, ErrFixturePayloadEmpty
	}

	fixture := Fixture{}
	if err = sonic.Unmarshal(bytes, &fixture); err != nil {
		return nil, fmt.Errorf("parse fixture file: %w", err)
	}
	fixture.normalize()

	return NewStore(&fixture), nil
}

func NewStore(seed *Fixture) *Store {
	if seed == nil {
		seed = &Fixture{}
	}

	normalized := seed.clone()
	normalized.normalize()

	store := &Store{
		seed:     normalized.clone(),
		state:    normalized.clone(),
		counters: map[idSpace]int64{},
	}
	store.rebuildCountersLocked()
	return store
}

func (s *Store) Reset() error {
	persister := s.currentPersister()
	if persister != nil {
		persister.fileMu.Lock()
		defer persister.fileMu.Unlock()
	}

	s.mu.Lock()
	seedLocations := s.seed.AssetLocation
	s.seed.AssetLocation = nil
	s.state = s.seed.clone()
	s.seed.AssetLocation = seedLocations
	s.state.AssetLocation = seedLocations
	s.state.normalize()
	s.locationVersion.Add(1)
	s.rebuildCountersLocked()
	s.revision++
	s.version.Add(1)
	revision := s.revision
	s.mu.Unlock()

	if persister == nil {
		return nil
	}
	if err := removeStateFile(persister.path); err != nil {
		return err
	}
	persister.persisted = revision
	return nil
}

func (s *Store) Summary() Summary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var persistence *PersistenceSummary
	if s.persister != nil {
		persistence = &PersistenceSummary{
			Path:   s.persister.path,
			Writes: s.persister.writes.Load(),
		}
	}
	resources := make(map[string]int, len(allResources))
	for _, resource := range allResources {
		if slot, ok := s.state.collection(resource); ok {
			resources[string(resource)] = len(*slot)
		}
	}
	return Summary{
		Persistence: persistence,
		Resources:   resources,
	}
}

func (s *Store) DataVersion() uint64 {
	return s.version.Load()
}

func (s *Store) List(resource Resource) ([]Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list, err := s.resourceSliceLocked(resource)
	if err != nil {
		return nil, err
	}
	return cloneRecords(list), nil
}

func (s *Store) Get(resource Resource, id string) (Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	recordIDValue := strings.TrimSpace(id)
	if recordIDValue == "" {
		return nil, ErrRecordIDRequired
	}

	list, err := s.resourceSliceLocked(resource)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		if recordID(item) == recordIDValue {
			return cloneRecord(item), nil
		}
	}
	return nil, ErrRecordNotFound
}

func (s *Store) Create(resource Resource, payload Record) (Record, error) {
	var created Record
	err := s.mutate(func() error {
		list, err := s.resourceSliceLocked(resource)
		if err != nil {
			return err
		}

		record := cloneRecord(payload)
		assignedID := strings.TrimSpace(recordID(record))
		if assignedID == "" {
			if generatedID := s.nextIDLocked(resource, list); generatedID != "" {
				record["id"] = generatedID
			}
		} else {
			for _, existing := range list {
				if recordID(existing) == assignedID {
					return ErrRecordConflict
				}
			}
			s.observeIDLocked(resource, assignedID)
		}
		if err = s.setResourceSliceLocked(resource, append(list, record)); err != nil {
			return err
		}
		created = cloneRecord(record)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Store) CreateFromAPI(
	resource Resource,
	payload Record,
	options CreateOptions,
) (Record, error) {
	fields := resourceTimestampFields(resource)
	record := cloneRecord(payload)
	stripServerManagedFields(record, fields)
	if fields != 0 {
		stamp := formatStoreTime(options.At)
		if fields.has(timestampCreated) {
			record[fieldCreatedAtTime] = stamp
		}
		if fields.has(timestampUpdated) {
			record[fieldUpdatedAtTime] = stamp
		}
	}

	var created Record
	err := s.mutate(func() error {
		list, err := s.resourceSliceLocked(resource)
		if err != nil {
			return err
		}
		if generatedID := s.nextIDLocked(resource, list); generatedID != "" {
			record["id"] = generatedID
		}
		if options.Decorate != nil {
			options.Decorate(record)
		}
		if err = s.setResourceSliceLocked(resource, append(list, record)); err != nil {
			return err
		}
		created = cloneRecord(record)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Store) Patch(resource Resource, id string, patch Record) (Record, error) {
	return s.patch(resource, id, patch, nil)
}

func (s *Store) PatchFromAPI(
	resource Resource,
	id string,
	patch Record,
	at time.Time,
) (Record, error) {
	fields := resourceTimestampFields(resource)
	sanitized := make(Record, len(patch))
	for key, value := range patch {
		sanitized[key] = value
	}
	stripServerManagedFields(sanitized, fields)
	return s.patch(resource, id, sanitized, func(record Record) {
		if fields.has(timestampUpdated) {
			record[fieldUpdatedAtTime] = formatStoreTime(at)
		}
	})
}

func (s *Store) patch(
	resource Resource,
	id string,
	patch Record,
	finalize func(record Record),
) (Record, error) {
	recordIDValue := strings.TrimSpace(id)
	if recordIDValue == "" {
		return nil, ErrRecordIDRequired
	}

	var updated Record
	err := s.mutate(func() error {
		list, err := s.resourceSliceLocked(resource)
		if err != nil {
			return err
		}
		for idx := range list {
			if recordID(list[idx]) != recordIDValue {
				continue
			}

			current := cloneRecord(list[idx])
			mergePatch(current, patch)
			current["id"] = recordIDValue
			if finalize != nil {
				finalize(current)
			}
			list[idx] = current
			updated = cloneRecord(current)
			return nil
		}
		return ErrRecordNotFound
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Store) Delete(resource Resource, id string) error {
	recordIDValue := strings.TrimSpace(id)
	if recordIDValue == "" {
		return ErrRecordIDRequired
	}

	return s.mutate(func() error {
		list, err := s.resourceSliceLocked(resource)
		if err != nil {
			return err
		}
		for idx := range list {
			if recordID(list[idx]) != recordIDValue {
				continue
			}
			updated := make([]Record, 0, len(list)-1)
			updated = append(updated, list[:idx]...)
			updated = append(updated, list[idx+1:]...)
			return s.setResourceSliceLocked(resource, updated)
		}
		return ErrRecordNotFound
	})
}

func (s *Store) AppendMessages(records []Record) error {
	if len(records) == 0 {
		return nil
	}
	return s.mutate(func() error {
		s.state.Messages = append(s.state.Messages, cloneRecords(records)...)
		return nil
	})
}

func (s *Store) Replace(resource Resource, records []Record) error {
	return s.mutate(func() error {
		if err := s.setResourceSliceLocked(resource, cloneRecords(records)); err != nil {
			return err
		}
		s.trackResourceIDsLocked(resource, records)
		return nil
	})
}

func (s *Store) ReplaceBaseline(resource Resource, records []Record) error {
	apply := func() error {
		baseline := cloneRecords(records)
		if err := s.setSeedSliceLocked(resource, baseline); err != nil {
			return err
		}
		current := baseline
		if !isTransientResource(resource) {
			current = cloneRecords(records)
		}
		if err := s.setResourceSliceLocked(resource, current); err != nil {
			return err
		}
		s.trackResourceIDsLocked(resource, records)
		return nil
	}
	if isTransientResource(resource) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if err := apply(); err != nil {
			return err
		}
		s.version.Add(1)
		return nil
	}
	return s.mutate(apply)
}

func (s *Store) mutate(apply func() error) error {
	persister, err := s.mutateLocked(apply)
	if err != nil {
		return err
	}
	if persister != nil {
		persister.markDirty()
	}
	return nil
}

func (s *Store) mutateLocked(apply func() error) (*statePersister, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := apply(); err != nil {
		return nil, err
	}
	s.revision++
	s.version.Add(1)
	return s.persister, nil
}

func (s *Store) WebhookTargets(eventType string) []WebhookTarget {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filter := strings.TrimSpace(eventType)
	targets := make([]WebhookTarget, 0, len(s.state.Webhooks))
	for _, record := range s.state.Webhooks {
		urlValue, ok := record["url"].(string)
		if !ok || strings.TrimSpace(urlValue) == "" {
			continue
		}

		target := WebhookTarget{
			ID:     recordID(record),
			Name:   stringValue(record, "name"),
			URL:    urlValue,
			Secret: stringValue(record, "secretKey"),
		}
		target.EventTypes = stringSlice(record["eventTypes"])
		target.CustomHeaders = webhookCustomHeaders(record)
		if rawDelivery, okDelivery := anyAsMap(record["simDelivery"]); okDelivery {
			target.SimDelivery = cloneMap(rawDelivery)
		}

		if webhookSubscribes(record, target.EventTypes, filter) {
			targets = append(targets, target)
		}
	}
	return targets
}

func (s *Store) resourceSliceLocked(resource Resource) ([]Record, error) {
	slot, ok := s.state.collection(resource)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedResource, resource)
	}
	return *slot, nil
}

func (s *Store) setResourceSliceLocked(resource Resource, records []Record) error {
	if resource == ResourceAssetLocation {
		s.locationVersion.Add(1)
	}
	return setFixtureSlice(&s.state, resource, records)
}

func (s *Store) setSeedSliceLocked(resource Resource, records []Record) error {
	return setFixtureSlice(&s.seed, resource, records)
}

func setFixtureSlice(fixture *Fixture, resource Resource, records []Record) error {
	slot, ok := fixture.collection(resource)
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnsupportedResource, resource)
	}
	*slot = records
	return nil
}

func (s *Store) rebuildCountersLocked() {
	s.counters = map[idSpace]int64{}
	for _, resource := range idSchemedResources {
		list, err := s.resourceSliceLocked(resource)
		if err != nil {
			continue
		}
		s.trackResourceIDsLocked(resource, list)
	}
}

func (s *Store) trackResourceIDsLocked(resource Resource, records []Record) {
	scheme := resourceIDScheme(resource)
	switch scheme.Kind {
	case idKindNumeric:
		current := max(s.counters[scheme.Space], scheme.Floor)
		for _, record := range records {
			if value, ok := numericID(recordID(record)); ok && value > current {
				current = value
			}
		}
		s.counters[scheme.Space] = current
	case idKindUUID, idKindToken:
		s.counters[scheme.Space] = max(s.counters[scheme.Space], int64(len(records)))
	case idKindNone:
	}
}

func (s *Store) observeIDLocked(resource Resource, id string) {
	scheme := resourceIDScheme(resource)
	if scheme.Kind != idKindNumeric {
		return
	}
	if value, ok := numericID(id); ok && value > s.counters[scheme.Space] {
		s.counters[scheme.Space] = value
	}
}

func (s *Store) nextIDLocked(resource Resource, existing []Record) string {
	scheme := resourceIDScheme(resource)
	switch scheme.Kind {
	case idKindNumeric:
		next := max(s.counters[scheme.Space], scheme.Floor) + scheme.Stride
		s.counters[scheme.Space] = next
		return strconv.FormatInt(next, 10)
	case idKindUUID, idKindToken:
		taken := make(map[string]struct{}, len(existing))
		for _, record := range existing {
			taken[recordID(record)] = struct{}{}
		}
		for {
			sequence := s.counters[scheme.Space] + 1
			s.counters[scheme.Space] = sequence
			candidate := generatedRecordID(scheme.Space, scheme.Kind, sequence)
			if _, collision := taken[candidate]; !collision {
				return candidate
			}
		}
	case idKindNone:
		return ""
	default:
		return ""
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stringSlice(raw any) []string {
	switch typed := raw.(type) {
	case []string:
		return append([]string{}, typed...)
	case []any:
		values := make([]string, 0, len(typed))
		for _, value := range typed {
			stringValueRaw, ok := value.(string)
			if ok && strings.TrimSpace(stringValueRaw) != "" {
				values = append(values, strings.TrimSpace(stringValueRaw))
			}
		}
		return values
	default:
		return []string{}
	}
}

func stringValue(record Record, key string) string {
	raw, ok := record[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
