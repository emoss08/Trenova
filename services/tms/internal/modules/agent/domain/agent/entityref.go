package agent

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MaxMentions          = 10
	maxMentionLabelLenth = 200
)

// EntityRef is a record a person pointed at by name while asking: the kind,
// the id, and the label they saw. It is data about the record, never the
// record itself; the model reads it with the matching get tool.
type EntityRef struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Validate reports each reference's problems under prefix[i], so a client can
// show them beside the chip that carried them.
func ValidateEntityRefs(prefix string, refs []EntityRef, multiErr *errortypes.MultiError) {
	if len(refs) > MaxMentions {
		multiErr.Add(prefix, errortypes.ErrInvalid, "Too many mentioned records")
		return
	}

	seen := make(map[string]struct{}, len(refs))
	for i, ref := range refs {
		field := prefix + "[" + strconv.Itoa(i) + "]"
		if !IsKnownPageEntityType(ref.Type) {
			multiErr.Add(field+".type", errortypes.ErrInvalid, "Unknown record type")
		}
		if _, err := pulid.Parse(ref.ID); err != nil {
			multiErr.Add(field+".id", errortypes.ErrInvalid, "Record identifier is invalid")
		}
		if len(ref.Label) > maxMentionLabelLenth {
			multiErr.Add(field+".label", errortypes.ErrInvalid, "Label is too long")
		}
		key := ref.Type + ":" + ref.ID
		if _, dup := seen[key]; dup {
			multiErr.Add(field, errortypes.ErrInvalid, "Record is mentioned twice")
		}
		seen[key] = struct{}{}
	}
}

// NormalizeEntityRefs trims each reference and drops empty ones, so what is
// stored is what was validated.
func NormalizeEntityRefs(refs []EntityRef) []EntityRef {
	if len(refs) == 0 {
		return nil
	}

	out := make([]EntityRef, 0, len(refs))
	for _, ref := range refs {
		normalized := EntityRef{
			Type:  strings.TrimSpace(ref.Type),
			ID:    strings.TrimSpace(ref.ID),
			Label: strings.TrimSpace(ref.Label),
		}
		if normalized.Type == "" && normalized.ID == "" {
			continue
		}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}

	return out
}
