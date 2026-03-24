package liveshares

import "errors"

var (
	ErrListLimitOutOfRange          = errors.New("live shares limit must be between 1 and 100")
	ErrIDRequired                   = errors.New("live share id is required")
	ErrNameRequired                 = errors.New("live share name is required")
	ErrAssetsLocationConfigRequired = errors.New(
		"assetsLocationLinkConfig is required for type assetsLocation",
	)
	ErrAssetsNearLocationAddressRequired = errors.New(
		"assetsNearLocationLinkConfig.addressId is required for type assetsNearLocation",
	)
	ErrRecurringRouteIDRequired = errors.New(
		"assetsOnRouteLinkConfig.recurringRouteId is required for type assetsOnRoute",
	)
)
