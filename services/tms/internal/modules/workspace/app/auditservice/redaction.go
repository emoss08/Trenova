package auditservice

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
)

const (
	changeFrom           = "from"
	changeTo             = "to"
	RedactedValue        = "[REDACTED]"
	fingerprintHexLength = 8
	fingerprintTailRunes = 4
)

type fieldAction int

const (
	actionOmit fieldAction = iota + 1
	actionFingerprint
)

var fingerprintedNames = map[string]struct{}{
	"taxid":                {},
	"ein":                  {},
	"ssn":                  {},
	"socialsecuritynumber": {},
	"accountnumber":        {},
	"bankaccountnumber":    {},
	"routingnumber":        {},
	"cardnumber":           {},
	"iban":                 {},
}

//nolint:exhaustive // only resources with a field the name policy cannot judge are listed
var fingerprintedPaths = map[permission.Resource]map[string]fieldAction{
	permission.ResourceOrganization: {"logoUrl": actionOmit},
}

var secretShapes = []*regexp.Regexp{
	regexp.MustCompile(
		`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`,
	),
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`),
	regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9+.-]*://[^\s:/@]+:[^\s/@]+@\S+`),
}

type Redactor struct {
	pepper []byte
}

func NewRedactor(pepper string) *Redactor {
	return &Redactor{pepper: []byte(pepper)}
}

func (r *Redactor) RedactEntry(entry *audit.Entry) {
	if entry == nil {
		return
	}
	paths := fingerprintedPaths[entry.Resource]
	changed := false
	entry.PreviousState, changed = r.redactMap(entry.PreviousState, "", paths, changed)
	entry.CurrentState, changed = r.redactMap(entry.CurrentState, "", paths, changed)
	entry.Metadata, changed = r.redactMap(entry.Metadata, "", paths, changed)
	entry.Changes, changed = r.redactChanges(entry.Changes, paths, changed)
	if redacted, hit := redactSecretShapes(entry.Comment); hit {
		entry.Comment = redacted
		changed = true
	}
	if changed {
		entry.SensitiveData = true
	}
}

func (r *Redactor) redactMap(
	values map[string]any,
	prefix string,
	paths map[string]fieldAction,
	changed bool,
) (map[string]any, bool) {
	if values == nil {
		return nil, changed
	}
	for key, value := range values {
		path := joinPath(prefix, key)
		switch actionFor(key, path, paths) {
		case actionOmit:
			if value != nil {
				values[key] = RedactedValue
				changed = true
			}
			continue
		case actionFingerprint:
			if value != nil && value != "" {
				values[key] = r.fingerprint(value)
				changed = true
			}
			continue
		}
		var hit bool
		values[key], hit = r.redactValue(value, path, paths)
		changed = changed || hit
	}
	return values, changed
}

func (r *Redactor) redactValue(
	value any,
	path string,
	paths map[string]fieldAction,
) (any, bool) {
	switch v := value.(type) {
	case map[string]any:
		return r.redactMap(v, path, paths, false)
	case []any:
		changed := false
		for i, item := range v {
			var hit bool
			v[i], hit = r.redactValue(item, path, paths)
			changed = changed || hit
		}
		return v, changed
	case string:
		return redactSecretShapes(v)
	default:
		return value, false
	}
}

func (r *Redactor) redactChanges(
	changes map[string]any,
	paths map[string]fieldAction,
	changed bool,
) (map[string]any, bool) {
	for path, raw := range changes {
		change, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		normalized := normalizePath(path)
		leaf := normalized
		if idx := strings.LastIndex(normalized, "."); idx >= 0 {
			leaf = normalized[idx+1:]
		}
		action := actionFor(leaf, normalized, paths)
		for _, side := range []string{changeFrom, changeTo} {
			value, has := change[side]
			if !has || value == nil {
				continue
			}
			switch action {
			case actionOmit:
				change[side] = RedactedValue
				changed = true
			case actionFingerprint:
				if value != "" {
					change[side] = r.fingerprint(value)
					changed = true
				}
			default:
				var hit bool
				change[side], hit = r.redactValue(value, normalized, paths)
				changed = changed || hit
			}
		}
	}
	return changes, changed
}

func (r *Redactor) fingerprint(value any) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	runes := []rune(text)
	tail := string(runes[max(0, len(runes)-fingerprintTailRunes):])
	if len(runes) <= fingerprintTailRunes {
		tail = ""
	}
	if len(r.pepper) == 0 {
		return RedactedValue + " …" + tail
	}
	mac := hmac.New(sha256.New, r.pepper)
	mac.Write([]byte(text))
	digest := hex.EncodeToString(mac.Sum(nil))[:fingerprintHexLength]
	return RedactedValue + " …" + tail + " #" + digest
}

func redactSecretShapes(value string) (string, bool) {
	if value == "" {
		return value, false
	}
	changed := false
	for _, shape := range secretShapes {
		if shape.MatchString(value) {
			value = shape.ReplaceAllString(value, RedactedValue)
			changed = true
		}
	}
	return value, changed
}

func actionFor(key, path string, paths map[string]fieldAction) fieldAction {
	if action, ok := paths[path]; ok {
		return action
	}
	if action, ok := paths[key]; ok {
		return action
	}
	words := splitWords(key)
	if isSecretName(words) {
		return actionOmit
	}
	if _, ok := fingerprintedNames[strings.Join(words, "")]; ok {
		return actionFingerprint
	}
	return 0
}

func isSecretName(words []string) bool {
	if len(words) == 0 {
		return false
	}
	for _, word := range words {
		switch word {
		case "password", "passwd", "passphrase":
			return true
		}
	}
	last := words[len(words)-1]
	switch last {
	case "secret", "token":
		return true
	case "apikey", "privatekey":
		return true
	}
	if len(words) < 2 {
		return false
	}
	pair := words[len(words)-2] + " " + last
	switch pair {
	case "api key", "private key", "signing key", "secret key", "access key", "encryption key",
		"recovery code", "recovery codes", "key hash", "token hash", "secret hash":
		return true
	}
	return false
}

func splitWords(name string) []string {
	words := make([]string, 0, 4)
	var current strings.Builder
	runes := []rune(name)
	flush := func() {
		if current.Len() > 0 {
			words = append(words, strings.ToLower(current.String()))
			current.Reset()
		}
	}
	for i, ch := range runes {
		switch {
		case ch == '_' || ch == '-' || ch == ' ' || ch == '.':
			flush()
			continue
		case unicode.IsUpper(ch):
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			prevLower := i > 0 && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1]))
			if prevLower || (nextLower && i > 0 && unicode.IsUpper(runes[i-1])) {
				flush()
			}
		}
		current.WriteRune(ch)
	}
	flush()
	return words
}

func joinPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

func normalizePath(path string) string {
	parts := strings.Split(path, ".")
	kept := parts[:0]
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err == nil {
			continue
		}
		kept = append(kept, strings.TrimSuffix(part, "[]"))
	}
	return strings.Join(kept, ".")
}
