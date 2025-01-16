package search

import "github.com/rotisserie/eris"

var (
	ErrServiceStopped = eris.New("search service is stopped")
	ErrStartupFailed  = eris.New("search service failed to start")
)
