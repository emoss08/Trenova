package quotaservice

import "errors"

var (
	ErrTransactionRequired = errors.New(
		"quota enforcement must run inside the write transaction that performs the insert",
	)
	ErrInvalidQuantity = errors.New("quota quantity must not be negative")
	ErrRequestRequired = errors.New("quota request is required")
	ErrCounterMissing  = errors.New("plan limits a meter that has no quota counter")
	ErrTenantRequired  = errors.New("quota checks require an organization and business unit")
)
