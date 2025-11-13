package reportjobs

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/uptrace/bun"
)

var ErrUnsupportedResourceType = fmt.Errorf("unsupported resource type")

type ResourceInfo struct {
	TableName string
	Alias     string
	Entity    domaintypes.PostgresSearchable
}

var resourceRegistry map[string]ResourceInfo

func init() {
	entities := map[string]domaintypes.PostgresSearchable{
		"user":                (*user.User)(nil),
		"customer":            (*customer.Customer)(nil),
		"shipment":            (*shipment.Shipment)(nil),
		"tractor":             (*tractor.Tractor)(nil),
		"trailer":             (*trailer.Trailer)(nil),
		"location":            (*location.Location)(nil),
		"worker":              (*worker.Worker)(nil),
		"equipment_type":      (*equipmenttype.EquipmentType)(nil),
		"commodity":           (*commodity.Commodity)(nil),
		"accessorial_charge":  (*accessorialcharge.AccessorialCharge)(nil),
		"hazardous_material":  (*hazardousmaterial.HazardousMaterial)(nil),
	}

	resourceRegistry = make(map[string]ResourceInfo, len(entities))

	for resourceType, entity := range entities {
		info := extractTableInfo(entity)
		resourceRegistry[resourceType] = info
	}
}

func extractTableInfo(entity domaintypes.PostgresSearchable) ResourceInfo {
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var tableName, alias string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Type == reflect.TypeOf(bun.BaseModel{}) {
			bunTag := field.Tag.Get("bun")
			if bunTag != "" {
				parts := strings.Split(bunTag, ",")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if strings.HasPrefix(part, "table:") {
						tableName = strings.TrimPrefix(part, "table:")
					} else if strings.HasPrefix(part, "alias:") {
						alias = strings.TrimPrefix(part, "alias:")
					}
				}
			}
			break
		}
	}

	return ResourceInfo{
		TableName: tableName,
		Alias:     alias,
		Entity:    entity,
	}
}

func GetResourceInfo(resourceType string) (ResourceInfo, error) {
	info, exists := resourceRegistry[resourceType]
	if !exists {
		return ResourceInfo{}, fmt.Errorf("%w: %s", ErrUnsupportedResourceType, resourceType)
	}
	return info, nil
}

func RegisterResource(resourceType string, entity domaintypes.PostgresSearchable) {
	info := extractTableInfo(entity)
	resourceRegistry[resourceType] = info
}
