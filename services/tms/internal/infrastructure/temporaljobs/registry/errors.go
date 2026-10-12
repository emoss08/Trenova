package registry

import "errors"

var ErrNoWorkersRegistered = errors.New("no workers registered")

var ErrTemporalClientNotConfigured = errors.New("temporal client is not configured")
