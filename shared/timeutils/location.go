package timeutils

import (
	"errors"
	"sync"
	"time"
)

const MaxLocationNameLength = 64

var (
	ErrLocationNameTooLong = errors.New("timezone name is too long")

	knownLocations sync.Map
)

func LookupLocation(name string) (*time.Location, error) {
	if name == "" {
		return time.UTC, nil
	}
	if len(name) > MaxLocationNameLength {
		return nil, ErrLocationNameTooLong
	}
	if cached, ok := knownLocations.Load(name); ok {
		if location, isLocation := cached.(*time.Location); isLocation {
			return location, nil
		}
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		return nil, err
	}
	knownLocations.Store(name, location)

	return location, nil
}
