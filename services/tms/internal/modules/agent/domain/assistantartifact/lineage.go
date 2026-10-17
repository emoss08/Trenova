package assistantartifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

var versionedKinds = map[Kind]bool{
	KindTableView:          true,
	KindEntityCard:         true,
	KindReportPreview:      true,
	KindRateExplanation:    true,
	KindRunDiff:            true,
	KindExtraction:         true,
	KindEDIConnectionDraft: true,
}

// hasLineage says a kind keeps versions: every lookup's view, and an analysis,
// which is made on purpose but run again under its title is the same result,
// revised.
func hasLineage(kind Kind) bool {
	return versionedKinds[kind] || kind == KindAnalysis
}

// LineageKeyFor names what an artifact is a view of, so reading the same
// thing again makes a new version of it rather than a second artifact. What
// it is about is the tool and the records it names (an id, the entity a
// view is composed over); how it was asked (filters, sorting, paging) is not.
// The billing queue read for items in review and read again for every item is
// one table seen twice; a card of shipment A and a card of shipment B are two.
func LineageKeyFor(kind Kind, toolName string, arguments map[string]any) string {
	if !hasLineage(kind) || toolName == "" {
		return ""
	}
	identity := identityOf(kind, arguments)
	canonical, err := json.Marshal(identity)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256([]byte(string(kind) + "\x00" + toolName + "\x00" + string(canonical)))

	return hex.EncodeToString(sum[:])
}

//nolint:exhaustive // only a kind made from one record is named by one argument
var namedBy = map[Kind]string{
	KindEDIConnectionDraft: "documentId",
}

// identityOf keeps the arguments that name what a call is about: the
// entity a view is over, and every id, at the top level or one level down.
func identityOf(kind Kind, arguments map[string]any) map[string]any {
	if kind == KindAnalysis {
		return map[string]any{"title": analysisIdentity(arguments)}
	}
	if key, ok := namedBy[kind]; ok {
		return map[string]any{key: arguments[key]}
	}
	identity := map[string]any{}
	for key, value := range arguments {
		switch {
		case key == "entity":
			identity[key] = value
		case namesRecord(key):
			identity[key] = value
		default:
			if nested, ok := value.(map[string]any); ok {
				for innerKey, innerValue := range nested {
					if namesRecord(innerKey) {
						identity[key+"."+innerKey] = innerValue
					}
				}
			}
		}
	}

	return identity
}

func namesRecord(key string) bool {
	return key == "id" || strings.HasSuffix(key, "Id") || strings.HasSuffix(key, "ID") ||
		strings.HasSuffix(key, "Ids")
}

func (a *Artifact) FollowLineage(previous *Artifact) {
	if previous == nil {
		return
	}
	a.LineageID = previous.LineageID
	if a.LineageID.IsNil() {
		a.LineageID = previous.ID
	}
	a.LineageSeq = max(previous.LineageSeq, 1) + 1
}

// IsLookup says an artifact is a view of what a lookup returned (a table, a
// record card, a report preview) rather than something made on purpose (a
// document, a draft, a plan). A lookup's view is kept only when the reply
// points to it.
func IsLookup(kind Kind) bool {
	return versionedKinds[kind]
}

// analysisIdentity is what an analysis is a view of: its title, as written
// in any case and spacing, or its purpose when it has no title. Running it
// again under the same title, after a fix or over newer data, is the next
// version of one result rather than a second.
func analysisIdentity(arguments map[string]any) string {
	for _, key := range []string{"title", "purpose"} {
		if text, ok := arguments[key].(string); ok {
			if name := strings.ToLower(strings.Join(strings.Fields(text), " ")); name != "" {
				return name
			}
		}
	}

	return ""
}
