package carrierok

import (
	"time"

	"github.com/emoss08/trenova/shared/restx"
)

type Endpoint string

const (
	EndpointProfile          Endpoint = "profile"
	EndpointProfileLite      Endpoint = "profile-lite"
	EndpointProfileFMCSA     Endpoint = "profile-fmcsa"
	EndpointSearch           Endpoint = "search"
	EndpointAutocomplete     Endpoint = "autocomplete"
	EndpointMonitoringAdd    Endpoint = "monitoring-add"
	EndpointMonitoringRemove Endpoint = "monitoring-remove"
	EndpointMonitoringList   Endpoint = "monitoring-list"
)

const (
	pathProfile          = "/v2/profile"
	pathProfileLite      = "/v2/profile-lite"
	pathProfileFMCSA     = "/v2/profile-fmcsa"
	pathSearch           = "/v2/search"
	pathAutocomplete     = "/v2/autocomplete"
	pathMonitoringAdd    = "/v2/monitoring/add"
	pathMonitoringRemove = "/v2/monitoring/remove"
	pathMonitoringList   = "/v2/monitoring/list"
)

var requestsPerMinute = map[Endpoint]int{
	EndpointProfile:          5,
	EndpointProfileLite:      30,
	EndpointProfileFMCSA:     150,
	EndpointAutocomplete:     2000,
	EndpointSearch:           30,
	EndpointMonitoringAdd:    60,
	EndpointMonitoringRemove: 60,
	EndpointMonitoringList:   120,
}

func Endpoints() []Endpoint {
	return []Endpoint{
		EndpointProfile,
		EndpointProfileLite,
		EndpointProfileFMCSA,
		EndpointSearch,
		EndpointAutocomplete,
		EndpointMonitoringAdd,
		EndpointMonitoringRemove,
		EndpointMonitoringList,
	}
}

func DefaultPolicies() map[Endpoint]restx.Bucket {
	policies := make(map[Endpoint]restx.Bucket, len(requestsPerMinute))
	for endpoint, limit := range requestsPerMinute {
		burst := max(1, (limit+59)/60)
		if endpoint == EndpointProfile {
			burst = 1
		}
		policies[endpoint] = restx.Bucket{
			Limit:  limit,
			Period: time.Minute,
			Burst:  burst,
			Cost:   1,
		}
	}
	return policies
}
