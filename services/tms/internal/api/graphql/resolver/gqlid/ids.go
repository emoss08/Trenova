package gqlid

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func OptionalPulid(raw *string) (*pulid.ID, error) {
	if raw == nil || *raw == "" {
		return nil, nil //nolint:nilnil // an absent optional ID is not an error
	}
	id, err := pulid.Parse(*raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func IDValuePtr(id pulid.ID) *pulid.ID {
	if id.IsNil() {
		return nil
	}
	value := id
	return &value
}

func PulidsFromStrings(values []string, field string) ([]pulid.ID, error) {
	if len(values) == 0 {
		return nil, nil
	}

	return ParsePulids(field, values)
}

func ParseIDs(values []string) ([]pulid.ID, error) {
	ids := make([]pulid.ID, 0, len(values))
	for _, value := range values {
		id, err := pulid.MustParse(value)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func ParseCatalogIDs(values []string) ([]pulid.ID, error) {
	ids := make([]pulid.ID, 0, len(values))
	for _, value := range values {
		id := pulid.ID(strings.TrimSpace(value))
		if id.IsNil() {
			return nil, errortypes.NewValidationError(
				"ids",
				errortypes.ErrInvalid,
				"Catalog id is required",
			)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func OptionalID(value *string) (pulid.ID, error) {
	if value == nil || *value == "" {
		return pulid.Nil, nil
	}
	return pulid.MustParse(*value)
}

// idPtrFromPtr renders an optional id, which is nil both when the pointer is
// absent and when it points at a zero id.
func IDPtrFromPtr(id *pulid.ID) *string {
	if id == nil {
		return nil
	}

	return IDPtr(*id)
}

func IDPtr(id pulid.ID) *string {
	if id.IsNil() {
		return nil
	}
	value := id.String()
	return &value
}

// requiredID parses an id a mutation input declares as non-null.
//
// GraphQL only guarantees the field is present, not that it holds anything: a
// form that posts a half-filled row sends an empty string, and the raw parse
// failure that produced surfaces in the client as "pulid: invalid length" with
// nothing naming what was blank. This returns the field-level error the form
// can attach to the control instead.
func RequiredID(field, value string) (pulid.ID, error) {
	if value == "" {
		return pulid.Nil, errortypes.NewValidationError(
			field,
			errortypes.ErrRequired,
			"This field is required",
		)
	}

	id, err := pulid.Parse(value)
	if err != nil {
		return pulid.Nil, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"This value is not a valid identifier",
		)
	}

	return id, nil
}

// optionalScopedID parses an optional id, naming the field when it holds
// something that is not one.
func OptionalScopedID(field string, value *string) (pulid.ID, error) {
	if value == nil || *value == "" {
		return pulid.Nil, nil
	}

	return RequiredID(field, *value)
}

func PulidsToStrings(ids []pulid.ID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func ParsePulids(field string, raw []string) ([]pulid.ID, error) {
	out := make([]pulid.ID, 0, len(raw))
	for i, value := range raw {
		id, err := pulid.MustParse(value)
		if err != nil {
			return nil, errortypes.NewValidationError(
				fmt.Sprintf("%s[%d]", field, i), errortypes.ErrInvalid, "Invalid identifier",
			)
		}
		out = append(out, id)
	}
	return out, nil
}

func InvalidIDError(field, message string) error {
	return errortypes.NewValidationError(field, errortypes.ErrInvalid, message)
}

func IDPtrFromPulidPtr(value *pulid.ID) *string {
	if value == nil || value.IsNil() {
		return nil
	}
	converted := value.String()
	return &converted
}

func PulidPtrFromOptionalString(value *string) (*pulid.ID, error) {
	if value == nil || *value == "" {
		return nil, nil //nolint:nilnil // an omitted ID stays unset
	}
	parsed, err := pulid.MustParse(*value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
