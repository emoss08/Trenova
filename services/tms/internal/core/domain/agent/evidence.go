package agent

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
)

type EvidenceRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Note string `json:"note,omitempty"`
}

func validateEvidence(field string, evidence []EvidenceRef, multiErr *errortypes.MultiError) {
	if len(evidence) == 0 {
		multiErr.Add(
			field,
			errortypes.ErrRequired,
			"At least one evidence reference is required",
		)
		return
	}

	for i, ref := range evidence {
		if strings.TrimSpace(ref.Type) == "" {
			multiErr.Add(
				fmt.Sprintf("%s[%d].type", field, i),
				errortypes.ErrRequired,
				"Evidence type is required",
			)
		}
		if strings.TrimSpace(ref.ID) == "" {
			multiErr.Add(
				fmt.Sprintf("%s[%d].id", field, i),
				errortypes.ErrRequired,
				"Evidence id is required",
			)
		}
	}
}
