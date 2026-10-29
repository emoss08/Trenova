package artifactpayload

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/maputils"
)

func Rereadable(artifact *assistantartifact.Artifact) bool {
	if artifact.Kind == assistantartifact.KindReportPreview {
		return true
	}

	return artifact.Kind == assistantartifact.KindTableView &&
		assistantartifact.FamilyOf(artifact.Kind, artifact.Payload) == assistantartifact.FamilyTable
}

func NoteReadWith(artifact *assistantartifact.Artifact, arguments map[string]any) {
	if arguments != nil && Rereadable(artifact) {
		artifact.Payload[PayloadReadWith] = maputils.CloneDeep(arguments)
	}
}
