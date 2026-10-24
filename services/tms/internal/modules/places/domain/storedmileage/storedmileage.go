package storedmileage

import (
	"sort"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*StoredMileage)(nil)
	_ domaintypes.PostgresSearchable     = (*StoredMileage)(nil)
	_ pagination.CursorEntity            = (*StoredMileage)(nil)
)

const (
	StatusActive   = "Active"
	StatusInactive = "Inactive"
	SourcePCMiler  = "PCMiler"

	routeHashLength    = 64
	maxShortCodeLength = 50
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

type JurisdictionDistance struct {
	Country  string  `json:"country"`
	Code     string  `json:"code"`
	Distance float64 `json:"distance"`
	Toll     float64 `json:"toll,omitempty"`
	Ferry    float64 `json:"ferry,omitempty"`
}

type StoredMileage struct {
	keel.Table            `keel:"stored_mileages"`
	ID                    pulid.ID               `json:"id" keel:"id,pk,autoid:smg_"`
	BusinessUnitID        pulid.ID               `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID        pulid.ID               `json:"organizationId" keel:"organization_id,pk,tenant"`
	Status                string                 `json:"status" keel:"status"`
	OriginKey             StopKey                `json:"originKey" keel:"origin_key,jsonb"`
	DestinationKey        StopKey                `json:"destinationKey" keel:"destination_key,jsonb"`
	IntermediateKeys      []StopKey              `json:"intermediateKeys" keel:"intermediate_keys,jsonb,nullzero"`
	RouteSignature        string                 `json:"routeSignature" keel:"route_signature"`
	RouteHash             string                 `json:"routeHash" keel:"route_hash"`
	Distance              float64                `json:"distance" keel:"distance"`
	DistanceUnits         string                 `json:"distanceUnits" keel:"distance_units"`
	Provider              string                 `json:"provider" keel:"provider"`
	Source                string                 `json:"source" keel:"source"`
	RoutingType           string                 `json:"routingType" keel:"routing_type"`
	Method                string                 `json:"method" keel:"method"`
	LocationGranularity   string                 `json:"locationGranularity" keel:"location_granularity"`
	DataVersion           string                 `json:"dataVersion" keel:"data_version"`
	DistanceProfileID     pulid.ID               `json:"distanceProfileId" keel:"distance_profile_id"`
	DistanceProfileName   string                 `json:"distanceProfileName" keel:"distance_profile_name,nullzero"`
	Hazmat                bool                   `json:"hazmat" keel:"hazmat"`
	HazmatTypes           []string               `json:"hazmatTypes" keel:"hazmat_types,nullzero"`
	HazmatSignature       string                 `json:"hazmatSignature" keel:"hazmat_signature"`
	ProviderMetadata      map[string]any         `json:"providerMetadata" keel:"provider_metadata,jsonb,nullzero"`
	JurisdictionDistances []JurisdictionDistance `json:"jurisdictionDistances,omitempty" keel:"jurisdiction_distances,jsonb,nullzero"`
	HitCount              int64                  `json:"hitCount" keel:"hit_count"`
	LastUsedAt            *int64                 `json:"lastUsedAt" keel:"last_used_at"`
	LastCalculatedAt      int64                  `json:"lastCalculatedAt" keel:"last_calculated_at"`
	Version               int64                  `json:"version" keel:"version,version"`
	CreatedAt             int64                  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt             int64                  `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector          string                 `json:"-"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
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
	s.ApplyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&s.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&s.Status,
			validation.Required.Error("Status is required"),
			validation.In(StatusActive, StatusInactive).Error("Status is invalid"),
		),
		validation.Field(&s.RouteSignature,
			validation.Required.Error("Route signature is required"),
		),
		validation.Field(&s.RouteHash,
			validation.Required.Error("Route hash is required"),
			validation.Length(routeHashLength, routeHashLength).
				Error("Route hash must be a 64 character digest"),
		),
		validation.Field(&s.Distance,
			validation.Min(0.0).Error("Distance must be greater than or equal to 0"),
		),
		validation.Field(&s.DistanceUnits,
			validation.Required.Error("Distance units are required"),
			validation.Length(1, maxShortCodeLength).
				Error("Distance units cannot be longer than 50 characters"),
		),
		validation.Field(&s.Provider,
			validation.Required.Error("Provider is required"),
			validation.Length(1, maxShortCodeLength).
				Error("Provider cannot be longer than 50 characters"),
		),
		validation.Field(&s.Source,
			validation.Required.Error("Source is required"),
			validation.Length(1, maxShortCodeLength).
				Error("Source cannot be longer than 50 characters"),
		),
		validation.Field(&s.RoutingType,
			validation.Required.Error("Routing type is required"),
			validation.Length(1, maxShortCodeLength).
				Error("Routing type cannot be longer than 50 characters"),
		),
		validation.Field(&s.Method,
			validation.Required.Error("Method is required"),
			validation.Length(1, maxShortCodeLength).
				Error("Method cannot be longer than 50 characters"),
		),
		validation.Field(&s.LocationGranularity,
			validation.Required.Error("Location granularity is required"),
			validation.Length(1, maxShortCodeLength).
				Error("Location granularity cannot be longer than 50 characters"),
		),
		validation.Field(&s.DistanceProfileID,
			validation.Required.Error("Distance profile is required"),
		),
		validation.Field(&s.HitCount,
			validation.Min(int64(0)).Error("Hit count cannot be negative"),
		),
	))
}

func (s *StoredMileage) HasJurisdictionBreakdown() bool {
	return s != nil && len(s.JurisdictionDistances) > 0
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
			{
				Name:   "route_signature",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "distance_profile_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "provider", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
		},
	}
}
