package versionutils

import (
	"cmp"
	"errors"
	"strconv"
	"strings"
)

var ErrInvalidVersion = errors.New("version must be MAJOR.MINOR.PATCH")

// Version is a MAJOR.MINOR.PATCH release number. Pre-release and build suffixes
// are not accepted: the versions compared here are the ones a release pipeline
// stamps, and a suffix would have to be ordered by rules nobody has agreed.
type Version struct {
	Major int
	Minor int
	Patch int
}

// Parse reads a release number, tolerating a leading "v".
func Parse(raw string) (Version, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(raw), "v")
	parts := strings.Split(trimmed, ".")
	if len(parts) != 3 {
		return Version{}, ErrInvalidVersion
	}

	var numbers [3]int
	for i, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return Version{}, ErrInvalidVersion
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return Version{}, ErrInvalidVersion
		}
		numbers[i] = n
	}

	return Version{Major: numbers[0], Minor: numbers[1], Patch: numbers[2]}, nil
}

// IsValid reports whether raw is a release number Parse accepts.
func IsValid(raw string) bool {
	_, err := Parse(raw)
	return err == nil
}

// Compare orders two versions: negative when a is older, zero when equal,
// positive when a is newer.
func Compare(a, b Version) int {
	if c := cmp.Compare(a.Major, b.Major); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Minor, b.Minor); c != 0 {
		return c
	}

	return cmp.Compare(a.Patch, b.Patch)
}

// AtLeast reports whether current meets minimum. An empty minimum is met by
// anything; an unparseable current meets no minimum, because a build that
// cannot say what it is cannot be shown to be new enough.
func AtLeast(current, minimum string) bool {
	if strings.TrimSpace(minimum) == "" {
		return true
	}

	want, err := Parse(minimum)
	if err != nil {
		return true
	}
	have, err := Parse(current)
	if err != nil {
		return false
	}

	return Compare(have, want) >= 0
}

func (v Version) String() string {
	return strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor) + "." + strconv.Itoa(v.Patch)
}
