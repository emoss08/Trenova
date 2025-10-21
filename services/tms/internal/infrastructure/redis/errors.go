package redis

import "errors"

var (
	ErrConnectionNotInitialized   = errors.New("connection not initialized")
	ErrScriptLoaderNotInitialized = errors.New("script loader not initialized")
)
