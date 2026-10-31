package errortypes

import (
	"errors"

	"github.com/emoss08/trenova/shared/pulid"
)

type SessionContextStaleError struct {
	BaseError
	UserID         pulid.ID `json:"userId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

func NewSessionContextStaleError(
	userID, organizationID, businessUnitID pulid.ID,
) *SessionContextStaleError {
	return &SessionContextStaleError{
		BaseError: BaseError{
			Code:    ErrSessionContextStale,
			Message: "Your session changed in another tab. Reload this page to continue.",
		},
		UserID:         userID,
		OrganizationID: organizationID,
		BusinessUnitID: businessUnitID,
	}
}

func IsSessionContextStaleError(err error) bool {
	_, ok := errors.AsType[*SessionContextStaleError](err)
	return ok
}

func (e *SessionContextStaleError) Params() map[string]string {
	return map[string]string{
		"userId":         e.UserID.String(),
		"organizationId": e.OrganizationID.String(),
		"businessUnitId": e.BusinessUnitID.String(),
	}
}
