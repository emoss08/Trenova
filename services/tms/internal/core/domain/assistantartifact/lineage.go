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
