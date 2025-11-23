package pulid

import (
	"crypto/rand"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/oklog/ulid/v2"
)

var (
	ErrScanningNil   = errors.New("pulid: scanning nil into ID")
	ErrInvalidLength = errors.New("pulid: invalid length")
)

type ID string

func (u ID) IsNil() bool { return u == Nil || u == "" }

func (u ID) IsNotNil() bool { return !u.IsNil() }

var Nil = ID("")

var NilPtr = &Nil

var (
	defaultEntropySource *ulid.MonotonicEntropy
	entropyMutex         sync.Mutex
)

func init() {
	defaultEntropySource = ulid.Monotonic(rand.Reader, 0)
}

func newULID() ulid.ULID {
	entropyMutex.Lock()
	defer entropyMutex.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), defaultEntropySource)
}

func MustNew(prefix string) ID { return ID(prefix + newULID().String()) }

func MustNewPtr(prefix string) *ID {
	id := MustNew(prefix)
	return &id
}

func Must(prefix string) *ID {
	id := MustNew(prefix)
	return &id
}

func (u *ID) Scan(src any) error {
	if src == nil {
		*u = Nil
		return nil
	}

	switch v := src.(type) {
	case string:
		*u = ID(v)
	case ID:
		*u = v
	case []byte:
		*u = ID(v)
	default:
		return fmt.Errorf("pulid: unexpected type, %T", v)
	}
	return nil
}

func (u ID) Value() (driver.Value, error) {
	if u.IsNil() {
		return nil, nil //nolint:nilnil // nil is a valid value for a driver.Value
	}
	return string(u), nil
}

// ConvertFromPtr safely converts a pointer to a PULID to a PULID.
// If the pointer is nil, it returns a nil PULID.
func ConvertFromPtr(ptr *ID) ID {
	if ptr == nil {
		return Nil
	}

	return *ptr
}

func (u ID) String() string { return string(u) }

func Parse(s string) (ID, error) {
	if len(s) < 27 {
		return Nil, ErrInvalidLength
	}
	return ID(s), nil
}

func MustParse(s string) (ID, error) {
	id, err := Parse(s)
	if err != nil {
		return Nil, fmt.Errorf("pulid: failed to parse PULID: %w", err)
	}
	return id, nil
}

// Equals checks if two PULIDs are equal.
func Equals(a, b ID) bool {
	return a == b
}

func (u ID) MarshalJSON() ([]byte, error) {
	if u.IsNil() {
		return []byte("null"), nil
	}
	return sonic.Marshal(string(u))
}

func (u *ID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || len(data) == 0 {
		*u = Nil
		return nil
	}

	var s string
	if err := sonic.Unmarshal(data, &s); err != nil {
		return err
	}

	if s == "" {
		*u = Nil
		return nil
	}

	*u = ID(s)
	return nil
}

func (u ID) Time() (time.Time, error) {
	if u.IsNil() {
		return time.Time{}, errors.New("pulid: cannot get time from nil ID")
	}

	s := string(u)
	if len(s) < 26 {
		return time.Time{}, errors.New("pulid: ID too short to contain ULID")
	}

	ulidPart := s[len(s)-26:]

	lID, err := ulid.Parse(ulidPart)
	if err != nil {
		return time.Time{}, fmt.Errorf("pulid: failed to parse ULID part: %w", err)
	}

	ms := lID.Time()
	safeMs, err := utils.SafeUint64ToInt64(ms)
	if err != nil {
		return time.Time{}, fmt.Errorf("pulid: timestamp overflow: %w", err)
	}

	return time.UnixMilli(safeMs), nil
}

func (u ID) Prefix() string {
	if u.IsNil() {
		return ""
	}

	s := string(u)
	if len(s) <= 26 {
		return ""
	}

	return s[:len(s)-26]
}
