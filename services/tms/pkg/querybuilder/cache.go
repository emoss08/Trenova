package querybuilder

import (
	"maps"
	"reflect"
	"sync"

	"github.com/emoss08/trenova/pkg/domaintypes"
)

var (
	fieldMapCache   = make(map[reflect.Type]map[string]string)
	fieldMapCacheMu sync.RWMutex

	fieldConfigCache   = make(map[reflect.Type]*domaintypes.FieldConfiguration)
	fieldConfigCacheMu sync.RWMutex
)

func getOrComputeFieldMap(t reflect.Type, compute func() map[string]string) map[string]string {
	fieldMapCacheMu.RLock()
	if cached, ok := fieldMapCache[t]; ok {
		fieldMapCacheMu.RUnlock()
		result := make(map[string]string, len(cached))
		maps.Copy(result, cached)
		return result
	}
	fieldMapCacheMu.RUnlock()

	fieldMapCacheMu.Lock()
	defer fieldMapCacheMu.Unlock()

	if cached, ok := fieldMapCache[t]; ok {
		result := make(map[string]string, len(cached))
		maps.Copy(result, cached)
		return result
	}

	computed := compute()
	fieldMapCache[t] = computed

	result := make(map[string]string, len(computed))
	maps.Copy(result, computed)
	return result
}

func getOrComputeFieldConfig[T domaintypes.PostgresSearchable](
	entity T,
) *domaintypes.FieldConfiguration {
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	fieldConfigCacheMu.RLock()
	if cached, ok := fieldConfigCache[t]; ok {
		fieldConfigCacheMu.RUnlock()
		return cached
	}
	fieldConfigCacheMu.RUnlock()

	fieldConfigCacheMu.Lock()
	defer fieldConfigCacheMu.Unlock()

	if cached, ok := fieldConfigCache[t]; ok {
		return cached
	}

	config := NewFieldConfigBuilder(entity).
		WithAutoMapping().
		WithAllFieldsFilterable().
		WithAllFieldsSortable().
		WithAutoEnumDetection().
		WithRelationshipFields().
		Build()

	fieldConfigCache[t] = config
	return config
}

func GetFieldConfiguration[T domaintypes.PostgresSearchable](
	entity T,
) *domaintypes.FieldConfiguration {
	return getOrComputeFieldConfig(entity)
}

func WarmFieldConfigCache(entities ...domaintypes.PostgresSearchable) {
	for _, entity := range entities {
		getOrComputeFieldConfig(entity)
	}
}

func ClearCaches() {
	fieldMapCacheMu.Lock()
	fieldMapCache = make(map[reflect.Type]map[string]string)
	fieldMapCacheMu.Unlock()

	fieldConfigCacheMu.Lock()
	fieldConfigCache = make(map[reflect.Type]*domaintypes.FieldConfiguration)
	fieldConfigCacheMu.Unlock()
}
