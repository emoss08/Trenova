package routes

import samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"

type Route = samsaraspec.BaseRouteWithOrdersResponseObjectResponseBody

type Stop = samsaraspec.CreateRouteStopWithOrdersRequestObjectRequestBody

type UpdateStop = samsaraspec.UpdateRoutesStopRequestObjectRequestBody

type AppointmentWindow = samsaraspec.RouteStopAppointmentWindowRequestBody

type CreateRequest = samsaraspec.RoutesCreateRouteRequestBody

type UpdateRequest = samsaraspec.RoutesPatchRouteRequestBody

type ListResponse = samsaraspec.RoutesFetchRoutesResponseBody

type routeResponse = samsaraspec.RoutesFetchRouteResponseBody

type createResponse = samsaraspec.RoutesCreateRouteResponseBody

type updateResponse = samsaraspec.RoutesPatchRouteResponseBody
