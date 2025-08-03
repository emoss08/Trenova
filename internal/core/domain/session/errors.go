/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package session

import "github.com/rotisserie/eris"

var (
	ErrNotFound   = eris.New("session not found")
	ErrNotActive  = eris.New("session is not active")
	ErrExpired    = eris.New("session is expired")
	ErrIPMismatch = eris.New("session ip mismatch")
)
