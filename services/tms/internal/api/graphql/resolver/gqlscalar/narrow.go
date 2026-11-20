package gqlscalar

import "github.com/emoss08/trenova/pkg/errortypes"

type NarrowInteger interface {
	~int8 | ~int16 | ~int32 | ~int64
}

func NarrowInt[D NarrowInteger](field string, value int) (D, error) {
	narrowed := D(value)
	if int(narrowed) != value {
		return 0, NarrowRangeError[D](field)
	}

	return narrowed, nil
}

func NarrowRangeError[D NarrowInteger](field string) *errortypes.Error {
	minimum, maximum := narrowBounds[D]()

	return errortypes.NewValidationError(
		field,
		errortypes.ErrInvalid,
		"Must be between {0} and {1}",
		minimum,
		maximum,
	)
}

func NarrowIntValue[D NarrowInteger](field string, value *int) (D, error) {
	if value == nil {
		return 0, nil
	}

	return NarrowInt[D](field, *value)
}

func NarrowIntPtr[D NarrowInteger](field string, value *int) (*D, error) {
	if value == nil {
		return nil, nil //nolint:nilnil // an omitted input leaves the optional field unset
	}
	narrowed, err := NarrowInt[D](field, *value)
	if err != nil {
		return nil, err
	}

	return &narrowed, nil
}

func ApplyInt[D NarrowInteger](field string, value *int, dest *D) error {
	if value == nil {
		return nil
	}
	narrowed, err := NarrowInt[D](field, *value)
	if err != nil {
		return err
	}
	*dest = narrowed

	return nil
}

func ApplyIntPtr[D NarrowInteger](field string, value *int, dest **D) error {
	if value == nil {
		return nil
	}
	narrowed, err := NarrowInt[D](field, *value)
	if err != nil {
		return err
	}
	*dest = &narrowed

	return nil
}

func narrowBounds[D NarrowInteger]() (minimum, maximum int64) {
	largest := D(1)
	for {
		next := largest<<1 | 1
		if next <= largest {
			break
		}
		largest = next
	}

	return -int64(largest) - 1, int64(largest)
}
