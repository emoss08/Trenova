// Package remedy names what a person can do about a refusal or a warning.
//
// A write is often refused, or warned about, for something outside it: a
// driver who was never invited to the portal, an organization with no email
// profile. Saying so leaves the person to work out where to go. A remedy says
// what would put it right, by kind and by the record it is done to, so the
// surface that shows the refusal can offer the fix itself: the action, where
// the person may take it and an API exists, or the place to do it otherwise.
//
// The package holds only the kinds and how one rides on an error, so the
// module that refuses (which knows the fix) and the module that shows the
// refusal (which offers it) meet without importing each other. Adding a
// remedy is one kind here, one Mark where the refusal is made, and one entry
// in the client's remedy registry.
package remedy

import (
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
)

// Kind names one fix. The client keeps what each kind is called, who may take
// it and how; the server only says which applies.
type Kind string

const (
	// InviteDriverToPortal invites a worker to the driver portal, so a message
	// sent to them has somewhere to arrive. It is done to the worker.
	InviteDriverToPortal = Kind("invite_driver_to_portal")
	// SetUpEmailProfile gives the organization a profile to send email from.
	// It is done to no record.
	SetUpEmailProfile = Kind("set_up_email_profile")
)

const param = "remedy"

// Mark says what would put a refusal right. It returns the error it was given.
func Mark(err *errortypes.BusinessError, kind Kind) *errortypes.BusinessError {
	return err.WithParam(param, string(kind))
}

// Of is the remedy a refusal names, when it names one.
func Of(err error) (Kind, bool) {
	var business *errortypes.BusinessError
	if !errors.As(err, &business) || business == nil {
		return "", false
	}
	kind, ok := business.Params[param]

	return Kind(kind), ok && kind != ""
}
