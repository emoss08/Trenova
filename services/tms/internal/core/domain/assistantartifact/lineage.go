package assistantartifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

var versionedKinds = map[Kind]bool{
	KindTableView:       true,
	KindEntityCard:      true,
	KindReportPreview:   true,
	KindRateExplanation: true,
	KindRunDiff:         true,
}

func LineageKeyFor(kind Kind, toolName string, arguments map[string]any) string {
	if !versionedKinds[kind] || toolName == "" {
		return ""
	}
	canonical, err := json.Marshal(arguments)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256([]byte(string(kind) + "\x00" + toolName + "\x00" + string(canonical)))

	return hex.EncodeToString(sum[:])
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
