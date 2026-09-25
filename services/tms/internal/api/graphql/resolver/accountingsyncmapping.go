package resolver

import "github.com/emoss08/trenova/pkg/errortypes"

func errAccountingNeedsAPerson() error {
	return errortypes.NewAuthorizationError(
		"Connecting or disconnecting an accounting system needs a signed-in person, not an API key",
	)
}
