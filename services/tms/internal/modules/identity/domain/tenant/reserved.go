package tenant

import (
	"regexp"
	"strings"

	"github.com/emoss08/trenova/shared/emailutils"
)

var reservedUsernames = map[string]struct{}{
	"system":  {},
	"trenova": {},
	"support": {},
	"root":    {},
}

var reservedUsernamePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^system-[0-9a-z]{13}$`),
	regexp.MustCompile(`^sup-[0-9a-z]{16}$`),
}

const reservedEmailTopLevel = "invalid"

func IsReservedUsername(username string) bool {
	name := strings.ToLower(strings.TrimSpace(username))
	if name == "" {
		return false
	}
	if _, ok := reservedUsernames[name]; ok {
		return true
	}
	for _, pattern := range reservedUsernamePatterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

func IsReservedEmailAddress(address string) bool {
	domain := strings.TrimSuffix(
		strings.ToLower(strings.TrimSpace(emailutils.Domain(address))),
		".",
	)
	return domain == reservedEmailTopLevel || strings.HasSuffix(domain, "."+reservedEmailTopLevel)
}
