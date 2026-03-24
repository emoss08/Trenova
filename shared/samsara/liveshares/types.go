package liveshares

import samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"

type ShareType = samsaraspec.LiveSharingLinksCreateLiveSharingLinkRequestBodyType

const (
	ShareTypeAssetsLocation     ShareType = samsaraspec.LiveSharingLinksCreateLiveSharingLinkRequestBodyTypeAssetsLocation
	ShareTypeAssetsNearLocation ShareType = samsaraspec.LiveSharingLinksCreateLiveSharingLinkRequestBodyTypeAssetsNearLocation
	ShareTypeAssetsOnRoute      ShareType = samsaraspec.LiveSharingLinksCreateLiveSharingLinkRequestBodyTypeAssetsOnRoute
)

type ListType = samsaraspec.GetLiveSharingLinksParamsType

const (
	ListTypeAll                ListType = samsaraspec.GetLiveSharingLinksParamsTypeAll
	ListTypeAssetsLocation     ListType = samsaraspec.GetLiveSharingLinksParamsTypeAssetsLocation
	ListTypeAssetsNearLocation ListType = samsaraspec.GetLiveSharingLinksParamsTypeAssetsNearLocation
	ListTypeAssetsOnRoute      ListType = samsaraspec.GetLiveSharingLinksParamsTypeAssetsOnRoute
)

type Tag = samsaraspec.GoaTagTinyResponseResponseBody

type Location = samsaraspec.AssetsLocationLinkConfigAddressDetailsObject

type AssetsLocationLinkConfig = samsaraspec.AssetsLocationLinkResponseConfigObjectResponseBody

type AssetsNearLocationLinkConfig = samsaraspec.AssetsNearLocationLinkConfigObjectResponseBody

type AssetsOnRouteLinkConfig = samsaraspec.AssetsOnRouteLinkConfigObjectResponseBody

type LiveShare = samsaraspec.LiveSharingLinkFullResponseObjectResponseBody

type AssetsLocationLinkRequestConfig = samsaraspec.AssetsLocationLinkRequestConfigObject

type CreateRequest = samsaraspec.LiveSharingLinksCreateLiveSharingLinkRequestBody

type UpdateRequest = samsaraspec.LiveSharingLinksUpdateLiveSharingLinkRequestBody

type ListPage = samsaraspec.LiveSharingLinksGetLiveSharingLinksResponseBody

type PaginationResponse = samsaraspec.GoaPaginationResponseResponseBody

type createResponse = samsaraspec.LiveSharingLinksCreateLiveSharingLinkResponseBody

type updateResponse = samsaraspec.LiveSharingLinksUpdateLiveSharingLinkResponseBody
