package sim

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
)

type Summary struct {
	Resources          map[string]int `json:"resources"`
	ActiveEventsByType map[string]int `json:"activeEventsByType"`
	ViolationsActive   int            `json:"violationsActive"`
	SpeedingActive     int            `json:"speedingActive"`
}

type WebhookTarget struct {
	ID          string
	Name        string
	URL         string
	Secret      string
	EventTypes  []string
	SimDelivery map[string]any
}

type Store struct {
	mu       sync.RWMutex
	seed     Fixture
	state    Fixture
	counters map[Resource]int
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
		counters: map[Resource]int{},
	}
	store.rebuildCountersLocked()
	return store
}

func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = s.seed.clone()
	s.state.normalize()
	s.rebuildCountersLocked()
}

func (s *Store) Summary() Summary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return Summary{
		Resources: map[string]int{
			string(ResourceAddresses):         len(s.state.Addresses),
			string(ResourceAssets):            len(s.state.Assets),
			string(ResourceAssetLocation):     len(s.state.AssetLocation),
			string(ResourceDrivers):           len(s.state.Drivers),
			string(ResourceRoutes):            len(s.state.Routes),
			string(ResourceFormTemplates):     len(s.state.FormTemplates),
			string(ResourceFormSubmissions):   len(s.state.FormSubmissions),
			string(ResourceLiveShares):        len(s.state.LiveShares),
			string(ResourceMessages):          len(s.state.Messages),
			string(ResourceWebhooks):          len(s.state.Webhooks),
			string(ResourceVehicleStats):      len(s.state.VehicleStats),
			string(ResourceHOSClocks):         len(s.state.HOSClocks),
			string(ResourceHOSLogs):           len(s.state.HOSLogs),
			string(ResourceDriverTachograph):  len(s.state.DriverTachograph),
			string(ResourceVehicleTachograph): len(s.state.VehicleTachograph),
		},
	}
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
	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.resourceSliceLocked(resource)
	if err != nil {
		return nil, err
	}

	record := cloneRecord(payload)
	assignedID := strings.TrimSpace(recordID(record))
	if assignedID == "" {
		record["id"] = s.nextIDLocked(resource)
	} else {
		for _, existing := range list {
			if recordID(existing) == assignedID {
				return nil, ErrRecordConflict
			}
		}
	}
	list = append(list, record)
	updated := list
	if err = s.setResourceSliceLocked(resource, updated); err != nil {
		return nil, err
	}
	return cloneRecord(record), nil
}

func (s *Store) Patch(resource Resource, id string, patch Record) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	recordIDValue := strings.TrimSpace(id)
	if recordIDValue == "" {
		return nil, ErrRecordIDRequired
	}

	list, err := s.resourceSliceLocked(resource)
	if err != nil {
		return nil, err
	}
	for idx := range list {
		if recordID(list[idx]) != recordIDValue {
			continue
		}

		current := cloneRecord(list[idx])
		mergePatch(current, patch)
		current["id"] = recordIDValue
		list[idx] = current
		if err = s.setResourceSliceLocked(resource, list); err != nil {
			return nil, err
		}
		return cloneRecord(current), nil
	}
	return nil, ErrRecordNotFound
}

func (s *Store) Delete(resource Resource, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	recordIDValue := strings.TrimSpace(id)
	if recordIDValue == "" {
		return ErrRecordIDRequired
	}

	list, err := s.resourceSliceLocked(resource)
	if err != nil {
		return err
	}
	for idx := range list {
		if recordID(list[idx]) != recordIDValue {
			continue
		}
		updated := append(cloneRecords(list[:idx]), cloneRecords(list[idx+1:])...)
		return s.setResourceSliceLocked(resource, updated)
	}
	return ErrRecordNotFound
}

func (s *Store) AppendMessages(records []Record) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(records) == 0 {
		return
	}
	s.state.Messages = append(s.state.Messages, cloneRecords(records)...)
}

func (s *Store) Replace(resource Resource, records []Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.setResourceSliceLocked(resource, records); err != nil {
		return err
	}
	if resource == ResourceMessages {
		s.trackResourceCounterLocked(resource, s.state.Messages)
	}
	return nil
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
		if rawDelivery, okDelivery := anyAsMap(record["simDelivery"]); okDelivery {
			target.SimDelivery = cloneMap(rawDelivery)
		}

		if filter == "" || len(target.EventTypes) == 0 ||
			containsString(target.EventTypes, filter) {
			targets = append(targets, target)
		}
	}
	return targets
}

