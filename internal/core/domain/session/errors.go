package session

import "github.com/rotisserie/eris"

var (
	SessionNotFound   = eris.New("session not found")
	SessionNotActive  = eris.New("session is not active")
	SessionExpired    = eris.New("session is expired")
	SessionIPMismatch = eris.New("session ip mismatch")
)
