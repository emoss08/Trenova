package addresses

import samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"

type Address = samsaraspec.Address

type CreateRequest = samsaraspec.CreateAddressRequest

type Geofence = samsaraspec.CreateAddressRequestGeofence

type GeofenceCircle = samsaraspec.AddressGeofenceCircle

type GeofencePolygon = samsaraspec.AddressGeofencePolygon

type GeofenceVertex = samsaraspec.AddressGeofencePolygonVertices

type UpdateRequest = samsaraspec.UpdateAddressRequest

type ListPage = samsaraspec.ListAddressesResponse

type PaginationResponse = samsaraspec.PaginationResponse

type AddressResponse = samsaraspec.AddressResponse
