// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package pulid

import (
	"crypto/rand"
	"database/sql/driver"
	"fmt"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/rotisserie/eris"
)

var (
	ErrScanningNil   = eris.New("pulid: scanning nil into PULID")
	ErrInvalidLength = eris.New("pulid: invalid length")
)

// ID implements a PULID - a prefixed ULID.
type ID string

func (u ID) IsNil() bool { return u == "" }

func (u ID) IsNotNil() bool { return !u.IsNil() }

var Nil = ID("")

var NilPtr = &Nil

// The default entropy source with a mutex for thread safety
var (
	defaultEntropySource *ulid.MonotonicEntropy
	entropyMutex         sync.Mutex
)

func init() {
	// Seed the default entropy source.
	defaultEntropySource = ulid.Monotonic(rand.Reader, 0)
}

// newULID returns a new ULID for time.Now() using the default entropy source.
func newULID() ulid.ULID {
	// Use a mutex to ensure thread-safe access to the entropy source
	entropyMutex.Lock()
	defer entropyMutex.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), defaultEntropySource)
}

// MustNew returns a new PULID for time.Now() given a prefix. This uses the default entropy source.
func MustNew(prefix string) ID { return ID(prefix + newULID().String()) }

func MustNewPtr(prefix string) *ID {
	id := MustNew(prefix)
	return &id
}

// Must return a pointer to a PULID
func Must(prefix string) *ID {
	id := MustNew(prefix)
	return &id
}

// Scan implements the Scanner interface.
func (u *ID) Scan(src any) error {
	if src == nil {
		return ErrScanningNil
	}

	switch v := src.(type) {
	case string:
		*u = ID(v)
	case ID:
		*u = v
	default:
		return fmt.Errorf("pulid: unexpected type, %T", v)
	}
	return nil
}

// Value implements the driver Valuer interface.
// Returns the string representation of the ID that can be used in SQL queries.
func (u ID) Value() (driver.Value, error) {
	if u.IsNil() {
		//nolint:nilnil // nil is a valid value for a PULID
		return nil, nil
	}
	return string(u), nil
}

func ConvertFromPtr(ptr *ID) ID {
	if ptr == nil {
		return Nil
	}

	return *ptr
}

// String returns the string representation of the PULID.
func (u ID) String() string { return string(u) }

// Parse parses a PULID from a string.
func Parse(s string) (ID, error) {
	if len(s) < 27 {
		return Nil, ErrInvalidLength
	}
	return ID(s), nil
}

// MustParse parses a PULID from a string. If the string is not a valid PULID, it panics.
func MustParse(s string) (ID, error) {
	id, err := Parse(s)
	if err != nil {
		return Nil, eris.Wrap(err, "pulid: failed to parse PULID")
	}
	return id, nil
}

func Equals(a, b ID) bool {
	return a == b
}
