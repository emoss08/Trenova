package tenant

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/emailutils"
)

const (
	MaxUsernameLength   = 20
	fallbackUsername    = "user"
	numberedCandidates  = 9
	usernameTrimSymbols = "._-"
)

func UsernameBase(email string) string {
	local := emailutils.LocalPart(email)

	var builder strings.Builder
	builder.Grow(min(len(local), MaxUsernameLength))
	for _, r := range local {
		if builder.Len() >= MaxUsernameLength {
			break
		}
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			builder.WriteRune(r)
		}
	}

	return strings.Trim(builder.String(), usernameTrimSymbols)
}

func UsernameCandidates(base, randomSuffix string) []string {
	base = strings.Trim(base, usernameTrimSymbols)
	if base == "" {
		base = fallbackUsername
	}

	candidates := make([]string, 0, numberedCandidates+2)
	candidates = append(candidates, truncateUsername(base, ""))
	for n := 2; n <= numberedCandidates+1; n++ {
		candidates = append(candidates, truncateUsername(base, "-"+strconv.Itoa(n)))
	}
	if randomSuffix != "" {
		candidates = append(candidates, truncateUsername(base, "-"+randomSuffix))
	}

	return candidates
}

func truncateUsername(base, suffix string) string {
	room := MaxUsernameLength - len(suffix)
	if len(base) > room {
		base = strings.TrimRight(base[:room], usernameTrimSymbols)
	}

	return base + suffix
}
