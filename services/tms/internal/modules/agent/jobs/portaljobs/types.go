// Package portaljobs follows an action an agent asked a portal to take, once
// a person approved it, until the portal finishes: it polls the vendor, records
// what the portal said, and ends the wait that picks the work up again.
package portaljobs

import "time"

const (
	pollInterval = 10 * time.Second
	actDeadline  = 20 * time.Minute
	pollTimeout  = time.Minute
	timedOutNote    = "The portal task was still running after 20 minutes, so it was stopped."
	unreachableNote = "Trenova could not reach the vendor to follow the portal task, so it was stopped."
)
