package storedmileage

import (
	"context"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*StoredMileage)(nil)
	_ validationframework.TenantedEntity = (*StoredMileage)(nil)
	_ domaintypes.PostgresSearchable     = (*StoredMileage)(nil)
	_ pagination.CursorEntity            = (*StoredMileage)(nil)
)

const (
	StatusActive   = "Active"
	StatusInactive = "Inactive"
	SourcePCMiler  = "PCMiler"
)

type StopKey struct {
	Method      string    `json:"method"`
	Key         string    `json:"key"`
	City        string    `json:"city,omitempty"`
	State       string    `json:"state,omitempty"`
	PostalCode  string    `json:"postalCode,omitempty"`
	PlaceID     string    `json:"placeId,omitempty"`
	Coordinates []float64 `json:"coordinates,omitempty"`
}

type StoredMileage struct {
	bun.BaseModel             `bun:"table:stored_mileages,alias:smg" json:"-"`
	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                  pulid.ID       `json:"id"                  bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID      pulid.ID       `json:"businessUnitId"      bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID      pulid.ID       `json:"organizationId"      bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Status              string         `json:"status"              bun:"status,type:VARCHAR(20),notnull"`
	OriginKey           StopKey        `json:"originKey"           bun:"origin_key,type:JSONB,notnull"`
	DestinationKey      StopKey        `json:"destinationKey"      bun:"destination_key,type:JSONB,notnull"`
	IntermediateKeys    []StopKey      `json:"intermediateKeys"    bun:"intermediate_keys,type:JSONB,nullzero"`
	RouteSignature      string         `json:"routeSignature"      bun:"route_signature,type:TEXT,notnull"`
	RouteHash           string         `json:"routeHash"           bun:"route_hash,type:VARCHAR(64),notnull"`
	Distance            float64        `json:"distance"            bun:"distance,type:FLOAT,notnull"`
	DistanceUnits       string         `json:"distanceUnits"       bun:"distance_units,type:VARCHAR(50),notnull"`
	Provider            string         `json:"provider"            bun:"provider,type:VARCHAR(50),notnull"`
	Source              string         `json:"source"              bun:"source,type:VARCHAR(50),notnull"`
	RoutingType         string         `json:"routingType"         bun:"routing_type,type:VARCHAR(50),notnull"`
	Method              string         `json:"method"              bun:"method,type:VARCHAR(50),notnull"`
	LocationGranularity string         `json:"locationGranularity" bun:"location_granularity,type:VARCHAR(50),notnull"`
	DataVersion         string         `json:"dataVersion"         bun:"data_version,type:VARCHAR(50),notnull"`
	DistanceProfileID   pulid.ID       `json:"distanceProfileId"   bun:"distance_profile_id,type:VARCHAR(100),notnull"`
	DistanceProfileName string         `json:"distanceProfileName" bun:"distance_profile_name,type:VARCHAR(100),nullzero"`
	Hazmat              bool           `json:"hazmat"              bun:"hazmat,type:BOOLEAN,notnull"`
	HazmatTypes         []string       `json:"hazmatTypes"         bun:"hazmat_types,array,type:TEXT[],nullzero"`
	HazmatSignature     string         `json:"hazmatSignature"     bun:"hazmat_signature,type:TEXT,notnull"`
	ProviderMetadata    map[string]any `json:"providerMetadata"    bun:"provider_metadata,type:JSONB,nullzero"`
	HitCount            int64          `json:"hitCount"            bun:"hit_count,type:BIGINT,notnull,default:0"`
	LastUsedAt          *int64         `json:"lastUsedAt"          bun:"last_used_at,type:BIGINT,nullzero"`
	LastCalculatedAt    int64          `json:"lastCalculatedAt"    bun:"last_calculated_at,type:BIGINT,notnull"`
	Version             int64          `json:"version"             bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt           int64          `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64          `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector        string         `json:"-"                   bun:"search_vector,type:TSVECTOR,scanonly"`

	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
}

func (s *StoredMileage) ApplyDefaults() {
	if s.Status == "" {
		s.Status = StatusActive
	}
	s.DistanceUnits = strings.TrimSpace(s.DistanceUnits)
	if s.DistanceUnits == "" {
		s.DistanceUnits = distanceprofile.DefaultDistanceUnits
	}
	if s.Source == "" {
		s.Source = SourcePCMiler
	}
	s.HazmatTypes = NormalizeHazmatTypes(s.HazmatTypes)
	s.Hazmat = len(s.HazmatTypes) > 0
	s.HazmatSignature = HazmatSignature(s.HazmatTypes)
	if s.LastCalculatedAt == 0 {
		s.LastCalculatedAt = timeutils.NowUnix()
	}
	if s.RouteHash == "" && s.RouteSignature != "" {
		s.RouteHash = hashutils.SHA256Hex(s.RouteSignature)
	}
}

func (s *StoredMileage) Validate(multiErr *errortypes.MultiError) {
	if s.Distance < 0 {
		multiErr.Add("distance", errortypes.ErrInvalid, "Distance must be greater than or equal to 0")
	}
	if s.RouteHash == "" {
		multiErr.Add("routeHash", errortypes.ErrRequired, "Route hash is required")
	}
	if s.DistanceUnits == "" {
		multiErr.Add("distanceUnits", errortypes.ErrRequired, "Distance units are required")
	}
	if s.RoutingType == "" {
		multiErr.Add("routingType", errortypes.ErrRequired, "Routing type is required")
	}
	if s.Method == "" {
		multiErr.Add("method", errortypes.ErrRequired, "Method is required")
	}
	if s.LocationGranularity == "" {
		multiErr.Add("locationGranularity", errortypes.ErrRequired, "Location granularity is required")
	}
	if s.DistanceProfileID.IsNil() {
		multiErr.Add("distanceProfileId", errortypes.ErrRequired, "Distance profile is required")
	}
}

func NormalizeHazmatTypes(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	sort.Strings(result)
	return result
}

func HazmatSignature(values []string) string {
	normalized := NormalizeHazmatTypes(values)
	if len(normalized) == 0 {
		return "none"
	}
	return strings.Join(normalized, ",")
}

func ConvertDistance(distance float64, fromUnits, toUnits string) float64 {
	if strings.EqualFold(fromUnits, toUnits) || fromUnits == "" || toUnits == "" {
		return distance
	}
	if strings.EqualFold(fromUnits, "Miles") && strings.EqualFold(toUnits, "Kilometers") {
		return distance * 1.609344
	}
	if strings.EqualFold(fromUnits, "Kilometers") && strings.EqualFold(toUnits, "Miles") {
		return distance / 1.609344
	}
	return distance
}

func (s *StoredMileage) GetID() pulid.ID             { return s.ID }
func (s *StoredMileage) GetCreatedAt() int64         { return s.CreatedAt }
func (s *StoredMileage) GetTableName() string        { return "stored_mileages" }
func (s *StoredMileage) GetOrganizationID() pulid.ID { return s.OrganizationID }
func (s *StoredMileage) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }

func (s *StoredMileage) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "smg",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "route_signature", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "distance_profile_name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{Name: "provider", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
		},
	}
}

func (s *StoredMileage) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("smg_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}
	return nil
}
