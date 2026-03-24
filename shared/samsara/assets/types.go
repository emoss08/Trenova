package assets

import samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"

type Type = samsaraspec.AssetsCreateAssetRequestBodyType

const (
	TypeUncategorized Type = "uncategorized"
	TypeTrailer       Type = "trailer"
	TypeEquipment     Type = "equipment"
	TypeUnpowered     Type = "unpowered"
	TypeVehicle       Type = "vehicle"
)

type Asset = samsaraspec.AssetResponseBody

type CreateRequest = samsaraspec.AssetsCreateAssetRequestBody

type UpdateRequest = samsaraspec.AssetsUpdateAssetRequestBody

type createResponse = samsaraspec.AssetsCreateAssetResponseBody

type updateResponse = samsaraspec.AssetsUpdateAssetResponseBody

type ListResponse = samsaraspec.AssetsListAssetsResponseBody

type PaginationResponse = samsaraspec.GoaPaginationResponseResponseBody

type StreamPaginationResponse = samsaraspec.GoaPaginationWithTokensResponseResponseBody

type StreamRecord = samsaraspec.LocationAndSpeedResponseResponseBody

type StreamAsset = samsaraspec.AssetResponseResponseBody

type LocationStreamResponse = samsaraspec.LocationAndSpeedGetLocationAndSpeedResponseBody

type CurrentLocationsResponse struct {
	Data []StreamRecord `json:"data"`
}

type HistoricalLocationsResponse struct {
	Data []StreamRecord `json:"data"`
}
