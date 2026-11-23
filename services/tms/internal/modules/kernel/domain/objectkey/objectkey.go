package objectkey

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/google/uuid"
)

const (
	organizationIDPrefix = "org_"
	maxExtension         = 12
)

type Family struct {
	Name    string
	Carried bool
	root    string
}

var (
	Organization   = Family{Name: "organization", Carried: true}
	Reports        = Family{Name: "reports", Carried: true, root: "reports/"}
	Capture        = Family{Name: "capture", Carried: true, root: "capture/"}
	AuditExports   = Family{Name: "audit-exports", root: "audit-exports/"}
	AIAuditExports = Family{Name: "ai-audit-exports", root: "ai-audit-exports/"}

	DeskTranscriptExports = Family{
		Name: "desk-transcript-exports",
		root: "desk-transcript-exports/",
	}
)

var families = []Family{
	Organization,
	Reports,
	Capture,
	AuditExports,
	AIAuditExports,
	DeskTranscriptExports,
}

func Families() []Family {
	return slices.Clone(families)
}

func (f Family) Prefix(organizationID pulid.ID) string {
	return f.root + organizationID.String() + "/"
}

func (f Family) Key(organizationID pulid.ID, parts ...string) string {
	return f.Prefix(organizationID) + strings.Join(parts, "/")
}

func (f Family) NewKey(organizationID pulid.ID, kind, fileName string) string {
	return f.Key(organizationID, kind, uuid.New().String()+extension(fileName))
}

func extension(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if len(ext) < 2 || len(ext) > maxExtension {
		return ""
	}
	for _, r := range ext[1:] {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return ""
		}
	}
	return ext
}

func Prefixes(organizationID pulid.ID) []string {
	out := make([]string, 0, len(families))
	for _, f := range families {
		out = append(out, f.Prefix(organizationID))
	}
	return out
}

func CarriedPrefixes(organizationID pulid.ID) []string {
	out := make([]string, 0, len(families))
	for _, f := range families {
		if f.Carried {
			out = append(out, f.Prefix(organizationID))
		}
	}
	return out
}

func Owner(key string) (pulid.ID, Family, bool) {
	for _, f := range families {
		if f.root == "" {
			continue
		}
		if rest, ok := strings.CutPrefix(key, f.root); ok {
			if org, owned := organizationOf(rest); owned {
				return org, f, true
			}
			return "", Family{}, false
		}
	}
	if org, ok := organizationOf(key); ok {
		return org, Organization, true
	}
	return "", Family{}, false
}

func OwnedBy(key string, organizationID pulid.ID) bool {
	if organizationID.IsNil() || !Clean(key) {
		return false
	}
	owner, _, ok := Owner(key)
	return ok && owner == organizationID
}

func CarriedBy(key string, organizationID pulid.ID) bool {
	if !OwnedBy(key, organizationID) {
		return false
	}
	_, family, _ := Owner(key)
	return family.Carried
}

func Rebase(key string, organizationID pulid.ID) (string, bool) {
	if !CarriedBy(key, organizationID) {
		return "", false
	}
	owner, family, _ := Owner(key)
	return family.Prefix(organizationID) + strings.TrimPrefix(key, family.Prefix(owner)), true
}

func Clean(key string) bool {
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "\\") ||
		strings.ContainsAny(key, "\x00\r\n") {
		return false
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func organizationOf(rest string) (pulid.ID, bool) {
	head, tail, ok := strings.Cut(rest, "/")
	if !ok || tail == "" || !strings.HasPrefix(head, organizationIDPrefix) ||
		len(head) <= len(organizationIDPrefix) {
		return "", false
	}
	return pulid.ID(head), true
}
