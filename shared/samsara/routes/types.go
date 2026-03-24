package routes

import samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"

type Route = samsaraspec.BaseRouteResponseObjectResponseBody

// Stop is used by callers when constructing route payloads.
type Stop = samsaraspec.CreateRoutesStopRequestObjectRequestBody

type CreateRequest = samsaraspec.RoutesCreateRouteRequestBody

type UpdateRequest = samsaraspec.RoutesPatchRouteRequestBody

type ListResponse = samsaraspec.RoutesFetchRoutesResponseBody

type routeResponse = samsaraspec.RoutesFetchRouteResponseBody

type createResponse = samsaraspec.RoutesCreateRouteResponseBody

type updateResponse = samsaraspec.RoutesPatchRouteResponseBody
