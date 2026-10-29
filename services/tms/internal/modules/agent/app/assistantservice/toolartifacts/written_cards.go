package toolartifacts

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	PayloadWritten    = "written"
	DecisionCallID    = "decision:"
	writtenChangeVerb = "change "
	WrittenIDField    = "id"
	WrittenNameField  = "name"
)

func writtenPhrase(artifact *assistantartifact.Artifact) string {
	action := typeutils.StringOfTrimmed(artifact.Payload[PayloadWritten])
	if action == "" {
		return ""
	}

	return writtenChangeVerb + action
}
