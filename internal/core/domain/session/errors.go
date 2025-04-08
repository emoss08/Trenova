package session

import "github.com/rotisserie/eris"

var (
	ErrNotFound   = eris.New("session not found")
	ErrNotActive  = eris.New("session is not active")
	ErrExpired    = eris.New("session is expired")
	ErrIPMismatch = eris.New("session ip mismatch")
)
