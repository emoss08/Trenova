package agent

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxResultWordChars = 40
	maxResultNameChars = 200
	maxResultIDs       = 8
	maxResultKeyChars  = 64
	maxResultIDChars   = 100
)

// ToolExecutionResult is what an executed write made, kept on the proposal
// that ran it.
//
// Approving a proposal used to leave only "it ran" behind, so the model
// reached for the one id it held, the proposal's, and passed an ap_ id to a
// tool that wanted the report it had just saved. The result names the record
// and its id under the parameter the next tool takes it as. It is a pointer
// to the record, never the record: a tool's full output does not belong on the
// proposal, and Bounded keeps it to about a kilobyte whatever a tool reports.
type ToolExecutionResult struct {
	// Action is what the write did to the record, as a past-tense verb:
	// "created", "updated".
	Action string `json:"action"`
	// Kind is the kind of record in words a person uses: "report".
	Kind string `json:"kind,omitempty"`
	// Name is the record's own name, as a person would recognize it.
	Name string `json:"name,omitempty"`
	// IDs holds each id by the parameter other tools take it as, such as
	// {"definitionId": "rd_…"}.
	IDs map[string]string `json:"ids,omitempty"`
	// Record names the one record the write made or changed, when there is
	// exactly one, by the record-link registry's entity and its id, so a
	// reader links to it without guessing from Kind and IDs. Kind, Name and
	// IDs stay for the model.
	Record *RecordRef `json:"record,omitempty"`
}

// RecordRef points at one record the way the app opens it: EntityType is a
// key of the record-link registry (client/apps/web/src/config/record-links.ts,
// served as productguide.Default.Record), such as "report".
type RecordRef struct {
	EntityType string `json:"entityType"`
	ID         string `json:"id"`
}

// Bounded is the reference kept on a result: an entity key of lower-case
// letters, digits and underscores, as the registry writes them, and an id on
// one line. A reference that names no entity or no id is nil.
func (r *RecordRef) Bounded() *RecordRef {
	if r == nil {
		return nil
	}

	entity := strings.TrimSpace(r.EntityType)
	id := stringutils.OneLine(r.ID, maxResultIDChars)
	if entity == "" || id == "" || len(entity) > maxResultKeyChars ||
		!isRecordEntityKey(entity) {
		return nil
	}

	return &RecordRef{EntityType: entity, ID: id}
}

func isRecordEntityKey(value string) bool {
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9', char == '_':
		default:
			return false
		}
	}

	return true
}

// Bounded is the result cut to what a proposal keeps: single-line words
// and names, and the first few ids in key order. Nil stays nil, and a
// result that says nothing is nil.
func (r *ToolExecutionResult) Bounded() *ToolExecutionResult {
	if r == nil {
		return nil
	}

	bounded := &ToolExecutionResult{
		Action: stringutils.OneLine(r.Action, maxResultWordChars),
		Kind:   stringutils.OneLine(r.Kind, maxResultWordChars),
		Name:   stringutils.OneLine(r.Name, maxResultNameChars),
		Record: r.Record.Bounded(),
	}

	keys := make([]string, 0, len(r.IDs))
	for key, id := range r.IDs {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(id) != "" {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	if len(keys) > maxResultIDs {
		keys = keys[:maxResultIDs]
	}
	if len(keys) > 0 {
		bounded.IDs = make(map[string]string, len(keys))
		for _, key := range keys {
			bounded.IDs[stringutils.OneLine(key, maxResultKeyChars)] = stringutils.OneLine(
				r.IDs[key],
				maxResultIDChars,
			)
		}
	}

	if bounded.Action == "" && bounded.Kind == "" && bounded.Name == "" &&
		len(bounded.IDs) == 0 && bounded.Record == nil {
		return nil
	}

	return bounded
}

// Describe says in one sentence, for a person, what the write made:
// `It created the report "Shipments for Peak Distributing".` The name is
// quoted as written, not escaped, because a person reads it. It is empty
// when the result names neither a kind nor a name.
func (r *ToolExecutionResult) Describe() string {
	if r == nil || (r.Kind == "" && r.Name == "") {
		return ""
	}

	action := r.Action
	if action == "" {
		action = "made"
	}

	switch {
	case r.Kind != "" && r.Name != "":
		return "It " + action + " the " + r.Kind + ` "` + r.Name + `".`
	case r.Kind != "":
		return "It " + action + " the " + r.Kind + "."
	default:
		return "It " + action + ` "` + r.Name + `".`
	}
}

// IDKeys are the parameter names the write's ids are held under, in order,
// so everything that lists them lists them the same way.
func (r *ToolExecutionResult) IDKeys() []string {
	if r == nil || len(r.IDs) == 0 {
		return nil
	}

	keys := make([]string, 0, len(r.IDs))
	for key := range r.IDs {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	return keys
}

// References lists the ids the write produced, key and value, in key
// order: "definitionId rd_…". Empty when it produced none.
func (r *ToolExecutionResult) References() string {
	keys := r.IDKeys()
	if len(keys) == 0 {
		return ""
	}

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+" "+r.IDs[key])
	}

	return strings.Join(parts, ", ")
}

// IDNote tells the model which ids the write produced and that they, not
// the proposal's id, name what it made. Empty when it produced none.
func (r *ToolExecutionResult) IDNote() string {
	references := r.References()
	if references == "" {
		return ""
	}

	those := "that id"
	if len(r.IDs) > 1 {
		those = "those ids"
	}

	return fmt.Sprintf(
		"It produced %s; use %s, not the proposal's, when you refer to what it made.",
		references, those,
	)
}
