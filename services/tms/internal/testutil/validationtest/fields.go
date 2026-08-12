package validationtest

import "github.com/emoss08/trenova/pkg/errortypes"

// FieldCodes returns the error codes recorded against one field.
//
// Assertions on a single field are what a validation test usually wants:
// HasErrors is true for an entity missing any required field, so it cannot show
// that the rule under test is the one that fired, and a test written that way
// keeps passing when that rule is deleted.
func FieldCodes(multiErr *errortypes.MultiError, field string) []string {
	if multiErr == nil {
		return nil
	}

	codes := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		if err.Field == field {
			codes = append(codes, string(err.Code))
		}
	}

	return codes
}
