package restx

import (
	"cmp"
	"net/url"
	"slices"
	"strings"
)

const (
	redactedValue   = "REDACTED"
	minSecretLength = 4
)

func RedactURL(u *url.URL, keys []string) string {
	if u == nil {
		return ""
	}
	clone := *u
	if clone.RawQuery != "" && len(keys) > 0 {
		query := clone.Query()
		for name, values := range query {
			if !matchesKey(name, keys) {
				continue
			}
			for i := range values {
				values[i] = redactedValue
			}
		}
		clone.RawQuery = query.Encode()
	}
	return clone.Redacted()
}

func matchesKey(name string, keys []string) bool {
	for _, key := range keys {
		if strings.EqualFold(name, key) {
			return true
		}
	}
	return false
}

type redactor struct {
	replacer *strings.Replacer
}

func newRedactor(secrets []string) redactor {
	candidates := make([]string, 0, len(secrets)*3)
	seen := make(map[string]struct{}, len(secrets)*3)
	add := func(value string) {
		if len(value) < minSecretLength {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		candidates = append(candidates, value)
	}
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		add(secret)
		add(url.QueryEscape(secret))
		add(url.PathEscape(secret))
		if fields := strings.Fields(secret); len(fields) > 1 {
			for _, field := range fields[1:] {
				add(field)
				add(url.QueryEscape(field))
			}
		}
	}
	if len(candidates) == 0 {
		return redactor{}
	}
	slices.SortFunc(candidates, func(a, b string) int {
		return cmp.Compare(len(b), len(a))
	})
	pairs := make([]string, 0, len(candidates)*2)
	for _, candidate := range candidates {
		pairs = append(pairs, candidate, redactedValue)
	}
	return redactor{replacer: strings.NewReplacer(pairs...)}
}

func (r redactor) apply(value string) string {
	if r.replacer == nil {
		return value
	}
	return r.replacer.Replace(value)
}

func (r redactor) bytes(value []byte) []byte {
	if r.replacer == nil || len(value) == 0 {
		return value
	}
	return []byte(r.replacer.Replace(string(value)))
}
