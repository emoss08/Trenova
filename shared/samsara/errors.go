package samsara

import samsaratypes "github.com/emoss08/trenova/shared/samsara/types"

type APIError = samsaratypes.APIError

var (
	IsRateLimit    = samsaratypes.IsRateLimit
	IsUnauthorized = samsaratypes.IsUnauthorized
	IsNotFound     = samsaratypes.IsNotFound
	Temporary      = samsaratypes.Temporary
)