func (s *Store) resourceSliceLocked(resource Resource) ([]Record, error) {
	switch resource {
	case ResourceAddresses:
		return s.state.Addresses, nil
	case ResourceAssets:
		return s.state.Assets, nil
	case ResourceAssetLocation:
		return s.state.AssetLocation, nil
	case ResourceDrivers:
		return s.state.Drivers, nil
	case ResourceRoutes:
		return s.state.Routes, nil
	case ResourceFormTemplates:
		return s.state.FormTemplates, nil
	case ResourceFormSubmissions:
		return s.state.FormSubmissions, nil
	case ResourceLiveShares:
		return s.state.LiveShares, nil
	case ResourceMessages:
		return s.state.Messages, nil
	case ResourceWebhooks:
		return s.state.Webhooks, nil
	case ResourceVehicleStats:
		return s.state.VehicleStats, nil
	case ResourceHOSClocks:
		return s.state.HOSClocks, nil
	case ResourceHOSLogs:
		return s.state.HOSLogs, nil
	case ResourceDriverTachograph:
		return s.state.DriverTachograph, nil
	case ResourceVehicleTachograph:
		return s.state.VehicleTachograph, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedResource, resource)
	}
}

func (s *Store) setResourceSliceLocked(resource Resource, records []Record) error {
	switch resource {
	case ResourceAddresses:
		s.state.Addresses = cloneRecords(records)
	case ResourceAssets:
		s.state.Assets = cloneRecords(records)
	case ResourceAssetLocation:
		s.state.AssetLocation = cloneRecords(records)
	case ResourceDrivers:
		s.state.Drivers = cloneRecords(records)
	case ResourceRoutes:
		s.state.Routes = cloneRecords(records)
	case ResourceFormTemplates:
		s.state.FormTemplates = cloneRecords(records)
	case ResourceFormSubmissions:
		s.state.FormSubmissions = cloneRecords(records)
	case ResourceLiveShares:
		s.state.LiveShares = cloneRecords(records)
	case ResourceMessages:
		s.state.Messages = cloneRecords(records)
	case ResourceWebhooks:
		s.state.Webhooks = cloneRecords(records)
	case ResourceVehicleStats:
		s.state.VehicleStats = cloneRecords(records)
	case ResourceHOSClocks:
		s.state.HOSClocks = cloneRecords(records)
	case ResourceHOSLogs:
		s.state.HOSLogs = cloneRecords(records)
	case ResourceDriverTachograph:
		s.state.DriverTachograph = cloneRecords(records)
	case ResourceVehicleTachograph:
		s.state.VehicleTachograph = cloneRecords(records)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedResource, resource)
	}
	return nil
}

func (s *Store) rebuildCountersLocked() {
	s.counters = map[Resource]int{}

	s.trackResourceCounterLocked(ResourceAddresses, s.state.Addresses)
	s.trackResourceCounterLocked(ResourceAssets, s.state.Assets)
	s.trackResourceCounterLocked(ResourceDrivers, s.state.Drivers)
	s.trackResourceCounterLocked(ResourceRoutes, s.state.Routes)
	s.trackResourceCounterLocked(ResourceFormSubmissions, s.state.FormSubmissions)
	s.trackResourceCounterLocked(ResourceLiveShares, s.state.LiveShares)
	s.trackResourceCounterLocked(ResourceMessages, s.state.Messages)
	s.trackResourceCounterLocked(ResourceWebhooks, s.state.Webhooks)
	s.trackResourceCounterLocked(ResourceVehicleStats, s.state.VehicleStats)
}

func (s *Store) trackResourceCounterLocked(resource Resource, records []Record) {
	maxID := 0
	prefix := mustResourcePrefix(resource)

	for _, record := range records {
		nextID := parseTrailingInt(prefix, recordID(record))
		if nextID > maxID {
			maxID = nextID
		}
	}
	if maxID == 0 {
		maxID = len(records)
	}
	s.counters[resource] = maxID
}

func (s *Store) nextIDLocked(resource Resource) string {
	current := s.counters[resource]
	current++
	s.counters[resource] = current
	return fmt.Sprintf("%s-%d", mustResourcePrefix(resource), current)
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
